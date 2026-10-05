package training

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"aimmeow/internal/models"
)

type GuidanceItem struct {
	Scenario string `json:"scenario"`
	Role     string `json:"role"`
	Runs     int    `json:"runs"`
	Reason   string `json:"reason"`
	Origin   string `json:"origin"`
}
type TrainingGuidance struct {
	Mode        string             `json:"mode"`
	PlanID      string             `json:"planId,omitempty"`
	Status      string             `json:"status,omitempty"`
	Theme       string             `json:"theme,omitempty"`
	Reason      string             `json:"reason"`
	Items       []GuidanceItem     `json:"items"`
	Exploration *ExplorationReport `json:"exploration,omitempty"`
}
type guidanceCache struct {
	key   string
	at    time.Time
	value *TrainingGuidance
}

func activeTraining(p *Plan) bool { return p != nil && p.Status != "completed" }
func guidanceItem(b Block) GuidanceItem {
	origin := "主线安排"
	if b.Role == "explore" || b.Role == "challenge" {
		origin = "同目标变式"
	}
	if len(b.Scenario.Sources) > 0 && b.Scenario.Sources[0].Title != "" {
		origin = b.Scenario.Sources[0].Title
	}
	return GuidanceItem{Scenario: b.Scenario.Name, Role: b.Role, Runs: max(0, max(1, b.PlayCount)-b.Runs), Reason: b.Reason, Origin: origin}
}
func currentGuidance(p *Plan) *TrainingGuidance {
	if !activeTraining(p) {
		return nil
	}
	v := &TrainingGuidance{Mode: "current", PlanID: p.ID, Status: p.Status, Theme: p.Theme, Reason: p.SelectionReason, Items: []GuidanceItem{}, Exploration: p.Exploration}
	if v.Reason == "" {
		v.Reason = "继续当前固定列表；新成绩与探索结果用于下一次安排。"
	}
	for i := max(0, p.Index); i < len(p.Blocks) && len(v.Items) < 5; i++ {
		v.Items = append(v.Items, guidanceItem(p.Blocks[i]))
	}
	copy := detachGuidance(*v)
	return &copy
}

func detachGuidance(v TrainingGuidance) TrainingGuidance {
	data, _ := json.Marshal(v)
	var copy TrainingGuidance
	_ = json.Unmarshal(data, &copy)
	return copy
}

// Guidance is read-only: it cannot generate a persisted list, reserve an
// assessment or mutate the active definition. Both UI surfaces consume it.
func (s *Service) Guidance(runs []models.RunRecord) TrainingGuidance {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Initializing {
		s.evaluateLocked(runs, time.Now())
	}
	return s.guidanceLocked(runs, time.Now())
}
func (s *Service) guidanceLocked(runs []models.RunRecord, now time.Time) TrainingGuidance {
	if s.state.Initializing {
		return TrainingGuidance{Mode: "initializing", Reason: "正在读取已有成绩与训练历史。", Items: []GuidanceItem{}}
	}
	if current := currentGuidance(s.state.Plan); current != nil {
		return *current
	}
	key := fmt.Sprintf("%s|%d", s.evaluationKey, s.dataRevision)
	if s.guidance.key == key && s.guidance.value != nil && now.Sub(s.guidance.at) < 30*time.Second {
		return detachGuidance(*s.guidance.value)
	}
	p := automaticTrainingPreferences(s.state.Preferences)
	history := append([]Plan(nil), s.state.History...)
	if s.state.Plan != nil {
		history = append(history, *s.state.Plan)
	}
	v := TrainingGuidance{Mode: "next", Items: []GuidanceItem{}, Reason: "结合已有成绩与练习覆盖安排下一次训练。"}
	t, err := selectCurriculumWithHistory(s.state.Curricula, s.state.Catalog, p, runs, now, history, s.state.CurriculumProgress)
	if err != nil {
		v.Mode = "unavailable"
		v.Reason = err.Error()
		return v
	}
	progress := progressFor(*t, history, s.state.CurriculumProgress)
	v.Theme = t.Theme
	v.Reason = ThemePriorities(s.state.Catalog, runs, now)[t.Theme].Evidence
	if resumeDue(progress, now) {
		v.Reason = "24 小时内优先续接尚未完成的练习。"
	}
	first := firstRoutineSeconds(*t, s.state.Catalog, runs, progress, now)
	baseline := progress.EverCompleted || curriculumBaseline(*t, history)
	extra, report, _ := selectTrainingExtras(*t, s.state.Catalog, runs, s.state.RunContexts, s.state.TrainingStudies, p, now, baseline, first, rand.New(rand.NewSource(0)), s.state.Curricula)
	window, _, _, err := curriculumWindowWithReserve(*t, s.state.Catalog, runs, p, now, history, progress, extra.seconds(), false, s.state.Curricula...)
	if err != nil {
		v.Mode = "unavailable"
		v.Reason = err.Error()
		return v
	}
	for _, row := range window.Rows {
		if len(v.Items) >= 3 {
			break
		}
		role := row.Role
		if role == "" {
			role = "practice"
		}
		v.Items = append(v.Items, GuidanceItem{Scenario: row.Name, Role: role, Runs: row.Count, Reason: "保留主线训练目标与顺序，按可用时间安排短组。", Origin: t.Name})
	}
	for _, b := range extra.after {
		if len(v.Items) < 5 {
			v.Items = append(v.Items, guidanceItem(b))
		}
	}
	v.Exploration = report
	// Detach nested slices from state before returning through either endpoint.
	copy := detachGuidance(v)
	s.guidance = guidanceCache{key: key, at: now, value: &copy}
	return detachGuidance(copy)
}

func firstRoutineSeconds(t Curriculum, catalog []Scenario, runs []models.RunRecord, progress RoutineProgress, now time.Time) int {
	obs := observed(runs)
	for i, row := range t.Rows {
		if !progress.complete() && i < len(progress.Rows) && progress.Rows[i].Target > 0 && progress.Rows[i].Completed >= progress.Rows[i].Target {
			continue
		}
		for _, c := range catalog {
			if strings.EqualFold(row.Name, c.Name) {
				return estimateTiming(c, obs[strings.ToLower(c.Name)], now).Seconds
			}
		}
		break
	}
	return 0
}
