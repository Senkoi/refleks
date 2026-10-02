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
	path            string
	state           State
	discovering     bool
	pendingReminder string
	localScanMu     sync.Mutex
	localFiles      map[string]string
	localAttempts   map[string]time.Time
	localParsed     map[string]localParsedSCE
	localDirty      bool
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
			*item = enrichMechanics(*item)
			if item.DifficultySource == "playlist" && item.Classification != "manual" {
				item.Difficulty, item.DifficultySource = "unknown", "unknown"
			}
		}
		if s.state.Preferences.ExecutionMode == "" {
			s.state.Preferences.ExecutionMode = "playlist"
		}
	}
	if p := s.state.Plan; p != nil && (p.Status == "running" || p.Status == "ready" || p.Status == "waiting") {
		p.Status = "paused"
		p.LastTick = 0
		s.state.Error = "应用重新启动，训练已暂停；请确认后继续。"
	}
	return s, nil
}

func (s *Service) save() error {
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
	s.state.Skills = SkillProfile(s.state.Catalog, runs, latestBenchmarkPreferences(s.state.Catalog, s.state.Preferences), time.Now())
	s.state.SearchConfigured = os.Getenv("REFLEKS_BRAVE_API_KEY") != ""
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

func (s *Service) Generate(p Preferences, runs []models.RunRecord) (*Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old := s.state.Plan; old != nil && (old.Status == "running" || old.Status == "ready" || old.Status == "paused" || old.Status == "waiting") {
		return nil, fmt.Errorf("请先结束当前计划，再生成新计划")
	}
	now := time.Now()
	rng := rand.New(rand.NewSource(now.UnixNano()))
	planningPreferences := latestBenchmarkPreferences(s.state.Catalog, p)
	var plan *Plan
	var err error
	if p.PlanningPolicy == "legacy" {
		plan, err = Generate(s.state.Catalog, runs, planningPreferences, now, rng)
	} else {
		p.PlanningPolicy = "curriculum"
		var t *Curriculum
		t, err = selectCurriculum(s.state.Curricula, s.state.Catalog, p, runs, now)
		if err == nil {
			explored := false
			for _, h := range s.state.History {
				if completedCurriculum(h, *t) {
					explored = true
				}
			}
			if old := s.state.Plan; old != nil && completedCurriculum(*old, *t) {
				explored = true
			}
			plan, err = GenerateCurriculum(*t, s.state.Catalog, runs, planningPreferences, now, rng, explored)
		}
	}
	if err != nil {
		return nil, err
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
			if err != nil || v.Duration <= 0 || math.IsNaN(v.Duration) || math.IsInf(v.Duration, 0) || v.Score < 0 || math.IsNaN(v.Score) || math.IsInf(v.Score, 0) {
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
			b.Best = math.Max(b.Best, v.Score)
		}
		if err := s.save(); err != nil {
			s.state.Error = "保存训练进度失败：" + err.Error()
		}
		return ""
	}
	if p == nil || (p.Status != "running" && p.Status != "ready" && p.Status != "waiting") {
		return ""
	}
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
			if sum.Duration <= 0 || math.IsNaN(sum.Duration) || math.IsInf(sum.Duration, 0) || sum.Score < 0 || math.IsNaN(sum.Score) || math.IsInf(sum.Score, 0) {
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
			b.Best = math.Max(b.Best, sum.Score)
			if b.Target > 0 && b.Signature != "" && signature(sum) != b.Signature {
				b.Target = 0
				b.Reason += " 本模块检测到版本或设置变化，已取消原阈值。"
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
	if err := s.save(); err != nil {
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
