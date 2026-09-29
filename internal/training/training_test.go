package training

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"refleks/internal/models"
)

var epoch = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func record(id, name string, seconds, score float64, end time.Time) models.RunRecord {
	return models.RunRecord{FilePath: id, Stats: models.RunStatsData{Summary: models.RunStatsSummary{Scenario: name, DatePlayed: end.Format(time.RFC3339), Duration: seconds, Score: score, Hash: "v1", HorizSens: 1, FOV: 103}}}
}

func fixture(t *testing.T) *Service {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Plan = &Plan{ID: "test", Status: "draft", Preferences: Preferences{Minutes: 5}, Blocks: []Block{
		{Scenario: Scenario{Name: "Smooth", Seconds: 60}, Role: "practice", Budget: 180, Target: 90, Outcome: "pending"},
		{Scenario: Scenario{Name: "Benchmark", Seconds: 60}, Role: "benchmark", Budget: 90, Outcome: "pending"},
	}, Seen: []string{}}
	return s
}

func TestPlaylistEvidenceAndManualCorrection(t *testing.T) {
	b := []byte(`{"playlistName":"Author routine","scenarioList":[{"scenarioName":"Smoothbot Easy"},{"scenarioName":"Smoothbot Easy"},{"scenarioName":"Mystery"}]}`)
	ss, err := ParsePlaylist(b, Source{URL: "https://example.com/playlist.json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ss) != 2 || ss[1].Enabled || ss[0].Sources[0].Title != "Author routine" {
		t.Fatalf("invalid extraction: %+v", ss)
	}
	ss[0].Skill = "reactive"
	ss[0].Classification = "manual"
	ss[0].Enabled = false
	ss[0].Preference = "liked"
	fresh, _ := ParsePlaylist(b, Source{URL: "https://example.com/playlist.json"})
	merged := mergeCatalog(ss, fresh)
	if len(merged) != 2 || merged[0].Skill != "reactive" || merged[0].Enabled || len(merged[0].Sources) != 1 || merged[0].Preference != "liked" {
		t.Fatal("refresh overwrote user curation or duplicated source")
	}
	if _, err = ParsePlaylist([]byte(`{"text":"play Smoothbot"}`), Source{}); err == nil {
		t.Fatal("invented a playlist from prose")
	}
}

func TestBudgetBenchmarkReservationAndNoDuplicates(t *testing.T) {
	pool := []Scenario{}
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		pool = append(pool, Scenario{Name: name, Skill: "smooth", Family: name, Seconds: 60, Enabled: true, Difficulty: "novice"})
	}
	pool = append(pool, Scenario{Name: "BM", Skill: "smooth", Family: "bm", Seconds: 60, Enabled: true, Difficulty: "novice", Benchmark: "Test / Novice"})
	p := defaults()
	p.Minutes = 15
	p.Benchmark = "Test / Novice"
	p.Difficulty = "novice"
	plan, err := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	total := 0
	for _, b := range plan.Blocks {
		if used[b.Scenario.Name] {
			t.Fatal("duplicate")
		}
		used[b.Scenario.Name] = true
		total += b.Budget
		if b.Target != 0 {
			t.Fatal("new scenario received threshold")
		}
	}
	if total > p.Minutes*60 || plan.Blocks[len(plan.Blocks)-1].Role != "benchmark" || plan.Blocks[len(plan.Blocks)-1].Scenario.Name != "BM" {
		t.Fatalf("bad plan: %+v", plan)
	}
	plan2, _ := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(7)))
	if !reflect.DeepEqual(plan, plan2) {
		t.Fatal("seeded planning is not reproducible")
	}
}

func TestThresholdRequiresComparableHistory(t *testing.T) {
	runs := []models.RunRecord{}
	for i := 0; i < 6; i++ {
		r := record(string(rune('a'+i)), "A", 60, 100, epoch.Add(-time.Duration(i+1)*time.Hour))
		if i > 2 {
			r.Stats.Summary.Hash = "old"
		}
		runs = append(runs, r)
	}
	if len(comparable(observed(runs)["a"])) != 3 {
		t.Fatal("mixed versions")
	}
}

func TestDuplicateAndUnrelatedRunsCannotAdvance(t *testing.T) {
	s := fixture(t)
	old := record("old", "Smooth", 60, 100, epoch.Add(-time.Hour))
	if _, err := s.Action("start", epoch, []models.RunRecord{old}); err != nil {
		t.Fatal(err)
	}
	r := record("new", "Smooth", 60, 80, epoch.Add(time.Minute))
	other := record("other", "Other", 60, 999, epoch.Add(time.Minute))
	s.Tick(epoch.Add(time.Minute), []models.RunRecord{old, r, r, other})
	s.Tick(epoch.Add(62*time.Second), []models.RunRecord{r, r})
	p := s.Snapshot(nil).Plan
	if p.Recorded != 60 || p.Blocks[0].Runs != 1 || p.Index != 0 {
		t.Fatalf("incorrect attribution %+v", p)
	}
}

func TestThresholdAdvancesAndBenchmarkRecordsLowScore(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.AutoAdvance = true
	_, _ = s.Action("start", epoch, nil)
	name := s.Tick(epoch.Add(time.Minute), []models.RunRecord{record("one", "Smooth", 60, 95, epoch.Add(time.Minute))})
	if name != "Benchmark" {
		t.Fatalf("expected auto next, got %s", name)
	}
	s.Tick(epoch.Add(2*time.Minute), []models.RunRecord{record("two", "Benchmark", 60, 1, epoch.Add(2*time.Minute))})
	p := s.Snapshot(nil).Plan
	if p.Status != "completed" || p.Recorded != 120 || p.Blocks[1].Outcome != "measured" {
		t.Fatalf("measurement was score-filtered: %+v", p)
	}
}

func TestTimeoutDoesNotLaunchOverLiveRun(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.AutoAdvance = true
	_, _ = s.Action("start", epoch, nil)
	if name := s.Tick(epoch.Add(181*time.Second), nil); name != "" {
		t.Fatal("launched over an unknown active run")
	}
	p := s.Snapshot(nil).Plan
	if p.Status != "waiting" || p.Index != 0 {
		t.Fatalf("cap not enforced %+v", p)
	}
	// Waiting for the next module still consumes the overall wall-time budget.
	s.Tick(epoch.Add(301*time.Second), nil)
	if s.Snapshot(nil).Plan.Status != "completed" {
		t.Fatal("total budget not enforced while waiting")
	}
}

func TestPauseResumeAndRestart(t *testing.T) {
	s := fixture(t)
	_, _ = s.Action("start", epoch, nil)
	_, _ = s.Action("pause", epoch.Add(30*time.Second), nil)
	s.Tick(epoch.Add(10*time.Minute), nil)
	if s.Snapshot(nil).Plan.Elapsed != 30 {
		t.Fatal("paused time was counted")
	}
	_, _ = s.Action("start", epoch.Add(10*time.Minute), nil)
	s.Tick(epoch.Add(11*time.Minute), []models.RunRecord{record("paused", "Smooth", 60, 100, epoch.Add(5*time.Minute))})
	if s.Snapshot(nil).Plan.Recorded != 0 {
		t.Fatal("paused run was attributed")
	}
	reloaded, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Snapshot(nil).Plan.Status != "paused" {
		t.Fatal("relaunch silently resumed")
	}
}

func TestFinalCompletedRunAtSessionDeadlineIsCounted(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.Minutes = 1
	s.state.Plan.Blocks = s.state.Plan.Blocks[:1]
	_, _ = s.Action("start", epoch, nil)
	s.Tick(epoch.Add(time.Minute), []models.RunRecord{record("last", "Smooth", 60, 80, epoch.Add(time.Minute))})
	p := s.Snapshot(nil).Plan
	if p.Recorded != 60 || p.Status != "completed" {
		t.Fatalf("deadline lost final run %+v", p)
	}
}

func TestCorruptStateNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "adaptive-training.json")
	_ = os.WriteFile(path, []byte("bad json"), 0600)
	if _, err := New(dir); err == nil {
		t.Fatal("corruption ignored")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "bad json" {
		t.Fatal("data overwritten")
	}
}

func TestExportRoundTrip(t *testing.T) {
	s := fixture(t)
	b, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if json.Unmarshal(b, &obj) != nil {
		t.Fatal("bad JSON")
	}
	ss, err := ParsePlaylist(b, Source{})
	if err != nil || len(ss) != 2 {
		t.Fatalf("export cannot be imported: %v", err)
	}
}

func TestPlaylistExecutionUsesGeneratedCountsAndTracksNextBlock(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.ExecutionMode = "playlist"
	s.state.Plan.Blocks[0].PlayCount = 2
	s.state.Plan.Blocks[1].PlayCount = 1
	b, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Scenarios []struct {
			Name  string `json:"scenarioName"`
			Count int    `json:"playCount"`
		} `json:"scenarioList"`
	}
	if json.Unmarshal(b, &file) != nil || len(file.Scenarios) != 2 || file.Scenarios[0].Count != 2 {
		t.Fatalf("wrong playlist: %s", b)
	}
	launch, err := s.Action("start", epoch, nil)
	if err != nil || launch != "" {
		t.Fatalf("playlist should be started in-game: %q %v", launch, err)
	}
	// Both records arrive in one watcher poll. The second belongs to the first
	// playlist row and the next record to the benchmark row.
	s.Tick(epoch.Add(2*time.Minute), []models.RunRecord{
		record("one", "Smooth", 60, 70, epoch.Add(time.Minute)),
		record("two", "Smooth", 60, 80, epoch.Add(2*time.Minute)),
	})
	if p := s.Snapshot(nil).Plan; p.Index != 1 || p.Blocks[0].Runs != 2 || p.Blocks[0].Outcome != "list_complete" {
		t.Fatalf("list row not completed: %+v", p)
	}
	s.Tick(epoch.Add(3*time.Minute), []models.RunRecord{record("three", "Benchmark", 60, 15, epoch.Add(3*time.Minute))})
	if p := s.Snapshot(nil).Plan; p.Status != "completed" || p.Recorded != 180 {
		t.Fatalf("list execution not attributed: %+v", p)
	}
}

func TestRejectPrivateAndNonHTTPDiscovery(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "http://example.com", "https://127.0.0.1/a", "https://[::1]/", "https://user:pass@example.com/"} {
		if publicURL(context.Background(), u) == nil {
			t.Fatalf("accepted %s", u)
		}
	}
}

func TestLateIngestionAtModuleAndSessionCap(t *testing.T) {
	s := fixture(t)
	_, _ = s.Action("start", epoch, nil)
	s.Tick(epoch.Add(180*time.Second), nil)
	if s.Snapshot(nil).Plan.Status != "waiting" {
		t.Fatal("must wait for final run")
	}
	s.Tick(epoch.Add(182*time.Second), []models.RunRecord{record("late", "Smooth", 60, 80, epoch.Add(180*time.Second))})
	if p := s.Snapshot(nil).Plan; p.Recorded != 60 || p.Index != 1 {
		t.Fatalf("late module run lost: %+v", p)
	}
	s = fixture(t)
	s.state.Plan.Preferences.Minutes = 1
	_, _ = s.Action("start", epoch, nil)
	s.Tick(epoch.Add(60*time.Second), nil)
	s.Tick(epoch.Add(63*time.Second), []models.RunRecord{record("last", "Smooth", 60, 80, epoch.Add(60*time.Second))})
	if p := s.Snapshot(nil).Plan; p.Status != "completed" || p.Recorded != 60 {
		t.Fatal("late final run lost")
	}
}

func TestSettingsChangeDisablesThreshold(t *testing.T) {
	s := fixture(t)
	old := record("old", "Smooth", 60, 100, epoch)
	s.state.Plan.Blocks[0].Signature = signature(old.Stats.Summary)
	_, _ = s.Action("start", epoch, nil)
	changed := record("changed", "Smooth", 60, 100, epoch.Add(time.Minute))
	changed.Stats.Summary.HorizSens = 2
	s.Tick(epoch.Add(time.Minute), []models.RunRecord{changed})
	if p := s.Snapshot(nil).Plan; p.Index != 0 || p.Blocks[0].Target != 0 {
		t.Fatal("applied old threshold after sensitivity change")
	}
}

func TestBenchmarkMembershipsSurviveMergeAndLegacyReload(t *testing.T) {
	first := Scenario{Name: "Shared", Skill: "smooth", Seconds: 60, Enabled: true, Benchmark: "System A", Thresholds: []float64{50, 80}}
	second := Scenario{Name: "Shared", Skill: "smooth", Seconds: 60, Enabled: true, Benchmarks: []BenchmarkMembership{{Name: "System B", Thresholds: []float64{60, 90}}}}
	merged := mergeCatalog([]Scenario{first}, []Scenario{second, second})
	if len(merged) != 1 || len(merged[0].Benchmarks) != 2 {
		t.Fatalf("membership lost or duplicated: %+v", merged)
	}
	if m, ok := member(merged[0], "System A"); !ok || m.Thresholds[1] != 80 {
		t.Fatalf("legacy membership lost: %+v", merged[0])
	}
	dir := t.TempDir()
	legacy := State{Version: 1, Catalog: []Scenario{first}, Preferences: defaults()}
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(filepath.Join(dir, "adaptive-training.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil || len(s.state.Catalog[0].Benchmarks) != 1 {
		t.Fatalf("legacy state migration failed: %v %+v", err, s)
	}
}

func TestSeveralBenchmarkSystemsChooseLeastRecentlyMeasured(t *testing.T) {
	pool := []Scenario{
		{Name: "A", Skill: "smooth", Family: "a", Seconds: 60, Enabled: true, Difficulty: "novice", Benchmarks: []BenchmarkMembership{{Name: "System A", Thresholds: []float64{50, 100}}}},
		{Name: "B", Skill: "smooth", Family: "b", Seconds: 60, Enabled: true, Difficulty: "novice", Benchmarks: []BenchmarkMembership{{Name: "System B", Thresholds: []float64{50, 100}}}},
		{Name: "Related", Skill: "smooth", Family: "related", Seconds: 60, Enabled: true, Difficulty: "novice", RelatedBenchmarks: []string{"System B"}},
	}
	p := defaults()
	p.Minutes = 15
	p.Benchmarks = []string{"System A", "System B"}
	plan, err := Generate(pool, []models.RunRecord{record("run-a", "A", 60, 70, epoch.Add(-time.Hour))}, p, epoch, rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatal(err)
	}
	last := plan.Blocks[len(plan.Blocks)-1]
	if last.Role != "benchmark" || last.Benchmark != "System B" || last.Scenario.Name != "B" {
		t.Fatalf("expected B measurement after recent A: %+v", plan.Blocks)
	}
}

func TestLateFileBeyondFifteenSeconds(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.Minutes = 1
	_, _ = s.Action("start", epoch, nil)
	s.Tick(epoch.Add(time.Minute), nil)
	s.Tick(epoch.Add(100*time.Second), []models.RunRecord{record("late", "Smooth", 60, 80, epoch.Add(time.Minute))})
	if got := s.Snapshot(nil).Plan.Recorded; got != 60 {
		t.Fatalf("delayed completed run lost: %v", got)
	}
}

func TestManualPauseAndFinishCollectFreshRuns(t *testing.T) {
	s := fixture(t)
	_, _ = s.Action("start", epoch, nil)
	_, _ = s.Action("pause", epoch.Add(time.Minute), []models.RunRecord{record("one", "Smooth", 60, 80, epoch.Add(time.Minute))})
	if p := s.Snapshot(nil).Plan; p.Recorded != 60 || p.Status != "paused" {
		t.Fatalf("pause lost a finished run: %+v", p)
	}
	_, _ = s.Action("start", epoch.Add(2*time.Minute), nil)
	_, _ = s.Action("finish", epoch.Add(3*time.Minute), []models.RunRecord{record("two", "Smooth", 60, 70, epoch.Add(3*time.Minute))})
	if p := s.Snapshot(nil).Plan; p.Recorded != 120 || p.Status != "completed" {
		t.Fatalf("finish lost a finished run: %+v", p)
	}
}

func TestManualVariantAndRelatedBenchmarkAreDistinct(t *testing.T) {
	s := fixture(t)
	s.state.Catalog = []Scenario{
		{Name: "Measured", Skill: "smooth", Family: "measured", Difficulty: "unknown", Seconds: 60, Enabled: true, Benchmarks: []BenchmarkMembership{{Name: "System A"}}},
		{Name: "Measured Easy", Skill: "smooth", Family: "measured", Difficulty: "unknown", Seconds: 60, Enabled: true},
		{Name: "Speed Match", Skill: "smooth", Family: "speed match", Difficulty: "unknown", Seconds: 60, Enabled: true},
	}
	item := s.state.Catalog[1]
	item.VariantOf = "Measured"
	if err := s.UpdateScenario(item); err != nil {
		t.Fatal(err)
	}
	other := s.state.Catalog[2]
	other.RelatedBenchmarks = []string{"System A"}
	if err := s.UpdateScenario(other); err != nil {
		t.Fatal(err)
	}
	if !variantFor(s.state.Catalog[1], s.state.Catalog, "System A") || variantFor(s.state.Catalog[2], s.state.Catalog, "System A") || !related(s.state.Catalog[2], "System A") {
		t.Fatal("direct variant and related training were conflated")
	}
	item.VariantOf = "Missing"
	if s.UpdateScenario(item) == nil {
		t.Fatal("accepted nonexistent original")
	}
}

func TestLiveDiscovery(t *testing.T) {
	if os.Getenv("REFLEKS_LIVE_DISCOVERY") != "1" {
		t.Skip("explicit network integration check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	items, d := discover(ctx, "smooth")
	if len(items) == 0 {
		t.Fatalf("no real playlist scenarios: %v", d.Warnings)
	}
	t.Logf("imported %d real scenarios from %d candidate pages; warnings: %v", len(items), len(d.Candidates), d.Warnings)
}
