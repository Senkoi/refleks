package training

import (
	"aimmeow/internal/models"
	"aimmeow/internal/practice"
	"reflect"
	"testing"
	"time"
)

func TestAssessmentUsesHistorySessionIdentityAndUserGap(t *testing.T) {
	now := epoch.Add(time.Hour)
	runs := []models.RunRecord{record("a", "map", 60, 0, epoch), record("b", "map", 120, 100, epoch.Add(21*time.Minute))}
	groups := practice.Group(practice.Entries(runs), 20*time.Minute, now)
	ctx := sessionPositions(nil, runs, 20*time.Minute, now)
	if len(groups) != 1 || ctx[runKey(runs[0])].SessionID != groups[0].ID || ctx[runKey(runs[1])].SessionID != groups[0].ID || ctx[runKey(runs[1])].PriorSeconds != 60 {
		t.Fatal(groups, ctx)
	}
	ctx = sessionPositions(nil, runs, 10*time.Minute, now)
	if ctx[runKey(runs[0])].SessionID == ctx[runKey(runs[1])].SessionID {
		t.Fatal("user gap was ignored")
	}
}
func TestEndReasonsAndActualCompletionSurviveReload(t *testing.T) {
	for _, tc := range []struct {
		action, reason string
		runs           int
	}{{"finish", "manual", 0}, {"next", "items_processed", 0}, {"next", "plan_complete", 2}} {
		t.Run(tc.reason, func(t *testing.T) {
			dir := t.TempDir()
			s, err := New(dir)
			if err != nil {
				t.Fatal(err)
			}
			s.state.Plan = &Plan{ID: "p", Status: "running", Preferences: Preferences{Minutes: 30, ExecutionMode: "playlist"}, Blocks: []Block{{PlayCount: 2, Runs: tc.runs, Recorded: float64(tc.runs * 60), Outcome: "pending"}}}
			if _, err = s.Action(tc.action, epoch, nil); err != nil {
				t.Fatal(err)
			}
			if s.state.Plan.EndReason != tc.reason || s.state.Plan.EndedAt != epoch.UnixMilli() {
				t.Fatal(s.state.Plan)
			}
			loaded, err := New(dir)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.state.Plan.EndReason != tc.reason || !reflect.DeepEqual(definePlan(s.state.Plan), definePlan(loaded.state.Plan)) {
				t.Fatal("definition/end reason changed on reload")
			}
			progress := loaded.Progress().Current
			expected := 0
			if tc.runs == 2 {
				expected = 1
			}
			if progress.CompletedBlocks != expected {
				t.Fatal("skip counted as complete", progress)
			}
		})
	}
}
func TestTimeoutDoesNotCompleteUnplayedItemsOrLoseLateRecord(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Plan = &Plan{ID: "p", Status: "running", LastTick: epoch.UnixMilli(), AcceptAfter: epoch.UnixMilli(), Preferences: Preferences{Minutes: 1, ExecutionMode: "playlist"}, Blocks: []Block{{Scenario: Scenario{Name: "map"}, PlayCount: 2, Outcome: "pending"}}}
	deadline := epoch.Add(time.Minute)
	s.Tick(deadline, nil)
	if s.state.Plan.EndReason != "time_budget" || s.Progress().Current.CompletedBlocks != 0 {
		t.Fatal(s.state.Plan)
	}
	s.Tick(deadline.Add(time.Second), []models.RunRecord{record("late", "map", 60, 10, deadline)})
	p := s.Progress().Current
	if p.Runs != 1 || p.Recorded != 60 || p.CompletedBlocks != 0 || p.EndReason != "time_budget" {
		t.Fatal("late completed CSV attribution was lost", p)
	}
	if !reflect.DeepEqual(s.PlanRunIDs("p"), []string{"late"}) {
		t.Fatal(s.state.RunContexts)
	}
}
func TestLegacyEndDoesNotClaimCompletion(t *testing.T) {
	p := &Plan{Status: "completed", Preferences: Preferences{ExecutionMode: "playlist"}, Blocks: []Block{{PlayCount: 2, Outcome: "skipped"}}}
	normalizePlanEnd(p)
	if p.EndReason != "legacy_unknown" || planSummary(*p).CompletedBlocks != 0 {
		t.Fatal(p)
	}
}
func TestCompletionAtBudgetKeepsItsActualReason(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Plan = &Plan{ID: "p", Status: "running", LastTick: epoch.UnixMilli(), AcceptAfter: epoch.UnixMilli(), Preferences: Preferences{Minutes: 1, ExecutionMode: "playlist"}, Blocks: []Block{{Scenario: Scenario{Name: "map"}, PlayCount: 1, Outcome: "pending"}}}
	s.Tick(epoch.Add(61*time.Second), []models.RunRecord{record("done", "map", 60, 100, epoch.Add(time.Minute))})
	if s.state.Plan.EndReason != "plan_complete" || s.state.Plan.EndedAt != epoch.Add(time.Minute).UnixMilli() {
		t.Fatal(s.state.Plan)
	}
}
func TestOldPausedPlanTargetsRemainCompatible(t *testing.T) {
	for _, changed := range []bool{false, true} {
		s, err := New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r := record("resumed", "map", 60, 50, epoch.Add(time.Minute))
		s.state.Plan = &Plan{ID: "p", Status: "paused", Preferences: Preferences{Minutes: 30, ExecutionMode: "threshold"}, Blocks: []Block{{Scenario: Scenario{Name: "map"}, Budget: 600, Target: 100, Signature: legacySignature(r.Stats.Summary), Outcome: "pending"}}}
		if changed {
			r.Stats.Summary.FOV = 90
		}
		if _, err = s.Action("start", epoch, nil); err != nil {
			t.Fatal(err)
		}
		s.Tick(epoch.Add(time.Minute), []models.RunRecord{r})
		b := s.state.Plan.Blocks[0]
		if b.Runs != 1 || (!changed && b.Target != 100) || (changed && b.Target != 0) {
			t.Fatalf("legacy target compatibility changed=%v: %+v", changed, b)
		}
	}
}
