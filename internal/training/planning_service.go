package training

import (
	"aimmeow/internal/models"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

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
		return nil, fmt.Errorf("这份还没练完喵。先结束当前训练，我再帮你安排新的。")
	}
	// Generation reserves assessments and replaces a draft in one transaction.
	// A failed validation/write must leave the previous plan and evidence intact.
	beforeBytes, copyErr := json.Marshal(s.state)
	if copyErr != nil {
		return nil, copyErr
	}
	var before State
	if copyErr = json.Unmarshal(beforeBytes, &before); copyErr != nil {
		return nil, copyErr
	}
	previousRevision, previousKey, previousAt := s.dataRevision, s.evaluationKey, s.evaluatedAt
	committed := false
	defer func() {
		if !committed {
			s.state = before
			s.dataRevision = previousRevision
			s.evaluationKey = previousKey
			s.evaluatedAt = previousAt
		}
	}()
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
			firstSeconds := firstRoutineSeconds(*t, s.state.Catalog, runs, progress, now)
			bundle, exploration, limits := selectTrainingExtras(*t, s.state.Catalog, runs, s.state.RunContexts, s.state.TrainingStudies, p, now, explored, firstSeconds, rng, s.state.Curricula)
			window, start, end, windowErr := curriculumWindowWithReserve(*t, s.state.Catalog, runs, p, now, history, progress, bundle.seconds(), false, s.state.Curricula...)
			if windowErr != nil {
				return nil, windowErr
			}
			plan, err = generateCurriculum(window, s.state.Catalog, runs, planningPreferences, now, rng, explored, false, s.state.Curricula...)
			if err == nil {
				plan.Blocks = append(append(bundle.before, plan.Blocks...), bundle.after...)
				if !s.scheduleAnchorEvaluationLocked(plan, runs, now) {
					requests := s.assessmentRequests(plan, runs, now)
					if len(requests) > 0 && requests[0].seconds > 0 {
						// Reserve only a real eligible assessment. Keep the original
						// order and leave the shortened tail for ordinary continuation.
						w, a, z, e := curriculumWindowWithReserve(*t, s.state.Catalog, runs, p, now, history, progress, bundle.seconds()+requests[0].seconds, false, s.state.Curricula...)
						if e == nil {
							candidate, e := generateCurriculum(w, s.state.Catalog, runs, planningPreferences, now, rng, explored, false, s.state.Curricula...)
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
				plan.Exploration, plan.Progression = exploration, limits
				if bundle.study != nil {
					bundle.study.PlanID = plan.ID
					s.state.TrainingStudies = append(s.state.TrainingStudies, *bundle.study)
				}
				anchors := personalAnchors(s.state.Catalog, runs, s.state.RunContexts, now, s.sessionGap)
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
	committed = true
	b, _ := json.Marshal(plan)
	var copy Plan
	_ = json.Unmarshal(b, &copy)
	return &copy, nil
}

// Export serializes the fixed playlist; execution progress stays separate.
func (s *Service) Export() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exportLocked()
}

func (s *Service) exportLocked() ([]byte, error) {
	p := s.state.Plan
	if p == nil {
		return nil, fmt.Errorf("先让我安排一份训练列表喵。")
	}
	rows := []map[string]any{}
	for _, b := range p.Blocks {
		count := b.PlayCount
		if count < 1 {
			count = 1
		}
		rows = append(rows, map[string]any{"scenarioName": b.Scenario.Name, "playCount": count})
	}
	return json.MarshalIndent(map[string]any{"playlistName": "AimMeow Training " + p.ID, "scenarioList": rows, "isFavorite": false}, "", "  ")
}
