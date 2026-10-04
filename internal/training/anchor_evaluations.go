package training

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"refleks/internal/models"
)

const dailyTrialProtocol = "daily_trial_v2"
const anchorObservationProtocol = "anchor_v2_1plus3"

func practiceDayCount(rows []observation, now time.Time) int {
	days := map[string]bool{}
	for _, r := range comparable(rows) {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -45)) {
			days[r.at.In(now.Location()).Format("2006-01-02")] = true
		}
	}
	return len(days)
}

// One module serves both practice and observation. Only MainPlayCount advances
// the original curriculum; the additional runs remain assessment exposure.
type AssessmentSpec struct {
	ID            string `json:"id"`
	ProtocolID    string `json:"protocolId"`
	MainPlayCount int    `json:"mainPlayCount"`
	ExtraRuns     int    `json:"extraRuns"`
}

type ExposureSummary struct {
	RecordedSeconds  float64  `json:"recordedSeconds"`
	SameSceneSeconds float64  `json:"sameSceneSeconds"`
	SameThemeSeconds float64  `json:"sameThemeSeconds"`
	TrialSeconds     float64  `json:"trialSeconds"`
	TrialScenarios   []string `json:"trialScenarios,omitempty"`
}

type AnchorEvaluation struct {
	ID             string             `json:"id"`
	PlanID         string             `json:"planId"`
	Theme          string             `json:"theme"`
	Scenario       string             `json:"scenario"`
	FileSHA256     string             `json:"fileSHA256"`
	ProtocolID     string             `json:"protocolId"`
	Status         string             `json:"status"`
	Reason         string             `json:"reason"`
	CreatedAt      int64              `json:"createdAt"`
	StartedAt      int64              `json:"startedAt,omitempty"`
	ExtraRuns      int                `json:"extraRuns"`
	ExtraSeconds   int                `json:"extraSeconds"`
	Result         *MeasurementResult `json:"result,omitempty"`
	Previous       *MeasurementResult `json:"previous,omitempty"`
	Change         *float64           `json:"change,omitempty"`
	IntervalHours  float64            `json:"intervalHours,omitempty"`
	IntervalKind   string             `json:"intervalKind,omitempty"`
	ComparableDays int                `json:"comparableDays"`
	Exposure       ExposureSummary    `json:"exposure"`
}

func mainPlayCount(b Block) int {
	if b.Assessment != nil {
		return b.Assessment.MainPlayCount
	}
	return b.PlayCount
}

// Exposure includes paused/scaled/low-score runs even when their scores cannot
// be compared. Missing CSVs, sleep and practice in other games remain unknown.
func validExposure(r models.RunRecord, now time.Time) bool {
	v := r.Stats.Summary
	at, err := time.Parse(time.RFC3339, v.DatePlayed)
	return err == nil && v.Scenario != "" && finite(v.Duration) && v.Duration > 0 && v.Duration <= 3600 &&
		!at.After(now.Add(2*time.Second)) && !at.Before(now.AddDate(0, 0, -45))
}

func sessionPositions(catalog []Scenario, runs []models.RunRecord, gap time.Duration, now time.Time) map[string]RunContext {
	if gap <= 0 {
		gap = 20 * time.Minute
	}
	themes := map[string]string{}
	for _, c := range catalog {
		themes[strings.ToLower(c.Name)] = scenarioTheme(c)
	}
	ordered := []models.RunRecord{}
	seen := map[string]bool{}
	for _, r := range runs {
		if !seen[runKey(r)] && validExposure(r, now) {
			ordered = append(ordered, r)
			seen[runKey(r)] = true
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := parseTime(ordered[i].Stats.Summary.DatePlayed), parseTime(ordered[j].Stats.Summary.DatePlayed)
		if a.Equal(b) {
			return runKey(ordered[i]) < runKey(ordered[j])
		}
		return a.Before(b)
	})
	out := map[string]RunContext{}
	var last time.Time
	session, ordinal, elapsed := "", 0, 0.0
	sceneOrdinals := map[string]int{}
	sceneSeconds, themeSeconds := map[string]float64{}, map[string]float64{}
	for _, r := range ordered {
		p := practiceSample(r)
		start, end := time.UnixMilli(p.StartedAt), time.UnixMilli(p.At)
		if last.IsZero() || start.Sub(last) >= gap {
			session, ordinal, elapsed = "session-"+p.RunID, 0, 0
			sceneOrdinals = map[string]int{}
			sceneSeconds, themeSeconds = map[string]float64{}, map[string]float64{}
		}
		name := strings.ToLower(r.Stats.Summary.Scenario)
		theme := themes[name]
		ordinal++
		sceneOrdinals[name]++
		out[p.RunID] = RunContext{ContextVersion: 2, Scenario: r.Stats.Summary.Scenario, At: p.At, Signature: p.Signature,
			SessionID: session, Ordinal: ordinal, SceneOrdinal: sceneOrdinals[name], PriorSeconds: elapsed,
			PriorSceneSeconds: sceneSeconds[name], PriorThemeSeconds: themeSeconds[theme], ScoreValid: validPractice(r, now)}
		elapsed += r.Stats.Summary.Duration
		sceneSeconds[name] += r.Stats.Summary.Duration
		themeSeconds[theme] += r.Stats.Summary.Duration
		if end.After(last) {
			last = end
		}
	}
	return out
}

// The application's existing session-gap preference also governs training.
func (s *Service) SetSessionGap(gap time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if gap <= 0 {
		gap = 20 * time.Minute
	}
	if gap != s.sessionGap {
		s.sessionGap = gap
		s.evaluationKey = ""
	}
}

func (s *Service) syncSessionContextsLocked(runs []models.RunRecord, now time.Time) {
	if s.state.RunContexts == nil {
		s.state.RunContexts = map[string]RunContext{}
	}
	for id, pos := range sessionPositions(s.state.Catalog, runs, s.sessionGap, now) {
		old := s.state.RunContexts[id]
		// Preserve only the file binding observed at execution. Deriving session
		// positions from old CSVs cannot establish a protocol or execution plan.
		if old.At == pos.At && old.Signature == pos.Signature && strings.EqualFold(old.Scenario, pos.Scenario) {
			pos.FileSHA256, pos.StudyID, pos.Phase = old.FileSHA256, old.StudyID, old.Phase
			pos.PlanID, pos.ProtocolID = old.PlanID, old.ProtocolID
			pos.BlockPosition, pos.BlockRun = old.BlockPosition, old.BlockRun
		}
		s.state.RunContexts[id] = pos
	}
}

func (s *Service) updateDailyTrialLocked(st *TrainingStudy, index map[string]Scenario, now time.Time) {
	if st.Status == "not_started" || st.Status == "invalidated" || st.Status == "incomplete" {
		return
	}
	if st.Trial != nil {
		st.Status = "observed"
		return
	}
	for name, hash := range map[string]string{st.AnchorScenario: st.AnchorHash, st.TrainingScenario: st.TrainingHash} {
		c, ok := index[strings.ToLower(name)]
		if !ok || c.LocalAssessment == nil || c.LocalAssessment.FileSHA256 != hash || c.LocalAssessment.Status != "file_parsed_model_unfitted" {
			st.Status = "invalidated"
			return
		}
	}
	for _, plan := range s.evidencePlans() {
		if plan.ID != st.PlanID {
			continue
		}
		for _, b := range plan.Blocks {
			if b.Measurement == nil || b.Measurement.StudyID != st.ID || b.Measurement.Phase != "trial" {
				continue
			}
			if r := measured(b.Observations, 0, 2, st.TrainingHash); r != nil && !b.ObservationInterrupted {
				r.ProtocolID = dailyTrialProtocol
				st.Trial, st.TrainedAt, st.Status = r, r.At, "observed"
				st.DueAt, st.ExpiresAt = 0, 0
			} else if b.Runs >= b.PlayCount || b.Outcome == "skipped" || b.Outcome == "missed" || plan.Status == "completed" {
				st.Status = "incomplete"
			}
		}
	}
}

func (s *Service) evidencePlans() []*Plan {
	plans := make([]*Plan, 0, len(s.state.History)+1)
	for i := range s.state.History {
		plans = append(plans, &s.state.History[i])
	}
	if s.state.Plan != nil {
		plans = append(plans, s.state.Plan)
	}
	return plans
}

func evaluationID(planID string, position int) string {
	return fmt.Sprintf("anchor-%s-%d", planID, position)
}

func (s *Service) assessmentAllowed(theme string, now time.Time) bool {
	count := 0
	for _, e := range s.state.AnchorEvaluations {
		if e.ExtraRuns <= 0 || e.StartedAt <= now.Add(-7*24*time.Hour).UnixMilli() || e.StartedAt > now.UnixMilli() {
			continue
		}
		if e.Theme == theme {
			return false
		}
		count++
	}
	return count < 2
}

func (s *Service) assessmentReason(scene Scenario, a PersonalAnchor, runs []models.RunRecord, now time.Time) string {
	if a.Samples < 3 || a.Days < 2 {
		return ""
	}
	if a.Status == "variable" {
		return "近期可比成绩波动较大，确认当前水平"
	}
	if a.Status == "stale" {
		return "锚图长期未练，更新当前水平"
	}
	trend, _ := performanceTrend(levelObservations(runs, now)[strings.ToLower(scene.Name)], now)
	if trend == "declining" && a.Days >= 3 {
		return "多个训练日持续下降，确认当前水平"
	}
	var since int64
	for _, e := range s.state.AnchorEvaluations {
		if strings.EqualFold(e.Scenario, scene.Name) && e.Result != nil {
			since = max(since, e.Result.At)
		}
	}
	days := map[string]bool{}
	for _, st := range s.state.TrainingStudies {
		if st.ProtocolID == dailyTrialProtocol && strings.EqualFold(st.AnchorScenario, scene.Name) && st.Trial != nil && st.Trial.At > since && st.Trial.At >= now.AddDate(0, 0, -45).UnixMilli() {
			days[time.UnixMilli(st.Trial.At).In(now.Location()).Format("2006-01-02")] = true
		}
	}
	if len(days) >= 3 {
		return "已积累多个训练日的探索，更新共同锚图；不归因于单个变体"
	}
	return ""
}

type assessmentRequest struct {
	index, extra, seconds int
	reason                string
}

func (s *Service) assessmentRequests(plan *Plan, runs []models.RunRecord, now time.Time) []assessmentRequest {
	if !s.assessmentAllowed(plan.Theme, now) {
		return nil
	}
	anchors := personalAnchors(s.state.Catalog, runs, s.state.RunContexts, now)
	limit := min(180, plan.Preferences.Minutes*60/10)
	requests := []assessmentRequest{}
	for i, b := range plan.Blocks {
		a := anchors[strings.ToLower(b.Scenario.Name)]
		if b.Role != "practice" || b.Measurement != nil || b.PlayCount <= 0 || b.Assessment != nil || b.Scenario.LocalAssessment == nil || b.Scenario.LocalAssessment.Status != "file_parsed_model_unfitted" {
			continue
		}
		reason := s.assessmentReason(b.Scenario, a, runs, now)
		if reason == "" {
			continue
		}
		covered := false
		for _, e := range s.state.AnchorEvaluations {
			if e.Result != nil && strings.EqualFold(e.Scenario, b.Scenario.Name) && e.FileSHA256 == b.Scenario.LocalAssessment.FileSHA256 && e.Result.Signature == a.Signature && e.Result.At >= a.LastPlayed && now.UnixMilli()-e.Result.At < int64(7*24*time.Hour/time.Millisecond) && strings.HasPrefix(e.Result.ContextKey, fmt.Sprintf("%d|", i)) {
				covered = true
			}
		}
		if covered {
			continue
		}
		extra := max(0, 4-b.PlayCount)
		seconds := extra * b.Timing.Seconds
		if seconds <= limit {
			requests = append(requests, assessmentRequest{i, extra, seconds, reason})
		}
	}
	return requests
}

func (s *Service) scheduleAnchorEvaluationLocked(plan *Plan, runs []models.RunRecord, now time.Time) bool {
	used := 0
	for _, b := range plan.Blocks {
		used += b.Budget
	}
	for _, request := range s.assessmentRequests(plan, runs, now) {
		if used+request.seconds > plan.Preferences.Minutes*60*9/10 {
			continue
		}
		i := request.index
		b := &plan.Blocks[i]
		id := evaluationID(plan.ID, i)
		b.Assessment = &AssessmentSpec{ID: id, ProtocolID: anchorObservationProtocol, MainPlayCount: b.PlayCount, ExtraRuns: request.extra}
		b.PlayCount += request.extra
		b.Budget += request.seconds
		b.Reason += " 固定观察嵌入本模块：一局熟悉、三局测量；仅原定次数推进主线。"
		b.Cue = "按平常节奏完成固定局数；低分保留，额外重试不替换测量。"
		s.state.AnchorEvaluations = append(s.state.AnchorEvaluations, AnchorEvaluation{ID: id, PlanID: plan.ID, Theme: plan.Theme, Scenario: b.Scenario.Name,
			FileSHA256: b.Scenario.LocalAssessment.FileSHA256, ProtocolID: anchorObservationProtocol, Status: "planned", Reason: request.reason,
			CreatedAt: now.UnixMilli(), ExtraRuns: request.extra, ExtraSeconds: request.seconds})
		return true
	}
	return false
}

func (s *Service) observationResult(block Block, position int) *MeasurementResult {
	if block.ObservationInterrupted || block.Scenario.LocalAssessment == nil {
		return nil
	}
	r := measured(block.Observations, 1, 3, block.Scenario.LocalAssessment.FileSHA256)
	if r == nil {
		return nil
	}
	var first RunContext
	for i, p := range block.Observations[:4] {
		c := s.state.RunContexts[p.RunID]
		if c.ContextVersion < 2 || c.ProtocolID != anchorObservationProtocol || c.BlockPosition != position || c.BlockRun != i+1 || c.SessionID == "" {
			return nil
		}
		if i == 0 {
			first = c
		} else if c.SessionID != first.SessionID || c.Ordinal != first.Ordinal+i {
			return nil
		}
	}
	r.ProtocolID = anchorObservationProtocol
	// Exact run ordinal and a coarse prior-load band are conservative context
	// gates, not a physiological warm-up/fatigue estimate.
	r.ContextKey = fmt.Sprintf("%d|%d|%d|%d|%d|%d", position, first.Ordinal, first.SceneOrdinal, int(first.PriorSeconds/300), int(first.PriorThemeSeconds/300), int(first.PriorSceneSeconds/300))
	return r
}

func (s *Service) updateAnchorEvaluationsLocked(runs []models.RunRecord, now time.Time) {
	for _, plan := range s.evidencePlans() {
		if plan.PlannerVersion < 8 || plan.Status == "draft" {
			continue
		}
		explicit := -1
		for i, b := range plan.Blocks {
			if b.Assessment != nil {
				explicit = i
				break
			}
		}
		for position, b := range plan.Blocks {
			if b.Role != "practice" || b.Measurement != nil || b.PlayCount < 4 || b.Scenario.LocalAssessment == nil || explicit >= 0 && position != explicit {
				continue
			}
			id := evaluationID(plan.ID, position)
			var e *AnchorEvaluation
			for i := range s.state.AnchorEvaluations {
				if s.state.AnchorEvaluations[i].ID == id {
					e = &s.state.AnchorEvaluations[i]
					break
				}
			}
			if e == nil {
				if len(b.Observations) == 0 {
					continue
				}
				s.state.AnchorEvaluations = append(s.state.AnchorEvaluations, AnchorEvaluation{ID: id, PlanID: plan.ID, Theme: plan.Theme, Scenario: b.Scenario.Name,
					FileSHA256: b.Scenario.LocalAssessment.FileSHA256, ProtocolID: anchorObservationProtocol, Status: "planned", Reason: "复用主线中已有的固定连续记录", CreatedAt: parseTime(plan.Created).UnixMilli()})
				e = &s.state.AnchorEvaluations[len(s.state.AnchorEvaluations)-1]
			}
			if e.Status == "not_started" {
				break
			}
			if e.StartedAt == 0 && len(b.Observations) > 0 {
				e.StartedAt = b.Observations[0].StartedAt
			}
			if b.ObservationInterrupted {
				e.Result, e.Previous, e.Change = nil, nil, nil
				e.Status = "incomparable"
				break
			}
			if e.Result == nil {
				if r := s.observationResult(b, position); r != nil {
					e.Result, e.Status = r, "observed"
				} else if b.Runs >= 4 || b.Outcome == "skipped" || b.Outcome == "missed" || plan.Status == "completed" {
					e.Status = "incomparable"
				}
			}
			break // One anchor per plan; ordinary practice remains unrestricted.
		}
	}
	for i := range s.state.AnchorEvaluations {
		e := &s.state.AnchorEvaluations[i]
		if e.Result == nil {
			continue
		}
		e.Previous, e.Change = nil, nil
		e.IntervalHours, e.IntervalKind, e.ComparableDays = 0, "", 0
		days := map[string]bool{}
		for _, other := range s.state.AnchorEvaluations {
			if other.Result == nil || other.Result.At > e.Result.At || !strings.EqualFold(other.Scenario, e.Scenario) || other.FileSHA256 != e.FileSHA256 || other.ProtocolID != e.ProtocolID || other.Result.Signature != e.Result.Signature || other.Result.ContextKey != e.Result.ContextKey {
				continue
			}
			days[time.UnixMilli(other.Result.At).In(now.Location()).Format("2006-01-02")] = true
			if other.Result.At < e.Result.At && (e.Previous == nil || other.Result.At > e.Previous.At) {
				e.Previous = other.Result
			}
		}
		e.ComparableDays = len(days)
		if e.Previous == nil {
			e.Status = "observed"
			continue
		}
		e.Status = "compared"
		e.IntervalHours = float64(e.Result.At-e.Previous.At) / float64(time.Hour/time.Millisecond)
		e.IntervalKind = "cross_day"
		if time.UnixMilli(e.Result.At).In(now.Location()).Format("2006-01-02") == time.UnixMilli(e.Previous.At).In(now.Location()).Format("2006-01-02") {
			e.IntervalKind = "same_day"
		} else if e.IntervalHours >= 72 {
			e.IntervalKind = "longer_interval"
		}
		if e.Previous.Score > 0 {
			v := e.Result.Score/e.Previous.Score - 1
			e.Change = &v
		}
		e.Exposure = s.intervalExposure(*e, runs, now)
	}
	if len(s.state.AnchorEvaluations) > 128 {
		// High volumes of free observations must not erase the seven-day
		// assessment quota. Keep started supplemental tests before other rows.
		protected := map[string]bool{}
		for _, e := range s.state.AnchorEvaluations {
			if e.ExtraRuns > 0 && e.StartedAt > now.Add(-7*24*time.Hour).UnixMilli() {
				protected[e.ID] = true
			}
		}
		keep := []AnchorEvaluation{}
		ordinary := max(0, 128-len(protected))
		for i := len(s.state.AnchorEvaluations) - 1; i >= 0; i-- {
			e := s.state.AnchorEvaluations[i]
			if protected[e.ID] || ordinary > 0 {
				keep = append(keep, e)
				if !protected[e.ID] {
					ordinary--
				}
			}
		}
		for i, j := 0, len(keep)-1; i < j; i, j = i+1, j-1 {
			keep[i], keep[j] = keep[j], keep[i]
		}
		s.state.AnchorEvaluations = keep
	}
}

func (s *Service) intervalExposure(e AnchorEvaluation, runs []models.RunRecord, now time.Time) ExposureSummary {
	result := ExposureSummary{}
	themes := map[string]string{}
	for _, c := range s.state.Catalog {
		themes[strings.ToLower(c.Name)] = scenarioTheme(c)
	}
	excluded := map[string]bool{}
	for _, id := range e.Result.RunIDs {
		excluded[id] = true
	}
	seen, trials := map[string]bool{}, map[string]bool{}
	for _, r := range runs {
		id := runKey(r)
		if seen[id] || excluded[id] || !validExposure(r, now) {
			continue
		}
		seen[id] = true
		p := practiceSample(r)
		if p.At <= e.Previous.At || p.At > e.Result.At {
			continue
		}
		duration, name := r.Stats.Summary.Duration, r.Stats.Summary.Scenario
		result.RecordedSeconds += duration
		if strings.EqualFold(name, e.Scenario) {
			result.SameSceneSeconds += duration
		}
		if themes[strings.ToLower(name)] == e.Theme {
			result.SameThemeSeconds += duration
		}
		if c := s.state.RunContexts[id]; c.ProtocolID == dailyTrialProtocol {
			result.TrialSeconds += duration
			trials[name] = true
		}
	}
	for name := range trials {
		result.TrialScenarios = append(result.TrialScenarios, name)
	}
	sort.Strings(result.TrialScenarios)
	return result
}

func parseTime(v string) time.Time { t, _ := time.Parse(time.RFC3339, v); return t }
