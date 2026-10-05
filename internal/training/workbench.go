package training

import (
	"aimmeow/internal/models"
	"aimmeow/internal/sceneanalysis"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
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

// WorkbenchDTO is a UI contract, not the persisted aggregate. Execution-only
// polling uses LiveState; history, raw definitions and run contexts stay backend.
type WorkbenchDTO struct {
	DemandCoverage    []DemandCoverage         `json:"demandCoverage,omitempty"`
	Version           int                      `json:"version"`
	Revision          uint64                   `json:"revision"`
	Catalog           []Scenario               `json:"catalog"`
	Preferences       Preferences              `json:"preferences"`
	Plan              *Plan                    `json:"plan"`
	Skills            []SkillStatus            `json:"skills"`
	Discovery         Discovery                `json:"discovery"`
	PlayerLevels      []PlayerLevel            `json:"playerLevels"`
	PersonalAnchors   []PersonalAnchor         `json:"personalAnchors,omitempty"`
	AnchorEvaluations []AnchorEvaluation       `json:"anchorEvaluations,omitempty"`
	TrainingStudies   []TrainingStudy          `json:"trainingStudies,omitempty"`
	ThemePriorities   map[string]ThemePriority `json:"themePriorities,omitempty"`
	TemplateTiers     map[string]string        `json:"templateTiers,omitempty"`
	Curricula         []Curriculum             `json:"curricula,omitempty"`
	Initializing      bool                     `json:"initializing"`
	Notice            string                   `json:"notice"`
	Error             string                   `json:"error"`
	SearchConfigured  bool                     `json:"searchConfigured"`
	RecentPlans       []PlanHistorySummary     `json:"recentPlans"`
}

type PlanHistorySummary struct {
	ID              string  `json:"id"`
	Created         string  `json:"created"`
	Status          string  `json:"status"`
	EndReason       string  `json:"endReason,omitempty"`
	EndedAt         int64   `json:"endedAt,omitempty"`
	BlockCount      int     `json:"blockCount"`
	CompletedBlocks int     `json:"completedBlocks"`
	ProcessedBlocks int     `json:"processedBlocks"`
	Runs            int     `json:"runs"`
	TargetRuns      int     `json:"targetRuns"`
	Minutes         int     `json:"minutes"`
	Elapsed         float64 `json:"elapsed"`
	Recorded        float64 `json:"recorded"`
}

type TrainingProgressDTO struct {
	Current     *PlanHistorySummary  `json:"current"`
	RecentPlans []PlanHistorySummary `json:"recentPlans"`
}

func blockCompleted(b Block, mode string) bool {
	if b.Runs <= 0 || b.Recorded <= 0 {
		return false
	}
	if mode == "playlist" {
		return b.Runs >= max(1, b.PlayCount)
	}
	return b.Outcome == "threshold" || b.Outcome == "measured" || b.Outcome == "time_limit" || b.Outcome == "list_complete"
}

func planSummary(p Plan) PlanHistorySummary {
	v := PlanHistorySummary{ID: p.ID, Created: p.Created, Status: p.Status, EndReason: p.EndReason, EndedAt: p.EndedAt, BlockCount: len(p.Blocks), Minutes: p.Preferences.Minutes, Elapsed: p.Elapsed, Recorded: p.Recorded}
	for _, b := range p.Blocks {
		v.Runs += b.Runs
		v.TargetRuns += max(1, b.PlayCount)
		if blockCompleted(b, p.Preferences.ExecutionMode) {
			v.CompletedBlocks++
		}
		if b.Outcome != "pending" && b.Outcome != "" {
			v.ProcessedBlocks++
		}
	}
	return v
}

// No catalog, assessment refresh or raw runs in the globally shared progress.
func (s *Service) Progress() TrainingProgressDTO {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := TrainingProgressDTO{RecentPlans: []PlanHistorySummary{}}
	if s.state.Plan != nil {
		p := planSummary(*s.state.Plan)
		v.Current = &p
	}
	for i := max(0, len(s.state.History)-30); i < len(s.state.History); i++ {
		v.RecentPlans = append(v.RecentPlans, planSummary(s.state.History[i]))
	}
	return v
}

func (s *Service) PlanRunIDs(id string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := []string{}
	for run, ctx := range s.state.RunContexts {
		if ctx.PlanID == id {
			ids = append(ids, run)
		}
	}
	sort.Strings(ids)
	return ids
}

type GenerateRequest struct {
	Preferences Preferences `json:"preferences"`
}

func (s *Service) WorkbenchJSON(runs []models.RunRecord) (string, error) {
	view, err := s.Workbench(runs)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(view)
	return string(b), err
}
func (s *Service) Workbench(runs []models.RunRecord) (*WorkbenchDTO, error) {

	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluateLocked(runs, time.Now())
	st := s.state
	view := WorkbenchDTO{Version: st.Version, Revision: s.dataRevision, Catalog: st.Catalog, Preferences: st.Preferences, Plan: st.Plan, Skills: st.Skills, Discovery: st.Discovery, PlayerLevels: st.PlayerLevels, PersonalAnchors: st.PersonalAnchors, AnchorEvaluations: st.AnchorEvaluations, TrainingStudies: st.TrainingStudies, ThemePriorities: st.ThemePriorities, TemplateTiers: st.TemplateTiers, Curricula: st.Curricula, Initializing: st.Initializing, Notice: st.Notice, Error: st.Error, SearchConfigured: st.SearchConfigured}
	view.RecentPlans = []PlanHistorySummary{}
	view.DemandCoverage = st.DemandCoverage
	for i := max(0, len(st.History)-10); i < len(st.History); i++ {
		p := st.History[i]
		view.RecentPlans = append(view.RecentPlans, planSummary(p))
	}
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
	if err != nil {
		return nil, err
	}
	var detached WorkbenchDTO
	err = json.Unmarshal(b, &detached)
	return &detached, err
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
		if a.Requirements != nil {
			r := *a.Requirements
			r.Definitions = nil
			r.Axes = []sceneanalysis.RequirementAxis{}
			r.Slots = []sceneanalysis.Slot{}
			r.Hazards = nil
			r.Influences = nil
			if r.Map != nil {
				m := *r.Map
				m.Objects = nil
				r.Map = &m
			}
			r.Targets = append([]sceneanalysis.Target{}, r.Targets...)
			r.Helpers = append([]sceneanalysis.Target{}, r.Helpers...)
			for i := range r.Targets {
				r.Targets[i].DodgeEntries = nil
				r.Targets[i].Abilities = nil
				r.Targets[i].Facts = []sceneanalysis.Fact{}
				r.Targets[i].Windows = nil
				r.Targets[i].MotionModels = nil
			}
			for i := range r.Helpers {
				r.Helpers[i].DodgeEntries = nil
				r.Helpers[i].Abilities = nil
				r.Helpers[i].Facts = []sceneanalysis.Fact{}
				r.Helpers[i].Windows = nil
				r.Helpers[i].MotionModels = nil
			}
			r.Features = append([]sceneanalysis.Fact{}, r.Features...)
			for i := range r.Features {
				r.Features[i].Sources = nil
			}
			a.Requirements = &r
		}
		s.LocalAssessment = &a
	}
	return s
}

// SceneRequirements reads one version, including fixed plan snapshots, on
// demand. Large raw definitions and map point records stay in the audit engine.
func (s *Service) SceneRequirements(name, hash string) (*sceneanalysis.Descriptor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var d *sceneanalysis.Descriptor
	accept := func(c Scenario) {
		a := c.LocalAssessment
		if c.Name == name && a != nil && a.Status == "file_parsed_model_unfitted" && (hash == "" || a.FileSHA256 == hash) && a.Requirements != nil {
			d = a.Requirements
		}
	}
	for _, c := range s.state.Catalog {
		accept(c)
	}
	if d == nil && hash != "" {
		if s.state.Plan != nil {
			for _, b := range s.state.Plan.Blocks {
				accept(b.Scenario)
			}
		}
		for _, p := range s.state.History {
			if d != nil {
				break
			}
			for _, b := range p.Blocks {
				accept(b.Scenario)
			}
		}
	}
	if d == nil {
		return nil, fmt.Errorf("没有这张图所选版本的场景需求；请重新扫描或刷新关卡库")
	}
	view := *d
	view.Definitions = nil
	if view.Map != nil {
		m := *view.Map
		m.Objects = nil
		view.Map = &m
	}
	data, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	var detached sceneanalysis.Descriptor
	err = json.Unmarshal(data, &detached)
	return &detached, err
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
	EndReason    string      `json:"endReason,omitempty"`
	EndedAt      int64       `json:"endedAt,omitempty"`
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

func (s *Service) LiveJSON() (string, error) { b, err := json.Marshal(s.Live()); return string(b), err }
func (s *Service) Live() LiveState {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := LiveState{Revision: s.dataRevision, Initializing: s.state.Initializing, Notice: s.state.Notice, Error: s.state.Error}
	if p := s.state.Plan; p != nil {
		v.Plan = &LivePlan{EndReason: p.EndReason, EndedAt: p.EndedAt, ID: p.ID, Status: p.Status, Index: p.Index, Elapsed: p.Elapsed, Recorded: p.Recorded, BlockElapsed: p.BlockElapsed, Reminder: p.Reminder}
		for _, b := range p.Blocks {
			v.Plan.Blocks = append(v.Plan.Blocks, LiveBlock{b.Recorded, b.Runs, b.Best, b.Outcome, b.Target, b.Reason})
		}
	}
	return v
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
	if !important && s.repository != nil {
		s.state.Revision = s.dataRevision
		return s.repository.SaveExecution(s.state)
	}
	return s.persist()
}
