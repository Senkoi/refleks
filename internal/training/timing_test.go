package training

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"refleks/internal/models"
)

func TestTimingUsesComparableHistoryAndIgnoresFutureRecords(t *testing.T) {
	runs := []models.RunRecord{}
	for i, duration := range []float64{30, 31, 900} {
		runs = append(runs, record(fmt.Sprint(i), "A", duration, 80, epoch.Add(-time.Duration(i+1)*time.Hour)))
	}
	old := record("old", "A", 120, 80, epoch.Add(-4*time.Hour))
	old.Stats.Summary.Hash = "old-version"
	future := record("future", "A", 500, 80, epoch.Add(time.Hour))
	future.Stats.Summary.Hash = "future-version"
	runs = append(runs, old, future)
	est := estimateTiming(Scenario{Name: "A", Seconds: 60}, observed(runs)["a"], epoch)
	if est.Source != "history" || est.Seconds != 31 || est.Samples != 3 {
		t.Fatalf("mixed versions or future records: %+v", est)
	}
	if got := estimateTiming(Scenario{Seconds: 45}, nil, epoch); got.Seconds != 45 || got.Source != "catalog" {
		t.Fatalf("did not use known catalog estimate: %+v", got)
	}
}

func TestTimingBudgetUsesWholeRunsAndMoreScenarios(t *testing.T) {
	pool := []Scenario{}
	for i := 0; i < 15; i++ {
		name := fmt.Sprintf("Scenario %d", i)
		pool = append(pool, Scenario{Name: name, Skill: "smooth", Family: name, Seconds: 60, Enabled: true})
	}
	p := defaults()
	p.Variety = 0
	plan, err := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(4)))
	if err != nil { t.Fatal(err) }
	total := 0
	for _, block := range plan.Blocks {
		if block.PlayCount < 1 || block.PlayCount > 3 || block.Budget != block.Timing.Seconds*block.PlayCount {
			t.Fatalf("invalid duration allocation: %+v", block)
		}
		total += block.Budget
	}
	if len(plan.Blocks) < 8 || total > 27*60 {
		t.Fatalf("unexpected budget/scenario count: %d / %d", total, len(plan.Blocks))
	}
	if plannedRepetitions(TimingEstimate{Seconds: 60, RecentSeconds: 180}, "practice") >= plannedRepetitions(TimingEstimate{Seconds: 60}, "practice") {
		t.Fatal("recent practice did not reduce repetition")
	}
}

func TestExtraRunsStayWithLastConfirmedScenario(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.ExecutionMode = "playlist"
	s.state.Plan.Preferences.Minutes = 10
	s.state.Plan.Blocks[0].PlayCount = 1
	s.state.Plan.Blocks[1].PlayCount = 2
	_, _ = s.Action("start", epoch, nil)
	for i := 1; i <= 3; i++ {
		end := epoch.Add(time.Duration(i) * time.Minute)
		s.Tick(end, []models.RunRecord{record(fmt.Sprint(i), "Smooth", 60, 70, end)})
	}
	p := s.Snapshot(nil).Plan
	if p.Index != 0 || p.Recorded != 180 || p.BlockElapsed != 180 || p.Blocks[0].Runs != 3 {
		t.Fatalf("extra runs moved to the next module: %+v", p)
	}
	s.Tick(epoch.Add(242*time.Second), []models.RunRecord{record("next", "Benchmark", 60, 40, epoch.Add(240*time.Second))})
	p = s.Snapshot(nil).Plan
	if p.Index != 1 || p.BlockElapsed != 62 || p.Blocks[0].Outcome != "list_complete" || p.Recorded != 240 {
		t.Fatalf("next module start was not reconstructed: %+v", p)
	}
}

func TestPlaylistSkipKeepsCollectingRuns(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.ExecutionMode = "playlist"
	s.state.Plan.Blocks[1].PlayCount = 2
	_, _ = s.Action("start", epoch, nil)
	_, _ = s.Action("next", epoch.Add(10*time.Second), nil)
	s.Tick(epoch.Add(70*time.Second), []models.RunRecord{record("next", "Benchmark", 60, 40, epoch.Add(70*time.Second))})
	p := s.Snapshot(nil).Plan
	if p.Status != "running" || p.Blocks[1].Runs != 1 || p.BlockElapsed != 60 {
		t.Fatalf("skip stalled collection: %+v", p)
	}
}
