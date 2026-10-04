package training

import (
	"strings"
	"time"
)

type RowProgress struct {
	Target    int `json:"target"`
	Completed int `json:"completed"`
}
type RoutineProgress struct {
	Cycle         int           `json:"cycle"`
	Rows          []RowProgress `json:"rows"`
	LastPracticed int64         `json:"lastPracticed"`
	EverCompleted bool          `json:"everCompleted"`
}

func progressKey(t Curriculum) string { return t.ID + "|" + t.ContentSHA256 }
func (r RoutineProgress) complete() bool {
	if len(r.Rows) == 0 {
		return false
	}
	for _, row := range r.Rows {
		if row.Target <= 0 || row.Completed < row.Target {
			return false
		}
	}
	return true
}
func matchingRoutine(t Curriculum, p Plan) bool {
	return p.CurriculumID == t.ID && p.CurriculumTotal == len(t.Rows) && (p.CurriculumHash == "" || p.CurriculumHash == t.ContentSHA256)
}

// Counts are absolute within a cycle; replaying a snapshot cannot count runs twice.
func applyRoutinePlan(t Curriculum, r RoutineProgress, p Plan) RoutineProgress {
	if !matchingRoutine(t, p) || p.Status == "draft" {
		return r
	}
	if len(r.Rows) != len(t.Rows) {
		r.Rows = make([]RowProgress, len(t.Rows))
	}
	if p.CurriculumCycle < r.Cycle {
		return r
	}
	if p.CurriculumCycle > r.Cycle {
		r.Cycle = p.CurriculumCycle
		r.Rows = make([]RowProgress, len(t.Rows))
	}
	for j, b := range p.Blocks {
		i := p.CurriculumStart + j
		if b.CurriculumRow != nil {
			i = *b.CurriculumRow
		}
		if i < 0 || i >= len(t.Rows) || b.Role == "explore" || b.Role == "challenge" {
			continue
		}
		row := t.Rows[i]
		role := row.Role
		if role == "" {
			role = "practice"
		}
		target := b.CompletedBefore + b.PlayCount
		if !strings.EqualFold(row.Name, b.Scenario.Name) || role != b.Role || b.PlayCount <= 0 || target > originalRowCount(row) || b.SourcePlayCount == 0 && target != originalRowCount(row) || b.SourcePlayCount > 0 && b.SourcePlayCount != originalRowCount(row) {
			continue
		}
		// Generation defines the short-set target; skipping never completes it.
		if r.Rows[i].Target == 0 {
			r.Rows[i].Target = target
		}
		if b.Recorded > 0 && b.Runs > 0 && !(b.CurriculumRow == nil && (b.Outcome == "skipped" || b.Outcome == "missed")) {
			r.Rows[i].Completed = max(r.Rows[i].Completed, min(r.Rows[i].Target, b.CompletedBefore+min(b.Runs, b.PlayCount)))
			at := b.LastCompletedAt
			if at == 0 {
				at = p.EndedAt
			} // Existing plans have no per-run timestamp.
			r.LastPracticed = max(r.LastPracticed, at)
		}
	}
	r.EverCompleted = r.EverCompleted || r.complete()
	return r
}
func routineProgress(t Curriculum, history []Plan) RoutineProgress {
	r := RoutineProgress{Rows: make([]RowProgress, len(t.Rows))}
	for _, p := range history {
		// Legacy plans had no cycle ID. Detect a recorded restart after full coverage.
		if p.PlannerVersion < 6 && r.complete() && p.CurriculumStart == 0 && p.Status != "draft" && matchingRoutine(t, p) {
			r.Rows = make([]RowProgress, len(t.Rows))
		}
		r = applyRoutinePlan(t, r, p)
	}
	return r
}
func (s *Service) syncProgressLocked() {
	if s.state.CurriculumProgress == nil {
		s.state.CurriculumProgress = map[string]RoutineProgress{}
	}
	for _, t := range s.state.Curricula {
		key := progressKey(t)
		r, ok := s.state.CurriculumProgress[key]
		if !ok {
			r = routineProgress(t, s.state.History)
		}
		if p := s.state.Plan; p != nil {
			r = applyRoutinePlan(t, r, *p)
		}
		s.state.CurriculumProgress[key] = r
	}
}
func progressFor(t Curriculum, history []Plan, saved map[string]RoutineProgress) RoutineProgress {
	if r, ok := saved[progressKey(t)]; ok && len(r.Rows) == len(t.Rows) {
		return r
	}
	return routineProgress(t, history)
}
func resumeDue(r RoutineProgress, now time.Time) bool {
	return !r.complete() && r.LastPracticed > 0 && r.LastPracticed <= now.UnixMilli() && now.UnixMilli()-r.LastPracticed <= int64(24*time.Hour/time.Millisecond)
}
