package training

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"aimmeow/internal/models"
)

type RunContext struct {
	ContextVersion    int     `json:"contextVersion,omitempty"`
	SessionID         string  `json:"sessionId,omitempty"`
	PlanID            string  `json:"planId,omitempty"`
	ProtocolID        string  `json:"protocolId,omitempty"`
	BlockPosition     int     `json:"blockPosition,omitempty"`
	BlockRun          int     `json:"blockRun,omitempty"`
	Ordinal           int     `json:"ordinal,omitempty"`
	SceneOrdinal      int     `json:"sceneOrdinal,omitempty"`
	PriorSeconds      float64 `json:"priorSeconds,omitempty"`
	PriorThemeSeconds float64 `json:"priorThemeSeconds,omitempty"`
	PriorSceneSeconds float64 `json:"priorSceneSeconds,omitempty"`
	ScoreValid        bool    `json:"scoreValid,omitempty"`
	StudyID           string  `json:"studyId,omitempty"`
	Phase             string  `json:"phase,omitempty"`
	Scenario          string  `json:"scenario"`
	FileSHA256        string  `json:"fileSHA256"`
	Signature         string  `json:"signature"`
	At                int64   `json:"at"`
}

type PracticeSample struct {
	Invalid       bool     `json:"invalid,omitempty"`
	StartedAt     int64    `json:"startedAt"`
	RunID         string   `json:"runId"`
	At            int64    `json:"at"`
	Score         float64  `json:"score"`
	Accuracy      *float64 `json:"accuracy,omitempty"`
	HitsPerSecond *float64 `json:"hitsPerSecond,omitempty"`
	Signature     string   `json:"signature"`
	Settings      string   `json:"settings"`
	FileSHA256    string   `json:"fileSHA256,omitempty"`
}

type AnchorPoint struct {
	At      int64   `json:"at"`
	Score   float64 `json:"score"`
	Samples int     `json:"samples"`
}

type PersonalAnchor struct {
	Scenario      string        `json:"scenario"`
	Theme         string        `json:"theme"`
	Status        string        `json:"status"`
	Evidence      string        `json:"evidence"`
	Signature     string        `json:"signature"`
	FileSHA256    string        `json:"fileSHA256,omitempty"`
	MedianScore   float64       `json:"medianScore"`
	ScoreMAD      float64       `json:"scoreMAD"`
	Accuracy      *float64      `json:"accuracy,omitempty"`
	HitsPerSecond *float64      `json:"hitsPerSecond,omitempty"`
	Samples       int           `json:"samples"`
	Sessions      int           `json:"sessions"`
	Days          int           `json:"days"`
	LastPlayed    int64         `json:"lastPlayed"`
	Points        []AnchorPoint `json:"points,omitempty"`
}

type SceneDecision struct {
	Source       string              `json:"source"`
	Anchor       PersonalAnchor      `json:"anchor"`
	Relation     *PrecisionRelation  `json:"relation,omitempty"`
	Prediction   *ResponsePrediction `json:"prediction,omitempty"`
	Requirements *ScenarioComparison `json:"requirements,omitempty"`
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func validPractice(r models.RunRecord, now time.Time) bool {
	s := r.Stats.Summary
	if s.Scenario == "" || !finite(s.Score) || s.Score < 0 || !finite(s.Duration) || s.Duration <= 0 || s.Duration > 3600 ||
		!finite(s.TimeRemaining) || s.TimeRemaining > 1 || s.PauseCount > 0 || s.PauseDuration > 0 ||
		!finite(s.AvgTargetScale) || s.AvgTargetScale != 0 && math.Abs(s.AvgTargetScale-1) > .001 ||
		!finite(s.AvgTimeDilation) || s.AvgTimeDilation != 0 && math.Abs(s.AvgTimeDilation-1) > .001 {
		return false
	}
	for _, v := range []float64{s.HorizSens, s.VertSens, s.FOV, s.DPI} {
		if !finite(v) {
			return false
		}
	}
	for _, event := range r.Stats.Events {
		if event.Cheated {
			return false
		}
	}
	at, err := time.Parse(time.RFC3339, s.DatePlayed)
	return err == nil && !at.After(now.Add(2*time.Second)) && !at.Before(now.AddDate(0, 0, -45))
}

func practiceSettings(s models.RunStatsSummary) string {
	// Duration/scale are part of score comparability, not an ability penalty.
	return fmt.Sprintf("%s|%s|%.6f|%.6f|%s|%.6f|%.6f|%.0f|%.3f|%.3f", s.GameVersion, s.SensScale, s.HorizSens, s.VertSens, s.FOVScale, s.FOV, s.DPI, s.Duration, s.AvgTargetScale, s.AvgTimeDilation)
}

func practiceSample(r models.RunRecord) PracticeSample {
	s := r.Stats.Summary
	at, _ := time.Parse(time.RFC3339, s.DatePlayed)
	p := PracticeSample{RunID: runKey(r), At: at.UnixMilli(), StartedAt: at.UnixMilli() - int64(s.Duration*1000), Score: s.Score, Settings: practiceSettings(s)}
	p.Signature = s.Hash + "|" + p.Settings
	if int64(s.HitCount)+int64(s.MissCount) > 0 && s.HitCount >= 0 && s.MissCount >= 0 {
		accuracy := float64(s.HitCount) / (float64(s.HitCount) + float64(s.MissCount))
		speed := float64(s.HitCount) / s.Duration
		p.Accuracy, p.HitsPerSecond = &accuracy, &speed
	} else if finite(s.Accuracy) && s.Accuracy > 0 && s.Accuracy <= 100 {
		accuracy := s.Accuracy
		if accuracy > 1 {
			accuracy /= 100
		}
		p.Accuracy = &accuracy
	}
	return p
}

func personalAnchors(catalog []Scenario, runs []models.RunRecord, contexts map[string]RunContext, now time.Time) map[string]PersonalAnchor {
	positions := sessionPositions(catalog, runs, 20*time.Minute, now)
	rows := map[string][]PracticeSample{}
	seen := map[string]bool{}
	for _, r := range runs {
		if seen[runKey(r)] || !validPractice(r, now) {
			continue
		}
		seen[runKey(r)] = true
		p := practiceSample(r)
		if c, ok := contexts[p.RunID]; ok && c.At == p.At && c.Signature == p.Signature && strings.EqualFold(c.Scenario, r.Stats.Summary.Scenario) {
			p.FileSHA256 = c.FileSHA256
		}
		name := strings.ToLower(r.Stats.Summary.Scenario)
		rows[name] = append(rows[name], p)
	}
	out := map[string]PersonalAnchor{}
	for _, s := range catalog {
		name := strings.ToLower(s.Name)
		data := rows[name]
		if len(data) == 0 {
			continue
		}
		sort.Slice(data, func(i, j int) bool { return data[i].At > data[j].At })
		// Once a current-file anchor can stand on its own, use only its bound
		// observations. A few new bindings must not relabel old named history.
		comparable, currentFile := []PracticeSample{}, []PracticeSample{}
		boundDays := map[string]bool{}
		for _, p := range data {
			if p.FileSHA256 != "" && s.LocalAssessment != nil && p.FileSHA256 != s.LocalAssessment.FileSHA256 {
				continue
			}
			if len(comparable) > 0 && p.Signature != comparable[0].Signature {
				continue
			}
			comparable = append(comparable, p)
			if p.FileSHA256 != "" && s.LocalAssessment != nil && p.FileSHA256 == s.LocalAssessment.FileSHA256 {
				currentFile = append(currentFile, p)
				boundDays[time.UnixMilli(p.At).In(now.Location()).Format("2006-01-02")] = true
			}
		}
		if len(comparable) == 0 {
			continue
		}
		data = comparable
		fileBound := len(currentFile) >= 3 && len(boundDays) >= 2
		if fileBound {
			data = currentFile
		}
		a := PersonalAnchor{Scenario: s.Name, Theme: scenarioTheme(s), Status: "provisional", Evidence: "history_unbound", Signature: data[0].Signature, LastPlayed: data[0].At}
		days := map[string]bool{}
		sessionScores, sessionAccuracy, sessionSpeed := [][]float64{}, [][]float64{}, [][]float64{}
		sessionTimes := []int64{}
		lastSession := ""
		bound := 0
		for _, p := range data {
			if p.Signature != a.Signature {
				continue
			}
			if p.FileSHA256 != "" && s.LocalAssessment != nil && p.FileSHA256 != s.LocalAssessment.FileSHA256 {
				continue
			}
			session := positions[p.RunID].SessionID
			if c := contexts[p.RunID]; c.ContextVersion >= 2 && c.SessionID != "" {
				session = c.SessionID
			}
			if session == "" {
				session = p.RunID
			}
			if lastSession == "" || lastSession != session {
				if len(sessionScores) == 6 {
					break
				}
				sessionScores = append(sessionScores, nil)
				sessionAccuracy = append(sessionAccuracy, nil)
				sessionSpeed = append(sessionSpeed, nil)
				sessionTimes = append(sessionTimes, p.At)
			}
			lastSession = session
			i := len(sessionScores) - 1
			if len(sessionScores[i]) >= 12 {
				continue
			}
			sessionScores[i] = append(sessionScores[i], p.Score)
			if p.Accuracy != nil {
				sessionAccuracy[i] = append(sessionAccuracy[i], *p.Accuracy)
			}
			if p.HitsPerSecond != nil {
				sessionSpeed[i] = append(sessionSpeed[i], *p.HitsPerSecond)
			}
			a.Samples++
			days[time.UnixMilli(p.At).In(now.Location()).Format("2006-01-02")] = true
			if p.FileSHA256 != "" && s.LocalAssessment != nil && p.FileSHA256 == s.LocalAssessment.FileSHA256 {
				bound++
			}
		}
		scores, accuracy, speed := []float64{}, []float64{}, []float64{}
		for i := range sessionScores {
			score := median(sessionScores[i])
			scores = append(scores, score)
			a.Points = append(a.Points, AnchorPoint{At: sessionTimes[i], Score: score, Samples: len(sessionScores[i])})
			if len(sessionAccuracy[i]) > 0 {
				accuracy = append(accuracy, median(sessionAccuracy[i]))
			}
			if len(sessionSpeed[i]) > 0 {
				speed = append(speed, median(sessionSpeed[i]))
			}
		}
		sort.Slice(a.Points, func(i, j int) bool { return a.Points[i].At < a.Points[j].At })
		a.Sessions, a.Days = len(scores), len(days)
		a.MedianScore = median(scores)
		deviations := []float64{}
		for _, score := range scores {
			deviations = append(deviations, math.Abs(score-a.MedianScore))
		}
		a.ScoreMAD = median(deviations) // Descriptive variability, never a confidence interval.
		if len(accuracy) > 0 {
			v := median(accuracy)
			a.Accuracy = &v
		}
		if len(speed) > 0 {
			v := median(speed)
			a.HitsPerSecond = &v
		}
		if a.Samples >= 3 && a.Days >= 2 && a.MedianScore > 0 {
			a.Status = "stable"
			if a.ScoreMAD/a.MedianScore > .15 {
				a.Status = "variable"
			}
		}
		if now.UnixMilli()-a.LastPlayed > int64(14*24*time.Hour/time.Millisecond) {
			a.Status = "stale"
		}
		if fileBound && bound >= 3 && a.Days >= 2 && s.LocalAssessment != nil {
			a.FileSHA256 = s.LocalAssessment.FileSHA256
			a.Evidence = "local_execution_context" // Local file context, not proof of the game's loaded payload.
		}
		if a.Samples > 0 {
			out[name] = a
		}
	}
	return out
}

func (s *Service) capturePractice(b *Block, r models.RunRecord, now time.Time) {
	p := practiceSample(r)
	p.Invalid = !validPractice(r, now)
	// Never bind old history to today's file. Require an unchanged, parsed local
	// file at generation and ingestion, observed before this run started.
	if a := b.Scenario.LocalAssessment; a != nil && a.Status == "file_parsed_model_unfitted" {
		for _, current := range s.state.Catalog {
			c := current.LocalAssessment
			if !strings.EqualFold(current.Name, b.Scenario.Name) || c == nil || c.Status != a.Status || c.FileSHA256 != a.FileSHA256 {
				continue
			}
			observed, err := time.Parse(time.RFC3339, a.ObservedAt)
			if err == nil && observed.UnixMilli() <= p.At-int64(r.Stats.Summary.Duration*1000) {
				p.FileSHA256 = a.FileSHA256
			}
			break
		}
	}
	if s.state.RunContexts == nil {
		s.state.RunContexts = map[string]RunContext{}
	}
	{
		c := s.state.RunContexts[p.RunID]
		c.Scenario, c.FileSHA256, c.Signature, c.At = b.Scenario.Name, p.FileSHA256, p.Signature, p.At
		c.ContextVersion, c.ScoreValid = 2, !p.Invalid
		if plan := s.state.Plan; plan != nil {
			c.PlanID, c.BlockPosition, c.BlockRun = plan.ID, plan.Index, b.Runs
		}
		if b.Measurement != nil {
			c.StudyID, c.Phase = b.Measurement.StudyID, b.Measurement.Phase
			c.ProtocolID = b.Measurement.ProtocolID
		} else if b.Assessment != nil {
			c.ProtocolID = b.Assessment.ProtocolID
		} else if s.state.Plan != nil && s.state.Plan.PlannerVersion >= 8 {
			c.ProtocolID = anchorObservationProtocol
		}
		s.state.RunContexts[p.RunID] = c
	}
	if b.Measurement != nil || b.Assessment != nil || s.state.Plan != nil && s.state.Plan.PlannerVersion >= 8 || b.Personalization != nil && b.Personalization.Relation != nil {
		for _, old := range b.Observations {
			if old.RunID == p.RunID {
				return
			}
		}
		// Extra retries cannot replace low scores or inflate the fixed protocol.
		if len(b.Observations) < max(1, b.PlayCount) {
			b.Observations = append(b.Observations, p)
		}
	}
}

func signatureForPersonalTarget(name string, runs []models.RunRecord, now time.Time) string {
	latest, result := int64(0), ""
	for _, r := range runs {
		if !strings.EqualFold(name, r.Stats.Summary.Scenario) || !validPractice(r, now) {
			continue
		}
		p := practiceSample(r)
		if p.At > latest {
			latest, result = p.At, signature(r.Stats.Summary)
		}
	}
	return result
}
