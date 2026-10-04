package training

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"aimmeow/internal/models"
	"testing"
	"time"
)

func TestBundledOfficialInitializationAndThirtyMinuteContinuation(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InitializeTraining(); err != nil {
		t.Fatal(err)
	}
	if len(s.state.Curricula) != 36 {
		t.Fatalf("expected 36 real playlists: %d", len(s.state.Curricula))
	}
	p, err := s.Generate(defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Blocks) == 0 || p.CurriculumTotal <= p.CurriculumEnd {
		t.Fatal("30 minute window wasn't selected", p)
	}
	for _, b := range p.Blocks {
		if b.Role == "explore" {
			t.Fatal("explored before baseline")
		}
	}
	used := 0
	for _, b := range p.Blocks {
		used += b.Budget
	}
	if used > 27*60 {
		t.Fatal("exceeded budget")
	}
	for i := range s.state.Plan.Blocks {
		b := &s.state.Plan.Blocks[i]
		b.Runs = b.PlayCount
		b.Recorded = float64(b.Budget)
		b.Outcome = "list_complete"
	}
	s.state.Plan.Status = "completed"
	prefs := defaults()
	prefs.Focus = p.Theme
	if p.Theme == "switching_speed" || p.Theme == "switching_evasive" {
		prefs.Focus = "switching"
	}
	next, err := s.Generate(prefs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.CurriculumID != p.CurriculumID || next.CurriculumStart != p.CurriculumEnd {
		t.Fatal("did not continue recorded window", next.CurriculumStart, p.CurriculumEnd)
	}
	// Idempotent initialization and cache reopening never duplicate the bundle.
	n := len(s.state.Catalog)
	if err = s.InitializeTraining(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.state.Curricula) != 36 || len(reopened.state.Catalog) != n {
		t.Fatal("duplicate initialization")
	}
}

func levelScenario(name, theme, system, tier string, ranks []string, thresholds []float64) Scenario {
	skill := theme
	if theme == "switching_speed" || theme == "switching_evasive" {
		skill = "switching"
	}
	return Scenario{Name: name, Skill: skill, Technique: theme, Enabled: true, Seconds: 60, Classification: "benchmark", Difficulty: "unknown", Benchmarks: []BenchmarkMembership{{Name: system + " / " + tier, System: system, NativeDifficulty: tier, Ranks: ranks, Thresholds: thresholds}}}
}
func levelRuns(name string, scores []float64, now time.Time) []models.RunRecord {
	runs := []models.RunRecord{}
	for i, v := range scores {
		runs = append(runs, record(fmt.Sprintf("%s-%d-%g", name, i, v), name, 60, v, now.Add(-time.Duration(i+1)*time.Hour)))
	}
	return runs
}
func TestPlayerTierNeedsCategoryCoverageAndChangesWithRecentScores(t *testing.T) {
	catalog := []Scenario{levelScenario("A", "static", "Voltaic S5", "Novice", []string{"Iron", "Bronze", "Silver", "Gold"}, []float64{10, 20, 30, 40}), levelScenario("B", "static", "Voltaic S5", "Novice", []string{"Iron", "Bronze", "Silver", "Gold"}, []float64{10, 20, 30, 40})}
	runs := levelRuns("A", []float64{50, 50, 50}, epoch)
	l := PlayerLevels(catalog, runs, epoch)
	if len(l) != 1 || l[0].Status != "partial" || l[0].Rank != "" || inferredTier("static", l) != "novice" {
		t.Fatal("one scene promoted category", l)
	}
	runs = append(runs, levelRuns("B", []float64{50, 50, 50}, epoch)...)
	l = PlayerLevels(catalog, runs, epoch)
	if l[0].Rank != "Gold" || inferredTier("static", l) != "adept" {
		t.Fatal("rank/tier not inferred", l)
	}
	// Independent weak category must not inherit static tier.
	if inferredTier("reactive", l) != "novice" {
		t.Fatal("cross-category leakage")
	}
	for i := range runs {
		runs[i].Stats.Summary.DatePlayed = epoch.Add(-15*24*time.Hour - time.Duration(i)*time.Hour).Format(time.RFC3339)
	}
	runs = append(runs, levelRuns("A", []float64{15, 15, 15}, epoch)...)
	runs = append(runs, levelRuns("B", []float64{15, 15, 15}, epoch)...)
	l = PlayerLevels(catalog, runs, epoch)
	if l[0].Rank != "Iron" || inferredTier("static", l) != "entry" {
		t.Fatal("historical PB prevented dynamic downgrade", l)
	}
	for i := range runs {
		runs[i].Stats.Summary.AvgTargetScale = 2
	}
	if PlayerLevels(catalog, runs, epoch)[0].Status != "insufficient" {
		t.Fatal("altered target scale became benchmark evidence")
	}
}

func TestAutomaticReferencesKeepAvailableNoviceAndNewestAdvanced(t *testing.T) {
	c := []Scenario{levelScenario("N", "static", "Voltaic S5", "Novice", []string{"Gold"}, []float64{10}), levelScenario("A", "static", "Voltaic S5", "Advanced", []string{"Nova"}, []float64{10}), levelScenario("AA", "static", "Voltaic S5.5", "Advanced", []string{"Nova"}, []float64{10})}
	p := automaticReferences(c, Preferences{Benchmark: "old", Benchmarks: []string{"old"}})
	if !selectedBenchmark(p, "Voltaic S5 / Novice") || !selectedBenchmark(p, "Voltaic S5.5 / Advanced") || selectedBenchmark(p, "Voltaic S5 / Advanced") {
		t.Fatal("latest-only discarded beginner references", p)
	}
}

func TestBenchmarkScenesAreAlignedExplorationVariants(t *testing.T) {
	base := demandScene("base", "precision")
	base.Classification = "manual"
	matched := demandScene("aligned", "precision")
	matched.Classification = "benchmark"
	matched.Benchmarks = []BenchmarkMembership{{Name: "Voltaic S5 / Novice", System: "Voltaic S5", NativeDifficulty: "Novice", Thresholds: []float64{100}}}
	tooHard := matched
	tooHard.Name = "unmatched"
	tooHard.Benchmarks = []BenchmarkMembership{{Name: "Voltaic S5.5 / Advanced", System: "Voltaic S5.5", NativeDifficulty: "Advanced", Thresholds: []float64{1000}}}
	p := defaults()
	p.Minutes = 15
	p.Variety = .5
	plan, err := GenerateCurriculum(Curriculum{ID: "t", Rows: []CurriculumRow{{Name: "base", Count: 1}}}, []Scenario{base, tooHard, matched}, nil, p, epoch, rand.New(rand.NewSource(1)), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Blocks) != 2 || plan.Blocks[1].Scenario.Name != "aligned" || plan.Blocks[1].Role != "explore" || plan.Blocks[1].Benchmark != "Voltaic S5 / Novice" {
		t.Fatal("benchmark wasn't an aligned exploration variant", plan.Blocks)
	}
}

func TestAlreadyDownloadedSceneOutsideCatalogIsEvaluated(t *testing.T) {
	s, _ := New(t.TempDir())
	root := t.TempDir()
	path := filepath.Join(root, "x.sce")
	if err := os.WriteFile(path, []byte(localSCEFixture), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, epoch.Add(-time.Minute), epoch.Add(-time.Minute))
	s.PollLocal([]string{root}, "", epoch)
	s.PollLocal([]string{root}, "", epoch.Add(15*time.Second))
	if len(s.state.Catalog) != 1 || s.state.Catalog[0].LocalAssessment == nil || len(s.state.Catalog[0].LocalAssessment.Measurements) == 0 {
		t.Fatal("downloaded scene not imported/evaluated", s.state.Catalog)
	}
	if s.state.Catalog[0].Difficulty != "unknown" {
		t.Fatal("invented calibrated total difficulty")
	}
}

func TestRecordedWindowsUnlockExplorationOnlyAfterWholeBaseline(t *testing.T) {
	tplt := Curriculum{ID: "window", Rows: []CurriculumRow{{Name: "A", Count: 2}, {Name: "B", Count: 3}}}
	window := func(start, end int) Plan {
		blocks := []Block{}
		for _, r := range tplt.Rows[start:end] {
			blocks = append(blocks, Block{Scenario: Scenario{Name: r.Name}, Role: "practice", PlayCount: r.Count, Runs: r.Count, Recorded: 60, Outcome: "list_complete"})
		}
		return Plan{CurriculumID: tplt.ID, CurriculumStart: start, CurriculumEnd: end, CurriculumTotal: 2, Status: "completed", Blocks: blocks}
	}
	a, b := window(0, 1), window(1, 2)
	if curriculumBaseline(tplt, []Plan{a}) {
		t.Fatal("partial baseline unlocked exploration")
	}
	if !curriculumBaseline(tplt, []Plan{a, b}) {
		t.Fatal("recorded windows never unlocked exploration")
	}
	b.Blocks[0].Outcome = "skipped"
	if curriculumBaseline(tplt, []Plan{a, b}) {
		t.Fatal("skipped row counted as baseline")
	}
	b = window(1, 2)
	tplt.Rows[0].Count = 3
	if curriculumBaseline(tplt, []Plan{a, b}) {
		t.Fatal("edited original template reused obsolete baseline")
	}
}

func TestPrecisionFeatureRequiresAuditedFileHash(t *testing.T) {
	var pairs []struct {
		A, B, HashA, HashB, Profile string
		Ratio                       float64
	}
	if err := json.Unmarshal(precisionReferences, &pairs); err != nil {
		t.Fatal(err)
	}
	if len(pairs) == 0 {
		t.Fatal("precision references missing")
	}
	a := &LocalAssessment{Status: "file_parsed_model_unfitted", FileSHA256: pairs[0].HashB}
	calculateFileEvidence(a)
	if len(a.PrecisionComparisons) == 0 || a.PrecisionComparisons[0].PrecisionDelta <= 0 {
		t.Fatal("audited smaller radius not calculated", a)
	}
	a.FileSHA256 = "new-revision"
	calculateFileEvidence(a)
	if len(a.PrecisionComparisons) != 0 {
		t.Fatal("name-only comparison reused stale file evidence")
	}
}
