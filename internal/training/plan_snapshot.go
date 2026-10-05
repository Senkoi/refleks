package training

import "encoding/json"

// PlanDefinition is the immutable selection/configuration snapshot. Execution
// is stored independently; the legacy Plan is assembled only at the boundary.
type PlanDefinition struct {
	CurriculumCycle int                `json:"curriculumCycle,omitempty"`
	SelectionReason string             `json:"selectionReason,omitempty"`
	Progression     *ProgressionBudget `json:"progression,omitempty"`
	TierReason      string             `json:"tierReason,omitempty"`
	PlayerTier      string             `json:"playerTier,omitempty"`
	TemplateTier    string             `json:"templateTier,omitempty"`
	PlannerVersion  int                `json:"plannerVersion,omitempty"`
	CurriculumID    string             `json:"curriculumId,omitempty"`
	CurriculumHash  string             `json:"curriculumHash,omitempty"`
	CurriculumName  string             `json:"curriculumName,omitempty"`
	CurriculumStart int                `json:"curriculumStart,omitempty"`
	CurriculumEnd   int                `json:"curriculumEnd,omitempty"`
	CurriculumTotal int                `json:"curriculumTotal,omitempty"`
	Theme           string             `json:"theme,omitempty"`
	ID              string             `json:"id"`
	Created         string             `json:"created"`
	Preferences     Preferences        `json:"preferences"`
	Blocks          []BlockDefinition  `json:"blocks"`
	Warnings        []string           `json:"warnings"`
}
type BlockDefinition struct {
	Assessment         *AssessmentSpec    `json:"assessment,omitempty"`
	Personalization    *SceneDecision     `json:"personalization,omitempty"`
	Measurement        *MeasurementSpec   `json:"measurement,omitempty"`
	CurriculumRow      *int               `json:"curriculumRow,omitempty"`
	CompletedBefore    int                `json:"completedBefore,omitempty"`
	AnchorScenario     string             `json:"anchorScenario,omitempty"`
	Timing             TimingEstimate     `json:"timing"`
	DifficultyEvidence DifficultyEvidence `json:"difficultyEvidence"`
	Signature          string             `json:"signature,omitempty"`
	Scenario           Scenario           `json:"scenario"`
	Role               string             `json:"role"`
	Budget             int                `json:"budget"`
	SourcePlayCount    int                `json:"sourcePlayCount,omitempty"`
	PlayCount          int                `json:"playCount"`
	Cue                string             `json:"cue"`
	Benchmark          string             `json:"benchmark,omitempty"`
	InitialTarget      float64            `json:"initialTarget"`
	InitialReason      string             `json:"initialReason"`
}
type ExecutionProgress struct {
	PlanID        string           `json:"planId"`
	Blocks        []BlockExecution `json:"blocks"`
	EndReason     string           `json:"endReason,omitempty"`
	EndedAt       int64            `json:"endedAt,omitempty"`
	Status        string           `json:"status"`
	Index         int              `json:"index"`
	Elapsed       float64          `json:"elapsed"`
	Recorded      float64          `json:"recorded"`
	BlockElapsed  float64          `json:"blockElapsed"`
	LastTick      int64            `json:"lastTick"`
	AcceptAfter   int64            `json:"acceptAfter"`
	Seen          []string         `json:"seen"`
	RemindedBlock int              `json:"remindedBlock,omitempty"`
	RemindedEnd   bool             `json:"remindedEnd,omitempty"`
	Reminder      string           `json:"reminder,omitempty"`
}
type BlockExecution struct {
	ObservationInterrupted bool             `json:"observationInterrupted,omitempty"`
	Observations           []PracticeSample `json:"observations,omitempty"`
	LastCompletedAt        int64            `json:"lastCompletedAt,omitempty"`
	Target                 float64          `json:"target"`
	Reason                 string           `json:"reason"`
	Recorded               float64          `json:"recorded"`
	Runs                   int              `json:"runs"`
	Best                   float64          `json:"best"`
	Outcome                string           `json:"outcome"`
}

func definePlan(p *Plan) *PlanDefinition {
	if p == nil {
		return nil
	}
	d := &PlanDefinition{
		CurriculumCycle: p.CurriculumCycle,
		SelectionReason: p.SelectionReason,
		Progression:     p.Progression,
		TierReason:      p.TierReason,
		PlayerTier:      p.PlayerTier,
		TemplateTier:    p.TemplateTier,
		PlannerVersion:  p.PlannerVersion,
		CurriculumID:    p.CurriculumID,
		CurriculumHash:  p.CurriculumHash,
		CurriculumName:  p.CurriculumName,
		CurriculumStart: p.CurriculumStart,
		CurriculumEnd:   p.CurriculumEnd,
		CurriculumTotal: p.CurriculumTotal,
		Theme:           p.Theme,
		ID:              p.ID,
		Created:         p.Created,
		Preferences:     p.Preferences,
		Warnings:        p.Warnings,
	}
	for _, b := range p.Blocks {
		d.Blocks = append(d.Blocks, BlockDefinition{
			Assessment:         b.Assessment,
			Personalization:    b.Personalization,
			Measurement:        b.Measurement,
			CurriculumRow:      b.CurriculumRow,
			CompletedBefore:    b.CompletedBefore,
			AnchorScenario:     b.AnchorScenario,
			Timing:             b.Timing,
			DifficultyEvidence: b.DifficultyEvidence,
			Signature:          b.Signature,
			Scenario:           b.Scenario,
			Role:               b.Role,
			Budget:             b.Budget,
			SourcePlayCount:    b.SourcePlayCount,
			PlayCount:          b.PlayCount,
			Cue:                b.Cue,
			Benchmark:          b.Benchmark,
			InitialTarget:      b.Target, InitialReason: b.Reason})
	}
	data, err := json.Marshal(d)
	if err != nil {
		return nil
	}
	var detached PlanDefinition
	if json.Unmarshal(data, &detached) != nil {
		return nil
	}
	return &detached
}
func executionOf(p *Plan) *ExecutionProgress {
	if p == nil {
		return nil
	}
	e := &ExecutionProgress{PlanID: p.ID,
		EndReason:     p.EndReason,
		EndedAt:       p.EndedAt,
		Status:        p.Status,
		Index:         p.Index,
		Elapsed:       p.Elapsed,
		Recorded:      p.Recorded,
		BlockElapsed:  p.BlockElapsed,
		LastTick:      p.LastTick,
		AcceptAfter:   p.AcceptAfter,
		Seen:          p.Seen,
		RemindedBlock: p.RemindedBlock,
		RemindedEnd:   p.RemindedEnd,
		Reminder:      p.Reminder,
	}
	for _, b := range p.Blocks {
		e.Blocks = append(e.Blocks, BlockExecution{
			ObservationInterrupted: b.ObservationInterrupted,
			Observations:           b.Observations,
			LastCompletedAt:        b.LastCompletedAt,
			Target:                 b.Target,
			Reason:                 b.Reason,
			Recorded:               b.Recorded,
			Runs:                   b.Runs,
			Best:                   b.Best,
			Outcome:                b.Outcome,
		})
	}
	return e
}
func assemblePlan(d *PlanDefinition, e *ExecutionProgress) *Plan {
	if d == nil {
		return nil
	}
	p := &Plan{
		CurriculumCycle: d.CurriculumCycle,
		SelectionReason: d.SelectionReason,
		Progression:     d.Progression,
		TierReason:      d.TierReason,
		PlayerTier:      d.PlayerTier,
		TemplateTier:    d.TemplateTier,
		PlannerVersion:  d.PlannerVersion,
		CurriculumID:    d.CurriculumID,
		CurriculumHash:  d.CurriculumHash,
		CurriculumName:  d.CurriculumName,
		CurriculumStart: d.CurriculumStart,
		CurriculumEnd:   d.CurriculumEnd,
		CurriculumTotal: d.CurriculumTotal,
		Theme:           d.Theme,
		ID:              d.ID,
		Created:         d.Created,
		Preferences:     d.Preferences,
		Warnings:        d.Warnings,
	}
	for _, b := range d.Blocks {
		p.Blocks = append(p.Blocks, Block{
			Assessment:         b.Assessment,
			Personalization:    b.Personalization,
			Measurement:        b.Measurement,
			CurriculumRow:      b.CurriculumRow,
			CompletedBefore:    b.CompletedBefore,
			AnchorScenario:     b.AnchorScenario,
			Timing:             b.Timing,
			DifficultyEvidence: b.DifficultyEvidence,
			Signature:          b.Signature,
			Scenario:           b.Scenario,
			Role:               b.Role,
			Budget:             b.Budget,
			SourcePlayCount:    b.SourcePlayCount,
			PlayCount:          b.PlayCount,
			Cue:                b.Cue,
			Benchmark:          b.Benchmark,
			Target:             b.InitialTarget, Reason: b.InitialReason})
	}
	if e == nil {
		return p
	}
	p.EndReason = e.EndReason
	p.EndedAt = e.EndedAt
	p.Status = e.Status
	p.Index = e.Index
	p.Elapsed = e.Elapsed
	p.Recorded = e.Recorded
	p.BlockElapsed = e.BlockElapsed
	p.LastTick = e.LastTick
	p.AcceptAfter = e.AcceptAfter
	p.Seen = e.Seen
	p.RemindedBlock = e.RemindedBlock
	p.RemindedEnd = e.RemindedEnd
	p.Reminder = e.Reminder
	for i, b := range e.Blocks {
		if i >= len(p.Blocks) {
			break
		}
		p.Blocks[i].ObservationInterrupted = b.ObservationInterrupted
		p.Blocks[i].Observations = b.Observations
		p.Blocks[i].LastCompletedAt = b.LastCompletedAt
		p.Blocks[i].Target = b.Target
		p.Blocks[i].Reason = b.Reason
		p.Blocks[i].Recorded = b.Recorded
		p.Blocks[i].Runs = b.Runs
		p.Blocks[i].Best = b.Best
		p.Blocks[i].Outcome = b.Outcome
	}
	return p
}
