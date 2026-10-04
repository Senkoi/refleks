package training

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"refleks/internal/models"
	"strings"
	"testing"
	"time"
)

func TestPartialOnlineAbilityAndBelowFirstAdvancedDoNotBecomeEntry(t *testing.T) {
	a := levelScenario("A", "static", "Voltaic S5", "Advanced", []string{"Master", "Grandmaster"}, []float64{100, 200})
	b := a
	b.Name = "B"
	b.Benchmarks = append([]BenchmarkMembership(nil), a.Benchmarks...)
	score := 90.0
	a.Benchmarks[0].BenchmarkScore = &score
	levels := PlayerLevels([]Scenario{a, b}, nil, epoch)
	if levels[0].Status != "partial" || levels[0].Rank != "" || levels[0].TrainingTier != "intermediate" || trainingTier("static", levels) != "intermediate" {
		t.Fatal(levels)
	}
	b.Benchmarks = append([]BenchmarkMembership(nil), a.Benchmarks...)
	levels = PlayerLevels([]Scenario{a, b}, nil, epoch)
	if levels[0].Rank != "unranked" || trainingTier("static", levels) == "entry" {
		t.Fatal("below Advanced threshold became entry", levels)
	}
}

func TestTrainingAbilityMedianDoesNotChangeCompleteRank(t *testing.T) {
	a := levelScenario("A", "static", "Voltaic S5", "Intermediate", []string{"Platinum", "Diamond", "Jade"}, []float64{100, 200, 300})
	b := a
	b.Name = "B"
	x, y := 110.0, 310.0
	a.Benchmarks[0].BenchmarkScore = &x
	b.Benchmarks = append([]BenchmarkMembership(nil), a.Benchmarks...)
	b.Benchmarks[0].BenchmarkScore = &y
	levels := PlayerLevels([]Scenario{a, b}, nil, epoch)
	if levels[0].Rank != "Platinum" || levels[0].TrainingTier != "intermediate" {
		t.Fatal("achievement and training level not separated", levels)
	}
}

func schedulingFixture() ([]Curriculum, []Scenario, []models.RunRecord) {
	ts := []Curriculum{}
	catalog := []Scenario{}
	for _, theme := range []string{"static", "reactive"} {
		s := levelScenario(theme, theme, "Voltaic S5", "Novice", []string{"Iron", "Bronze", "Silver", "Gold"}, []float64{10, 20, 30, 40})
		x := 40.0
		if theme == "reactive" {
			x = 20
		}
		s.Benchmarks[0].BenchmarkScore = &x
		catalog = append(catalog, s)
		ts = append(ts, Curriculum{ID: theme, Theme: theme, Tier: "novice", Rows: []CurriculumRow{{Name: theme, Count: 2}}})
	}
	return ts, catalog, nil
}
func TestCategorySchedulingUsesAbilityAndRecentTime(t *testing.T) {
	ts, catalog, runs := schedulingFixture()
	got, err := selectCurriculumWithHistory(ts, catalog, defaults(), runs, epoch, nil)
	if err != nil || got.Theme != "reactive" {
		t.Fatal("weak category not preferred", got, err)
	}
	for i := 0; i < 40; i++ {
		runs = append(runs, record(fmt.Sprint(i), "reactive", 60, 20, epoch.Add(-time.Duration(i+1)*time.Minute)))
	}
	got, err = selectCurriculumWithHistory(ts, catalog, defaults(), runs, epoch, nil)
	if err != nil || got.Theme != "static" {
		t.Fatal("weak category monopolized training after heavy practice", got, err)
	}
}

func TestPartialResumeBudgetAndCompletedLaterRows(t *testing.T) {
	s, tplt := templateFixture(t)
	tplt.Rows = []CurriculumRow{{Name: "Static A", Count: 5}, {Name: "Static B", Count: 5}, {Name: "Static Measure", Count: 5}}
	a, b, c := 0, 1, 2
	p := Plan{PlannerVersion: 6, CurriculumID: tplt.ID, CurriculumTotal: 3, CurriculumEnd: 3, Status: "completed", Blocks: []Block{
		{CurriculumRow: &a, Scenario: Scenario{Name: "Static A"}, Role: "practice", SourcePlayCount: 5, PlayCount: 2, Runs: 2, Recorded: 120, LastCompletedAt: epoch.UnixMilli()},
		{CurriculumRow: &b, Scenario: Scenario{Name: "Static B"}, Role: "practice", SourcePlayCount: 5, PlayCount: 2, Runs: 1, Recorded: 60, LastCompletedAt: epoch.UnixMilli()},
		{CurriculumRow: &c, Scenario: Scenario{Name: "Static Measure"}, Role: "practice", SourcePlayCount: 5, PlayCount: 2, Runs: 2, Recorded: 120, LastCompletedAt: epoch.UnixMilli()},
	}}
	w, start, end, err := curriculumWindow(tplt, s.state.Catalog, nil, defaults(), epoch, []Plan{p})
	if err != nil || start != 1 || end != 3 || len(w.Rows) != 1 || w.Rows[0].Name != "Static B" || w.Rows[0].Count != 1 || w.Rows[0].CompletedBefore != 1 {
		t.Fatal("completed rows repeated or partial count lost", w, start, end, err)
	}
	next, err := GenerateCurriculum(w, s.state.Catalog, nil, defaults(), epoch, rand.New(rand.NewSource(1)), false)
	if err != nil {
		t.Fatal(err)
	}
	next.CurriculumTotal = 3
	next.CurriculumStart = start
	next.CurriculumEnd = end
	next.Status = "completed"
	next.Blocks[0].Runs = 1
	next.Blocks[0].Recorded = 60
	r := routineProgress(tplt, []Plan{p, *next, *next})
	if !r.complete() || r.Rows[1].Completed != 2 {
		t.Fatal("resume snapshots counted incorrectly", r)
	}
	w, start, _, err = curriculumWindow(tplt, s.state.Catalog, nil, defaults(), epoch, []Plan{p, *next})
	if err != nil || start != 0 || len(w.Rows) == 0 {
		t.Fatal("finished routine did not restart", start, err)
	}
}

func TestResumeWithin24HoursOverridesCategoryButExpirationKeepsProgress(t *testing.T) {
	ts, catalog, _ := schedulingFixture()
	for i := range ts {
		ts[i].Rows = append(ts[i].Rows, ts[i].Rows[0])
	}
	saved := map[string]RoutineProgress{progressKey(ts[0]): {Rows: []RowProgress{{Target: 1, Completed: 1}, {}}, LastPracticed: epoch.Add(-23 * time.Hour).UnixMilli()}}
	got, err := selectCurriculumWithHistory(ts, catalog, defaults(), nil, epoch, nil, saved)
	if err != nil || got.ID != "static" {
		t.Fatal("recent continuation lost to weak category", got, err)
	}
	r := saved[progressKey(ts[0])]
	r.LastPracticed = epoch.Add(-25 * time.Hour).UnixMilli()
	saved[progressKey(ts[0])] = r
	got, err = selectCurriculumWithHistory(ts, catalog, defaults(), nil, epoch, nil, saved)
	if err != nil || got.ID != "reactive" {
		t.Fatal("expired continuation still overrides", got, err)
	}
	_, start, _, err := curriculumWindowWithProgress(ts[0], catalog, nil, defaults(), epoch, nil, r)
	if err != nil || start != 1 {
		t.Fatal("expiration erased progress", start, err)
	}
	r.LastPracticed = epoch.Add(-24 * time.Hour).UnixMilli()
	if !resumeDue(r, epoch) {
		t.Fatal("24h boundary")
	}
	r.LastPracticed = epoch.Add(time.Minute).UnixMilli()
	if resumeDue(r, epoch) {
		t.Fatal("future practice accepted")
	}
}

func TestRoutineProgressSurvivesHistoryTrimAndVersionChanges(t *testing.T) {
	s, tplt := templateFixture(t)
	i := 0
	s.state.Plan = &Plan{PlannerVersion: 6, CurriculumID: tplt.ID, CurriculumHash: tplt.ContentSHA256, CurriculumTotal: len(tplt.Rows), Status: "completed", Blocks: []Block{{CurriculumRow: &i, Scenario: Scenario{Name: tplt.Rows[0].Name}, Role: tplt.Rows[0].Role, SourcePlayCount: tplt.Rows[0].Count, PlayCount: 2, Runs: 1, Recorded: 60, LastCompletedAt: epoch.UnixMilli()}}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	reopened.state.History = nil
	reopened.state.Plan = nil
	r := progressFor(tplt, nil, reopened.state.CurriculumProgress)
	if r.Rows[0].Completed != 1 || !resumeDue(r, epoch) {
		t.Fatal("durable partial progress lost", r)
	}
	changed := tplt
	changed.ContentSHA256 = "new-version"
	if progressFor(changed, nil, reopened.state.CurriculumProgress).LastPracticed != 0 {
		t.Fatal("changed template inherited old cursor")
	}
}

func TestWorkbenchCachesEvaluationAndClockPayloadExcludesCatalog(t *testing.T) {
	s, _ := templateFixture(t)
	s.state.Catalog[0].LocalAssessment = &LocalAssessment{Fields: []SCEField{{Key: "PlayerProfile", Raw: "parser-only"}}}
	if _, err := s.WorkbenchJSON(nil); err != nil {
		t.Fatal(err)
	}
	first := s.evaluatedAt
	firstKey := s.evaluationKey
	s.state.History = make([]Plan, 100)
	_, _ = s.WorkbenchJSON(nil)
	if !s.evaluatedAt.Equal(first) {
		t.Fatal("unchanged evidence recalculated")
	}
	full, _ := s.WorkbenchJSON(nil)
	var state State
	_ = json.Unmarshal([]byte(full), &state)
	if len(state.History) != 0 || len(state.Curricula[0].Rows) != 0 {
		t.Fatal("unused historical data sent to UI")
	}
	if strings.Contains(full, "parser-only") || len(s.state.Catalog[0].LocalAssessment.Fields) != 1 {
		t.Fatal("raw fields transmitted or backend evidence mutated")
	}
	live, _ := s.LiveJSON()
	if strings.Contains(live, "catalog") || strings.Contains(live, "measurements") {
		t.Fatal("clock poll sent catalog")
	}
	runs := levelRuns("Static A", []float64{1, 2, 3}, time.Now())
	if _, err := s.WorkbenchJSON(runs); err != nil {
		t.Fatal(err)
	}
	// Consecutive refreshes may share a clock tick on Windows. The cache key
	// proves that changed evidence was evaluated without requiring time to pass.
	if s.evaluationKey == firstKey || s.evaluationKey != evaluationFingerprint(runs, s.dataRevision) {
		t.Fatal("new evidence did not invalidate")
	}
}

func TestClockCheckpointBatchesButRecordedRunsSaveImmediately(t *testing.T) {
	s, _ := templateFixture(t)
	if _, err := s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	_, _ = s.Action("start", epoch, nil)
	s.Tick(epoch.Add(2*time.Second), nil)
	read := func() State {
		data, err := os.ReadFile(s.path)
		if err != nil {
			t.Fatal(err)
		}
		var st State
		_ = json.Unmarshal(data, &st)
		return st
	}
	first := read().Plan.Elapsed
	s.Tick(epoch.Add(4*time.Second), nil)
	if read().Plan.Elapsed != first {
		t.Fatal("clock wrote entire state every tick")
	}
	run := record("completed", s.state.Plan.Blocks[0].Scenario.Name, 60, 10, epoch.Add(60*time.Second))
	s.Tick(epoch.Add(60*time.Second), []models.RunRecord{run})
	if read().Plan.Recorded != 60 {
		t.Fatal("completed run was not immediately persisted")
	}
}

func TestRecordedPartialSessionGeneratesOnlyRemainingRuns(t *testing.T) {
	s, _ := templateFixture(t)
	pref := defaults()
	pref.Minutes = 5
	first, err := s.Generate(pref, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-4 * time.Minute)
	if _, err = s.Action("start", start, nil); err != nil {
		t.Fatal(err)
	}
	runs := []models.RunRecord{
		record("a1", first.Blocks[0].Scenario.Name, 60, 10, start.Add(time.Minute)),
		record("a2", first.Blocks[0].Scenario.Name, 60, 10, start.Add(2*time.Minute)),
		record("b1", first.Blocks[1].Scenario.Name, 60, 10, start.Add(3*time.Minute)),
	}
	s.Tick(start.Add(3*time.Minute), runs)
	if _, err = s.Action("finish", start.Add(3*time.Minute), runs); err != nil {
		t.Fatal(err)
	}
	next, err := s.Generate(pref, runs)
	if err != nil {
		t.Fatal(err)
	}
	if next.CurriculumStart != 1 || next.Blocks[0].Scenario.Name != first.Blocks[1].Scenario.Name || next.Blocks[0].PlayCount != 1 || next.Blocks[0].CompletedBefore != 1 || !strings.Contains(next.SelectionReason, "24 小时") {
		t.Fatal("actual partial records did not resume correctly", next)
	}
	data, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Rows []CurriculumRow `json:"scenarioList"`
	}
	if err = json.Unmarshal(data, &exported); err != nil || exported.Rows[0].Count != 1 {
		t.Fatal("export repeated completed run", err)
	}
}

func BenchmarkWorkbenchClockPayload(b *testing.B) {
	s, _ := New(b.TempDir())
	for i := 0; i < 2000; i++ {
		s.state.Catalog = append(s.state.Catalog, Scenario{Name: fmt.Sprint(i), Skill: "static", Enabled: true})
	}
	s.state.Plan = &Plan{ID: "running", Status: "running", Blocks: []Block{{Scenario: s.state.Catalog[0]}}}
	b.Run("full_cached", func(b *testing.B) {
		_, _ = s.WorkbenchJSON(nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = s.WorkbenchJSON(nil)
		}
	})
	b.Run("clock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = s.LiveJSON()
		}
	})
}
