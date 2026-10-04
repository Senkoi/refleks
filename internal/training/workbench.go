package training

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"refleks/internal/models"
	"time"
)

// This fingerprint is evaluated on the slow refresh, never on clock polling.
func evaluationFingerprint(runs []models.RunRecord, revision uint64) string {
	h := sha256.New()
	fmt.Fprint(h, revision)
	for _, r := range runs {
		fmt.Fprint(h, runKey(r))
		_ = json.NewEncoder(h).Encode(r.Stats.Summary)
		cheated := false
		for _, e := range r.Stats.Events {
			cheated = cheated || e.Cheated
		}
		fmt.Fprint(h, cheated)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *Service) WorkbenchJSON(runs []models.RunRecord) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluateLocked(runs, time.Now())
	view := s.state
	view.Revision = s.dataRevision
	// Historical execution and template rows are not used by the workbench.
	view.History = nil
	view.CurriculumProgress = nil
	view.RunContexts = nil
	view.AnchorEvaluations = append([]AnchorEvaluation(nil), view.AnchorEvaluations...)
	for i := range view.AnchorEvaluations {
		view.AnchorEvaluations[i].Result = workbenchMeasurement(view.AnchorEvaluations[i].Result)
		view.AnchorEvaluations[i].Previous = workbenchMeasurement(view.AnchorEvaluations[i].Previous)
	}
	view.TrainingStudies = append([]TrainingStudy(nil), view.TrainingStudies...)
	for i := range view.TrainingStudies {
		st := &view.TrainingStudies[i]
		st.Baseline, st.Trial, st.Retest = workbenchMeasurement(st.Baseline), workbenchMeasurement(st.Trial), workbenchMeasurement(st.Retest)
		st.TransferBaseline, st.TransferRetest = workbenchMeasurement(st.TransferBaseline), workbenchMeasurement(st.TransferRetest)
	}
	view.Catalog = append([]Scenario(nil), view.Catalog...)
	for i := range view.Catalog {
		view.Catalog[i] = workbenchScenario(view.Catalog[i])
	}
	if view.Plan != nil {
		p := *view.Plan
		p.Blocks = append([]Block(nil), p.Blocks...)
		for i := range p.Blocks {
			p.Blocks[i].Scenario = workbenchScenario(p.Blocks[i].Scenario)
			p.Blocks[i].Observations = nil
		}
		view.Plan = &p
	}
	view.Curricula = append([]Curriculum(nil), view.Curricula...)
	for i := range view.Curricula {
		view.Curricula[i].Rows = nil
	}
	b, err := json.Marshal(view)
	return string(b), err
}

func workbenchMeasurement(r *MeasurementResult) *MeasurementResult {
	if r == nil {
		return nil
	}
	view := *r
	view.RunIDs, view.ContextKey = nil, ""
	return &view
}

func workbenchScenario(s Scenario) Scenario {
	if s.LocalAssessment != nil {
		a := *s.LocalAssessment
		// Raw parser fields are needed by backend validation, never by the UI.
		a.Fields = nil
		s.LocalAssessment = &a
	}
	return s
}

type LiveBlock struct {
	Recorded float64 `json:"recorded"`
	Runs     int     `json:"runs"`
	Best     float64 `json:"best"`
	Outcome  string  `json:"outcome"`
	Target   float64 `json:"target"`
	Reason   string  `json:"reason"`
}
type LivePlan struct {
	ID           string      `json:"id"`
	Status       string      `json:"status"`
	Index        int         `json:"index"`
	Elapsed      float64     `json:"elapsed"`
	Recorded     float64     `json:"recorded"`
	BlockElapsed float64     `json:"blockElapsed"`
	Reminder     string      `json:"reminder"`
	Blocks       []LiveBlock `json:"blocks"`
}
type LiveState struct {
	Revision     uint64    `json:"revision"`
	Initializing bool      `json:"initializing"`
	Notice       string    `json:"notice"`
	Error        string    `json:"error"`
	Plan         *LivePlan `json:"plan"`
}

func (s *Service) LiveJSON() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := LiveState{Revision: s.dataRevision, Initializing: s.state.Initializing, Notice: s.state.Notice, Error: s.state.Error}
	if p := s.state.Plan; p != nil {
		v.Plan = &LivePlan{ID: p.ID, Status: p.Status, Index: p.Index, Elapsed: p.Elapsed, Recorded: p.Recorded, BlockElapsed: p.BlockElapsed, Reminder: p.Reminder}
		for _, b := range p.Blocks {
			v.Plan.Blocks = append(v.Plan.Blocks, LiveBlock{b.Recorded, b.Runs, b.Best, b.Outcome, b.Target, b.Reason})
		}
	}
	b, err := json.Marshal(v)
	return string(b), err
}
func (s *Service) checkpoint(now time.Time, important bool) error {
	if !important && !s.checkpointAt.IsZero() && now.Sub(s.checkpointAt) < 15*time.Second {
		return nil
	}
	s.checkpointAt = now
	if important {
		s.updateStudiesLocked(now)
		s.dataRevision++
		s.evaluationKey = ""
	}
	return s.persist()
}
