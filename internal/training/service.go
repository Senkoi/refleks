package training

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"refleks/internal/models"
)

type Service struct {
	mu              sync.Mutex
	dataRevision    uint64
	evaluationKey   string
	evaluatedAt     time.Time
	checkpointAt    time.Time
	path            string
	state           State
	discovering     bool
	pendingReminder string
	localScanMu     sync.Mutex
	localFiles      map[string]string
	localAttempts   map[string]time.Time
	localParsed     map[string]localParsedSCE
	localDirty      bool
	sessionGap      time.Duration
}

// TakeReminder returns each cue once to the desktop notification layer.
func (s *Service) TakeReminder() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	message := s.pendingReminder
	s.pendingReminder = ""
	return message
}

func New(dir string) (*Service, error) {
	s := &Service{path: filepath.Join(dir, "adaptive-training.json"), state: State{Version: 1, Catalog: []Scenario{}, History: []Plan{}, Skills: []SkillStatus{}, Preferences: defaults(), Discovery: Discovery{Candidates: []Candidate{}, Warnings: []string{}}}}
	b, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(b, &s.state); err != nil {
			return nil, fmt.Errorf("训练数据损坏，未覆盖原文件：%w", err)
		}
		if s.state.Version != 1 {
			return nil, fmt.Errorf("不支持的训练数据版本")
		}
		// Keep version 1 readable: the new multi-system field extends the JSON
		// schema without dropping its original benchmark and score cutoffs.
		for i := range s.state.Catalog {
			item := &s.state.Catalog[i]
			item.Benchmarks = mergeMemberships(nil, memberships(*item))
			for j := range item.Benchmarks {
				item.Benchmarks[j] = normalizeMembership(item.Benchmarks[j])
			}
			calculateFileEvidence(item.LocalAssessment)
			*item = enrichMechanics(*item)
			if item.DifficultySource == "playlist" && item.Classification != "manual" {
				item.Difficulty, item.DifficultySource = "unknown", "unknown"
			}
		}
		if s.state.Preferences.ExecutionMode == "" {
			s.state.Preferences.ExecutionMode = "playlist"
		}
	}
	s.state.Initializing = false
	if s.state.Error == "应用重新启动，训练已暂停；请确认后继续。" {
		s.state.Error = ""
		s.state.Notice = "应用重新启动，训练已暂停；可继续或结束本次。"
	}
	if p := s.state.Plan; p != nil && (p.Status == "running" || p.Status == "ready" || p.Status == "waiting") {
		p.Status = "paused"
		p.LastTick = 0
		s.state.Notice = "应用重新启动，训练已暂停；可继续或结束本次。"
	}
	return s, nil
}

func (s *Service) save() error {
	s.dataRevision++
	s.evaluationKey = ""
	return s.persist()
}

func (s *Service) persist() error {
	s.syncProgressLocked()
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err = os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Service) Snapshot(runs []models.RunRecord) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluateLocked(runs, time.Now())
	b, _ := json.Marshal(s.state)
	var copy State
	_ = json.Unmarshal(b, &copy)
	return copy
}

func (s *Service) Import(data []byte, src Source) (int, error) {
	items, err := ParsePlaylist(data, src)
	if err != nil {
		return 0, err
	}
	return s.Add(items)
}

func (s *Service) Add(items []Scenario) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.state.Catalog)
	s.state.Catalog = mergeCatalog(s.state.Catalog, items)
	s.mergeCurricula(items)
	return len(s.state.Catalog) - n, s.save()
}

func (s *Service) ImportURL(ctx context.Context, raw string) (int, error) {
	items, c, err := readSource(ctx, raw, raw)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		s.mu.Lock()
		s.state.Discovery.Candidates = append(s.state.Discovery.Candidates, c)
		err = s.save()
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("已保存页面线索；未取得结构化关卡列表，请导入其 playlist JSON")
	}
	return s.Add(items)
}

func (s *Service) Discover(ctx context.Context) (Discovery, error) {
	s.mu.Lock()
	if s.discovering {
		s.mu.Unlock()
		return Discovery{}, fmt.Errorf("关卡发现正在进行")
	}
	s.discovering = true
	focus := s.state.Preferences.Focus
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.discovering = false; s.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(ctx, 110*time.Second)
	defer cancel()
	items, d := discover(ctx, focus)
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.state.Catalog)
	s.state.Catalog = mergeCatalog(s.state.Catalog, items)
	s.mergeCurricula(items)
	d.Imported = len(s.state.Catalog) - before
	s.state.Discovery = d
	return d, s.save()
}

func (s *Service) DiscoveryDue(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Preferences.AutoDiscover || s.discovering {
		return false
	}
	t, err := time.Parse(time.RFC3339, s.state.Discovery.Updated)
	return err != nil || now.Sub(t) > 7*24*time.Hour
}

func (s *Service) UpdateScenario(item Scenario) error {
	if !validSkill(item.Skill) && item.Skill != "unknown" {
		return fmt.Errorf("无效分类")
	}
	if item.Seconds < 10 || item.Seconds > 3600 {
		return fmt.Errorf("关卡时长需为 10–3600 秒")
	}
	if item.Difficulty != "unknown" && item.Difficulty != "novice" && item.Difficulty != "intermediate" && item.Difficulty != "advanced" {
		return fmt.Errorf("无效难度")
	}
	if item.Preference != "" && item.Preference != "liked" && item.Preference != "neutral" && item.Preference != "disliked" {
		return fmt.Errorf("无效喜好反馈")
	}
	if item.PersonalDifficulty != "" && item.PersonalDifficulty != "easy" && item.PersonalDifficulty != "suitable" && item.PersonalDifficulty != "hard" {
		return fmt.Errorf("无效体感难度")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.VariantOf != "" {
		found := false
		for _, s := range s.state.Catalog {
			if strings.EqualFold(s.Name, strings.TrimSpace(item.VariantOf)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("原版关卡尚不在关卡库中")
		}
	}
	for i := range s.state.Catalog {
		if s.state.Catalog[i].Name == item.Name {
			if item.VariantOf != "" && strings.EqualFold(item.VariantOf, item.Name) {
				return fmt.Errorf("变体不能指向自身")
			}
			for _, name := range item.RelatedBenchmarks {
				if len(name) > 200 {
					return fmt.Errorf("关联 benchmark 名称过长")
				}
			}
			e := &s.state.Catalog[i]
			e.Skill = item.Skill
			e.Family = item.Family
			e.Difficulty = item.Difficulty
			e.DifficultySource = "manual"
			e.Technique = technique(item.Name, item.Skill)
			e.Seconds = item.Seconds
			e.VariantOf = strings.TrimSpace(item.VariantOf)
			e.RelatedBenchmarks = mergeStrings(nil, item.RelatedBenchmarks)
			e.Preference = item.Preference
			e.PersonalDifficulty = item.PersonalDifficulty
			e.Enabled = item.Enabled && validSkill(item.Skill)
			e.Classification = "manual"
			return s.save()
		}
	}
	return fmt.Errorf("关卡不存在")
}

func legacyFixedSets(p Plan) bool {
	if p.PlannerVersion >= 6 {
		return false
	}
	for _, b := range p.Blocks {
		if b.SourcePlayCount == 0 && (b.PlayCount >= 5 || b.PlayCount == 0 && b.Budget == 300 && b.Scenario.Seconds <= 90) {
			return true
		}
	}
	return false
}

// BeginStartup hides persisted drafts until benchmark caches and history are ready.
func (s *Service) BeginStartup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = true
}

// PrepareStartup rebuilds inactive lists with current history and policy before
// exposing them. Current-policy sessions remain immutable; legacy fixed
// sets are archived once at restart.
func (s *Service) PrepareStartup(runs []models.RunRecord) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = true
	migrated := false
	if old := s.state.Plan; old != nil && old.Status == "paused" && legacyFixedSets(*old) {
		// Replace the legacy fixed five-minute startup policy.
		// Preserve every recorded score/progress as ended history; pending
		// rows cannot masquerade as a completed curriculum baseline.
		archived := *old
		archived.Blocks = append([]Block(nil), old.Blocks...)
		archived.Status, archived.EndedAt = "completed", time.Now().UnixMilli()
		for i := range archived.Blocks {
			if archived.Blocks[i].Outcome == "pending" || archived.Blocks[i].Outcome == "" {
				archived.Blocks[i].Outcome = "skipped"
			}
		}
		s.state.Plan = &archived
		migrated = true
	}
	if old := s.state.Plan; old != nil && (old.Status == "running" || old.Status == "ready" || old.Status == "paused" || old.Status == "waiting") {
		s.state.Initializing = false
		return false, nil
	}
	p := automaticTrainingPreferences(s.state.Preferences)
	if old := s.state.Plan; old != nil && old.PlannerVersion >= 7 && old.Status == "draft" {
		s.state.Initializing = false
		return false, nil
	}
	if err := validatePreferences(p); err != nil {
		p = defaults()
	}
	if _, err := s.generateLocked(p, runs); err != nil {
		s.state.Error = "启动训练列表生成失败：" + err.Error()
		s.state.Initializing = false
		return false, err
	}
	s.state.Notice = "已根据已有 benchmark 成绩与训练历史生成本次列表。"
	if migrated {
		s.state.Notice = "旧版固定五分钟的暂停列表已归档，记录已保留；本次按最新档位与短组策略重新生成。"
	}
	return true, s.save()
}

// FinishStartup releases the UI only after the owned playlist installation.
func (s *Service) FinishStartup(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = false
	if err != nil {
		s.state.Error = err.Error()
	}
}

func (s *Service) StartupFailed(err error) { s.FinishStartup(err) }

func (s *Service) Generate(p Preferences, runs []models.RunRecord) (*Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Initializing {
		return nil, fmt.Errorf("正在读取 benchmark 成绩与训练历史，请稍候")
	}
	return s.generateLocked(p, runs)
}

func (s *Service) generateLocked(p Preferences, runs []models.RunRecord) (*Plan, error) {
	if old := s.state.Plan; old != nil && (old.Status == "running" || old.Status == "ready" || old.Status == "paused" || old.Status == "waiting") {
		return nil, fmt.Errorf("请先结束当前计划，再生成新计划")
	}
	now := time.Now()
	s.syncSessionContextsLocked(runs, now)
	s.updateStudiesLocked(now)
	s.updateAnchorEvaluationsLocked(runs, now)
	s.auditTransferExposureLocked(runs, now)
	rng := rand.New(rand.NewSource(now.UnixNano()))
	if p.PlanningPolicy != "legacy" {
		p = automaticTrainingPreferences(p)
	}
	planningPreferences := automaticReferences(s.state.Catalog, p)
	var plan *Plan
	var err error
	if p.PlanningPolicy == "legacy" {
		plan, err = Generate(s.state.Catalog, runs, planningPreferences, now, rng)
	} else {
		p.PlanningPolicy = "curriculum"
		var t *Curriculum
		history := append([]Plan(nil), s.state.History...)
		if old := s.state.Plan; old != nil {
			history = append(history, *old)
		}
		s.syncProgressLocked()
		t, err = selectCurriculumWithHistory(s.state.Curricula, s.state.Catalog, p, runs, now, history, s.state.CurriculumProgress)
		if err == nil {
			progress := progressFor(*t, history, s.state.CurriculumProgress)
			explored := progress.EverCompleted || curriculumBaseline(*t, history)
			firstSeconds := 0
			for i, row := range t.Rows {
				if !progress.complete() && i < len(progress.Rows) && progress.Rows[i].Target > 0 && progress.Rows[i].Completed >= progress.Rows[i].Target {
					continue
				}
				for _, c := range s.state.Catalog {
					if strings.EqualFold(row.Name, c.Name) {
						firstSeconds = estimateTiming(c, observed(runs)[strings.ToLower(c.Name)], now).Seconds
						break
					}
				}
				break
			}
			bundle := preparePersonalization(*t, s.state.Catalog, runs, s.state.RunContexts, s.state.TrainingStudies, now, p, explored, firstSeconds, rng)
			window, start, end, windowErr := curriculumWindowWithReserve(*t, s.state.Catalog, runs, p, now, history, progress, bundle.seconds(), bundle.seconds() == 0, s.state.Curricula...)
			if windowErr != nil {
				return nil, windowErr
			}
			plan, err = GenerateCurriculum(window, s.state.Catalog, runs, planningPreferences, now, rng, explored && bundle.seconds() == 0, s.state.Curricula...)
			if err == nil {
				plan.Blocks = append(append(bundle.before, plan.Blocks...), bundle.after...)
				if !s.scheduleAnchorEvaluationLocked(plan, runs, now) {
					requests := s.assessmentRequests(plan, runs, now)
					if len(requests) > 0 && requests[0].seconds > 0 {
						// Reserve only a real eligible assessment. Keep the original
						// order and leave the shortened tail for ordinary continuation.
						w, a, z, e := curriculumWindowWithReserve(*t, s.state.Catalog, runs, p, now, history, progress, bundle.seconds()+requests[0].seconds, false, s.state.Curricula...)
						if e == nil {
							candidate, e := GenerateCurriculum(w, s.state.Catalog, runs, planningPreferences, now, rng, false, s.state.Curricula...)
							if e == nil {
								candidate.ID = plan.ID
								candidate.Blocks = append(append(bundle.before, candidate.Blocks...), bundle.after...)
								if s.scheduleAnchorEvaluationLocked(candidate, runs, now) {
									plan, start, end = candidate, a, z
								}
							}
						}
					}
				}
				if bundle.study != nil {
					bundle.study.PlanID = plan.ID
					s.state.TrainingStudies = append(s.state.TrainingStudies, *bundle.study)
				}
				anchors := personalAnchors(s.state.Catalog, runs, s.state.RunContexts, now)
				for i := range plan.Blocks {
					b := &plan.Blocks[i]
					a, ok := anchors[strings.ToLower(b.Scenario.Name)]
					if ok && b.Personalization == nil && b.Measurement == nil {
						b.Personalization = &SceneDecision{Source: "direct_history", Anchor: a}
						if a.Status == "stable" && b.Role == "practice" {
							b.Target = a.MedianScore + max(a.ScoreMAD, a.MedianScore*.02)
							b.Signature = signatureForPersonalTarget(b.Scenario.Name, runs, now)
							b.Reason += " 个人目标取近期训练块中位成绩加波动/小幅进步目标；固定局数不变。"
						}
					}
				}
				plan.CurriculumCycle = progress.Cycle
				if progress.complete() {
					plan.CurriculumCycle++
				}
				if resumeDue(progress, now) {
					plan.SelectionReason = "24 小时内优先续接：跳过已完成行，部分完成的行只安排剩余次数。"
				} else {
					plan.SelectionReason = ThemePriorities(s.state.Catalog, runs, now)[t.Theme].Evidence
				}
				plan.CurriculumStart = start
				plan.CurriculumEnd = end
				plan.CurriculumTotal = len(t.Rows)
				if start != 0 || end != len(t.Rows) {
					plan.Warnings = append(plan.Warnings, fmt.Sprintf("按原顺序运行 VDIM 第 %d–%d / %d 行；本次使用自适应短组，完成后下一次接续后续段落。", start+1, end, len(t.Rows)))
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if old := s.state.Plan; old != nil && old.Status == "draft" {
		for i := range s.state.TrainingStudies {
			st := &s.state.TrainingStudies[i]
			if st.PlanID == old.ID && st.Status == "planned" {
				st.Status = "not_started"
			}
		}
		for i := range s.state.AnchorEvaluations {
			e := &s.state.AnchorEvaluations[i]
			if e.PlanID == old.ID && e.Status == "planned" {
				e.Status = "not_started"
			}
		}
	}
	if old := s.state.Plan; old != nil && old.Status != "draft" {
		s.state.History = append(s.state.History, *old)
		if len(s.state.History) > 100 {
			s.state.History = s.state.History[len(s.state.History)-100:]
		}
	}
	s.state.Preferences = p
	s.state.Plan = plan
	s.state.Error = ""
	s.state.Notice = ""
	if err = s.save(); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(plan)
	var copy Plan
	_ = json.Unmarshal(b, &copy)
	return &copy, nil
}

// Advance returns a scenario to launch. The caller owns Steam integration.
func (s *Service) Action(action string, now time.Time, runs []models.RunRecord) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.state.Plan
	if p == nil {
		return "", fmt.Errorf("请先生成计划")
	}
	if action == "pause" || action == "finish" {
		// The watcher can write a finished run before the next two-second poll.
		// Ingest it before changing state, otherwise a manual pause/finish loses it.
		s.tickLocked(now, runs)
	}
	launch := ""
	switch action {
	case "start":
		s.clock(now)
		s.expire()
		if p.Status != "draft" && p.Status != "ready" && p.Status != "paused" {
			return "", fmt.Errorf("当前状态不能开始")
		}
		if p.Elapsed >= float64(p.Preferences.Minutes*60) || p.Index >= len(p.Blocks) {
			return "", fmt.Errorf("计划已结束")
		}
		if p.Status == "draft" {
			p.Seen = []string{}
			for _, r := range runs {
				p.Seen = append(p.Seen, runKey(r))
			}
		}
		// Ignore any records that finished while paused or before this block began.
		p.AcceptAfter = now.UnixMilli()
		p.LastTick = now.UnixMilli()
		p.Status = "running"
		s.state.Error = ""
		if p.Preferences.ExecutionMode != "playlist" {
			launch = p.Blocks[p.Index].Scenario.Name
		}
	case "pause":
		if p.Status == "running" || p.Status == "ready" || p.Status == "waiting" {
			s.clock(now)
			s.expire()
			if p.Status != "completed" {
				p.Status = "paused"
			}
			p.LastTick = 0
		}
	case "next":
		if p.Status != "running" && p.Status != "paused" && p.Status != "waiting" {
			return "", fmt.Errorf("当前状态不能跳过")
		}
		s.clock(now)
		s.expire()
		paused := p.Status == "paused"
		if p.Status != "completed" {
			s.advance("skipped", now)
			if p.Status == "ready" && p.Preferences.ExecutionMode == "playlist" {
				p.Status = "running"
				if paused {
					p.Status = "paused"
				}
			}
		}
	case "finish":
		s.clock(now)
		if p.Status != "completed" {
			p.Status = "completed"
			p.EndedAt = now.UnixMilli()
			p.LastTick = 0
		}
	default:
		return "", fmt.Errorf("未知操作")
	}
	return launch, s.save()
}

func (s *Service) clock(now time.Time) {
	p := s.state.Plan
	if p == nil || (p.Status != "running" && p.Status != "ready" && p.Status != "waiting") {
		return
	}
	if p.LastTick > 0 {
		dt := math.Max(0, float64(now.UnixMilli()-p.LastTick)/1000)
		p.Elapsed += dt
		if p.Status == "running" || p.Status == "waiting" {
			p.BlockElapsed += dt
		}
	}
	p.LastTick = now.UnixMilli()
}

func (s *Service) expire() {
	p := s.state.Plan
	if p != nil && p.Elapsed >= float64(p.Preferences.Minutes*60) {
		if !p.RemindedEnd {
			p.RemindedEnd = true
			p.Reminder = "训练总时长已到。完成当前局后，请在游戏中结束列表并休息。"
			s.pendingReminder = p.Reminder
		}
		if p.EndedAt == 0 {
			p.EndedAt = p.LastTick - int64((p.Elapsed-float64(p.Preferences.Minutes*60))*1000)
		}
		p.Status = "completed"
		p.LastTick = 0
		if p.Index < len(p.Blocks) && p.Blocks[p.Index].Outcome == "pending" {
			p.Blocks[p.Index].Outcome = "session_limit"
		}
	}
}

func (s *Service) advance(outcome string, now time.Time) {
	p := s.state.Plan
	p.Blocks[p.Index].Outcome = outcome
	p.Reminder = ""
	p.Index++
	p.BlockElapsed = 0
	p.AcceptAfter = now.UnixMilli()
	p.Status = "ready"
	if p.Index >= len(p.Blocks) {
		p.Status = "completed"
		p.LastTick = 0
	}
}

// Tick consumes each completed run once, in chronological order, even when the
// Training page is closed. It never kills a live run at a time limit.
func (s *Service) Tick(now time.Time, runs []models.RunRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tickLocked(now, runs)
}

func (s *Service) tickLocked(now time.Time, runs []models.RunRecord) string {
	p := s.state.Plan
	// File ingestion may lag the cap by more than one polling interval. Accept
	// only runs that actually ended before the cap, for a bounded grace period.
	if p != nil && p.Status == "completed" && p.EndedAt > 0 && now.UnixMilli() <= p.EndedAt+120000 && p.Index < len(p.Blocks) {
		previousRecorded := p.Recorded
		seen := map[string]bool{}
		for _, k := range p.Seen {
			seen[k] = true
		}
		ordered := append([]models.RunRecord{}, runs...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Stats.Summary.DatePlayed < ordered[j].Stats.Summary.DatePlayed })
		for _, r := range ordered {
			if seen[runKey(r)] {
				continue
			}
			seen[runKey(r)] = true
			p.Seen = append(p.Seen, runKey(r))
			v := r.Stats.Summary
			end, err := time.Parse(time.RFC3339, v.DatePlayed)
			if err != nil || v.TimeRemaining > 1 || v.Duration <= 0 || math.IsNaN(v.Duration) || math.IsInf(v.Duration, 0) || v.Score < 0 || math.IsNaN(v.Score) || math.IsInf(v.Score, 0) {
				continue
			}
			start := end.Add(-time.Duration(v.Duration * float64(time.Second)))
			b := &p.Blocks[p.Index]
			if !strings.EqualFold(v.Scenario, b.Scenario.Name) || start.UnixMilli() < p.AcceptAfter-2000 || end.UnixMilli() > p.EndedAt+1000 || end.After(now.Add(2*time.Second)) {
				continue
			}
			b.Recorded += v.Duration
			p.Recorded += v.Duration
			b.Runs++
			b.LastCompletedAt = end.UnixMilli()
			s.capturePractice(b, r, now)
			b.Best = math.Max(b.Best, v.Score)
		}
		if p.Recorded != previousRecorded {
			s.syncSessionContextsLocked(runs, now)
			s.updateAnchorEvaluationsLocked(runs, now)
		}
		if err := s.checkpoint(now, p.Recorded != previousRecorded); err != nil {
			s.state.Error = "保存训练进度失败：" + err.Error()
		}
		return ""
	}
	if p == nil || (p.Status != "running" && p.Status != "ready" && p.Status != "waiting") {
		return ""
	}
	previousRecorded, previousStatus, previousIndex := p.Recorded, p.Status, p.Index
	s.clock(now)
	launch := ""
	if p.Status == "running" || p.Status == "waiting" {
		seen := map[string]bool{}
		for _, id := range p.Seen {
			seen[id] = true
		}
		ordered := append([]models.RunRecord{}, runs...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Stats.Summary.DatePlayed < ordered[j].Stats.Summary.DatePlayed })
		for _, r := range ordered {
			if p.Status != "running" && p.Status != "waiting" {
				break
			}
			if seen[runKey(r)] {
				continue
			}
			seen[runKey(r)] = true
			p.Seen = append(p.Seen, runKey(r))
			sum := r.Stats.Summary
			end, err := time.Parse(time.RFC3339, sum.DatePlayed)
			if err != nil {
				continue
			}
			start := end.Add(-time.Duration(sum.Duration * float64(time.Second)))
			if start.UnixMilli() < p.AcceptAfter-2000 || end.After(now.Add(2*time.Second)) {
				continue
			}
			if sum.TimeRemaining > 1 || sum.Duration <= 0 || math.IsNaN(sum.Duration) || math.IsInf(sum.Duration, 0) || sum.Score < 0 || math.IsNaN(sum.Score) || math.IsInf(sum.Score, 0) {
				b := &p.Blocks[p.Index]
				if p.PlannerVersion >= 8 && strings.EqualFold(sum.Scenario, b.Scenario.Name) && b.Runs < 4 {
					b.ObservationInterrupted = true
				}
				continue
			}
			if p.Preferences.ExecutionMode == "playlist" && p.Index+1 < len(p.Blocks) {
				current, next := p.Blocks[p.Index], p.Blocks[p.Index+1]
				// Same-name adjacent rows provide no name-change signal. Use the
				// exported repetition boundary; different-name rows still wait for
				// a completed run in the next scenario, allowing extra practice.
				if current.Runs >= max(1, current.PlayCount) && strings.EqualFold(current.Scenario.Name, next.Scenario.Name) && strings.EqualFold(sum.Scenario, next.Scenario.Name) {
					s.advance("list_complete", start)
					p.BlockElapsed = math.Min(p.Elapsed, math.Max(0, now.Sub(start).Seconds()))
					p.Status = "running"
				}
			}
			if p.Preferences.ExecutionMode == "playlist" && !strings.EqualFold(p.Blocks[p.Index].Scenario.Name, sum.Scenario) {
				// A finished run in a later playlist row is stronger evidence than
				// the generated play counts. Keep the local plan aligned if the
				// player skipped a row or manually advanced in-game.
				future := -1
				for i := p.Index + 1; i < len(p.Blocks); i++ {
					if strings.EqualFold(p.Blocks[i].Scenario.Name, sum.Scenario) {
						future = i
						break
					}
				}
				if future >= 0 {
					for p.Index < future {
						outcome := "missed"
						if p.Blocks[p.Index].Outcome == "list_complete" {
							outcome = "list_complete"
						}
						s.advance(outcome, start)
					}
					// Backdate to this completed run's start, not its file-arrival
					// time. The first run of the next scenario already used time.
					p.BlockElapsed = math.Min(p.Elapsed, math.Max(0, now.Sub(start).Seconds()))
					p.Status = "running"
				}
			}
			b := &p.Blocks[p.Index]
			if !strings.EqualFold(b.Scenario.Name, sum.Scenario) {
				continue
			}
			b.Recorded += sum.Duration
			p.Recorded += sum.Duration
			b.Runs++
			b.LastCompletedAt = end.UnixMilli()
			s.capturePractice(b, r, now)
			b.Best = math.Max(b.Best, sum.Score)
			if b.Target > 0 && b.Signature != "" && signature(sum) != b.Signature {
				b.Target = 0
				b.Reason += " 本模块检测到版本或设置变化，已取消原阈值。"
			}
			if b.Target > 0 && b.Personalization != nil && practiceSample(r).Signature != b.Personalization.Anchor.Signature {
				b.Target = 0
				b.Reason += " 个人锚点的版本、设置或时长不再可比，取消原目标。"
			}
			outcome := ""
			if p.Preferences.ExecutionMode == "playlist" {
				count := b.PlayCount
				if count < 1 {
					count = 1
				}
				if b.Runs >= count {
					b.Outcome = "list_complete"
					// Repetitions are a target, not evidence that the player has
					// changed scenarios. Keep attributing extra runs to this block.
					if p.Index == len(p.Blocks)-1 {
						outcome = "list_complete"
						p.Reminder = "本次列表计划已完成，可以结束训练并休息。"
						s.pendingReminder = p.Reminder
					}
				}
			} else if b.Role == "benchmark" {
				outcome = "measured"
			} else if b.Target > 0 && sum.Score >= b.Target {
				outcome = "threshold"
			} else if p.BlockElapsed >= float64(b.Budget) || b.Recorded >= float64(b.Budget) {
				outcome = "time_limit"
			}
			if outcome != "" {
				boundary := now
				if p.Preferences.ExecutionMode == "playlist" {
					boundary = end
				}
				s.advance(outcome, boundary)
				if p.Status == "ready" && p.Preferences.ExecutionMode == "playlist" {
					p.Status = "running"
				} else if p.Status == "ready" && p.Preferences.AutoAdvance {
					next := p.Blocks[p.Index]
					if p.Elapsed+float64(next.Scenario.Seconds) <= float64(p.Preferences.Minutes*60) {
						p.Status = "running"
						launch = next.Scenario.Name
					}
				}
			}
		}
		// No run was completed at the cap: do not launch another scenario over a live run.
		if p.Status == "running" && p.Preferences.ExecutionMode != "playlist" && p.BlockElapsed >= float64(p.Blocks[p.Index].Budget) {
			p.Status = "waiting"
		}
	}
	if p.Status == "running" && p.Preferences.ExecutionMode == "playlist" && p.Index < len(p.Blocks) && p.BlockElapsed >= float64(p.Blocks[p.Index].Budget) && p.RemindedBlock != p.Index+1 {
		p.RemindedBlock = p.Index + 1
		p.Reminder = "当前关卡已达到本次时间预算。完成当前局后，请在游戏中换关或休息。"
		s.pendingReminder = p.Reminder
	}
	s.expire()
	if p.Status == "completed" {
		launch = ""
	}
	if p.Recorded != previousRecorded || p.Status != previousStatus || p.Index != previousIndex {
		s.syncSessionContextsLocked(runs, now)
		s.updateAnchorEvaluationsLocked(runs, now)
	}
	if err := s.checkpoint(now, p.Recorded != previousRecorded || p.Status != previousStatus || p.Index != previousIndex); err != nil {
		s.state.Error = "保存训练进度失败：" + err.Error()
	}
	return launch
}

func (s *Service) LaunchFailed(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Error = "无法启动关卡：" + err.Error()
	if p := s.state.Plan; p != nil {
		p.Status = "paused"
		p.LastTick = 0
	}
	_ = s.save()
}

func (s *Service) Export() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exportLocked()
}

func (s *Service) exportLocked() ([]byte, error) {
	p := s.state.Plan
	if p == nil {
		return nil, fmt.Errorf("请先生成计划")
	}
	rows := []map[string]any{}
	for _, b := range p.Blocks {
		count := b.PlayCount
		if count < 1 {
			count = 1
		}
		rows = append(rows, map[string]any{"scenarioName": b.Scenario.Name, "playCount": count})
	}
	return json.MarshalIndent(map[string]any{"playlistName": "Refleks Adaptive " + p.ID, "scenarioList": rows, "isFavorite": false}, "", "  ")
}

func (s *Service) evaluateLocked(runs []models.RunRecord, now time.Time) {
	key := evaluationFingerprint(runs, s.dataRevision)
	if key == s.evaluationKey && now.Sub(s.evaluatedAt) < 5*time.Minute {
		return
	}
	s.syncSessionContextsLocked(runs, now)
	s.updateStudiesLocked(now)
	s.updateAnchorEvaluationsLocked(runs, now)
	anchors := personalAnchors(s.state.Catalog, runs, s.state.RunContexts, now)
	s.auditTransferExposureLocked(runs, now)
	s.state.PersonalAnchors = nil
	for _, c := range s.state.Catalog {
		if a, ok := anchors[strings.ToLower(c.Name)]; ok {
			s.state.PersonalAnchors = append(s.state.PersonalAnchors, a)
		}
	}
	references := automaticReferences(s.state.Catalog, s.state.Preferences)
	obs := levelObservations(runs, now)
	for i := range s.state.Catalog {
		c := &s.state.Catalog[i]
		c.Evaluation = assessCatalog(*c, obs[strings.ToLower(c.Name)], now, references)
	}
	s.state.PlayerLevels = PlayerLevels(s.state.Catalog, runs, now)
	s.state.ThemePriorities = themePrioritiesWithLevels(s.state.Catalog, runs, now, s.state.PlayerLevels)
	s.state.TemplateTiers = map[string]string{}
	for _, theme := range vdimThemes {
		s.state.TemplateTiers[theme] = trainingTier(theme, s.state.PlayerLevels)
	}
	s.state.Skills = SkillProfile(s.state.Catalog, runs, automaticReferences(s.state.Catalog, s.state.Preferences), time.Now())
	s.state.SearchConfigured = os.Getenv("REFLEKS_BRAVE_API_KEY") != ""

	s.evaluationKey, s.evaluatedAt = key, now
}
