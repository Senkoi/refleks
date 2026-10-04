package training

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"aimmeow/internal/models"
	"strings"
	"testing"
	"time"
)

func nativeFixture() ([]models.Benchmark, map[int]models.BenchmarkProgress) {
	definitions := []models.Benchmark{{BenchmarkName: "Voltaic S5", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Advanced", KovaaksBenchmarkID: 51, Ranks: []models.RankDef{{Name: "Master"}, {Name: "Grandmaster"}, {Name: "Nova"}}}}}}
	// Same shape as the live progress builder, including its UI baseline.
	progress := map[int]models.BenchmarkProgress{51: {Ranks: definitions[0].Difficulties[0].Ranks, Categories: []models.ProgressCategory{{Name: "Clicking", Groups: []models.ProgressGroup{{Name: "Static", Scenarios: []models.ScenarioProgress{{Name: "Static A", Score: 110, Thresholds: []float64{50, 100, 200, 300}}, {Name: "Static B", Score: 220, Thresholds: []float64{50, 100, 200, 300}}}}}}}}}
	return definitions, progress
}

func TestStartupUsesExistingBenchmarkScoresAndMigratesFiveRunDraft(t *testing.T) {
	s, novice := templateFixture(t)
	advanced := novice
	advanced.ID = "advanced"
	advanced.Tier = "advanced"
	advanced.Name = "VDIM Advanced Clicking I"
	// Use identical goals to isolate audience selection and actual repetition.
	novice.Rows = []CurriculumRow{{Name: "Static A", Count: 5}, {Name: "Static B", Count: 5}}
	advanced.Rows = append([]CurriculumRow(nil), novice.Rows...)
	s.state.Curricula = []Curriculum{novice, advanced}
	defs, progress := nativeFixture()
	if _, err := s.Add(BenchmarkScenarios(defs, progress, epoch)); err != nil {
		t.Fatal(err)
	}
	old := &Plan{ID: "old five-run draft", Status: "draft", Preferences: defaults(), Blocks: []Block{{Scenario: s.state.Catalog[0], PlayCount: 5, Budget: 300}}}
	s.state.Plan = old
	s.state.Preferences.Focus = "static"
	s.BeginStartup()
	if _, err := s.Generate(defaults(), nil); err == nil {
		t.Fatal("manual generation raced startup history import")
	}
	generated, err := s.PrepareStartup(nil)
	if err != nil || !generated {
		t.Fatal(err)
	}
	p := s.Snapshot(nil).Plan
	if !s.state.Initializing {
		t.Fatal("startup exposed draft before installing its playlist")
	}
	if p.ID == old.ID || p.CurriculumID != advanced.ID || p.PlayerTier != "advanced" || p.PlannerVersion != currentPlannerVersion {
		t.Fatal("benchmark achievements did not select advanced", p)
	}
	if s.state.Error != "" || s.state.Notice == "" {
		t.Fatal("startup not finalized")
	}
	for _, b := range p.Blocks {
		if b.PlayCount > 2 || b.Budget > 120 || b.SourcePlayCount != 5 {
			t.Fatal("old five-run draft visible", b)
		}
	}
	path, err := s.Install(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.FinishStartup(nil)
	if s.state.Initializing {
		t.Fatal("installed startup was not released")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Rows []CurriculumRow `json:"scenarioList"`
	}
	_ = json.Unmarshal(b, &out)
	for i, row := range out.Rows {
		if row.Count != p.Blocks[i].PlayCount {
			t.Fatal("startup installation differs from preview")
		}
	}
	levels := PlayerLevels(s.state.Catalog, nil, epoch)
	if len(levels) != 1 || levels[0].Status != "estimated" || levels[0].Samples != 0 || levels[0].WindowDays != 0 || levels[0].LastPlayed != "" {
		t.Fatal("undated PB was fabricated as recent samples", levels)
	}
}

func TestRecentHistoryOutranksUndatedBenchmarkPB(t *testing.T) {
	defs, progress := nativeFixture()
	scenes := BenchmarkScenarios(defs, progress, epoch)
	runs := append(levelRuns("Static A", []float64{30, 30, 30}, epoch), levelRuns("Static B", []float64{30, 30, 30}, epoch)...)
	levels := PlayerLevels(scenes, runs, epoch)
	if levels[0].Status != "inferred" || levels[0].Rank != "unranked" || inferredTier("static", levels) != "entry" {
		t.Fatal("PB overrode current weak scores", levels)
	}
	// A partially observed category uses the other scenario's real benchmark
	// result without inventing repeated local records.
	levels = PlayerLevels(scenes, levelRuns("Static A", []float64{110, 110, 110}, epoch), epoch)
	if levels[0].Status != "estimated" || levels[0].Source != "mixed" || levels[0].Samples != 3 || levels[0].Rank != "Master" {
		t.Fatal("mixed evidence was not distinguished", levels)
	}
}

func TestLevelWindowPrefersRecentAndFallsBackFortyFiveDaysWithoutDecay(t *testing.T) {
	scene := levelScenario("A", "static", "Voltaic S5", "Novice", []string{"Iron", "Bronze", "Silver", "Gold"}, []float64{10, 20, 30, 40})
	for _, age := range []int{2, 10, 25, 40} {
		now := epoch.AddDate(0, 0, -age)
		levels := PlayerLevels([]Scenario{scene}, levelRuns("A", []float64{45, 45, 45}, now), epoch)
		if levels[0].Rank != "Gold" || inferredTier("static", levels) != "adept" {
			t.Fatal("age applied an ability penalty", age, levels)
		}
	}
	old := levelRuns("A", []float64{45, 45, 45, 45, 45, 45, 45}, epoch.AddDate(0, 0, -40))
	recent := levelRuns("A", []float64{15, 15, 15}, epoch)
	levels := PlayerLevels([]Scenario{scene}, append(old, recent...), epoch)
	if levels[0].Rank != "Iron" || levels[0].WindowDays != 7 {
		t.Fatal("old scores overrode reliable recent median", levels)
	}
	for _, when := range []time.Time{epoch.AddDate(0, 0, -46), epoch.Add(24 * time.Hour)} {
		if PlayerLevels([]Scenario{scene}, levelRuns("A", []float64{45, 45, 45}, when), epoch)[0].Status != "insufficient" {
			t.Fatal("future or expired scores counted")
		}
	}
}

func TestUnplayedAndNewVersionRostersDoNotEraseOlderEstablishedAbility(t *testing.T) {
	older := levelScenario("old master", "static", "Voltaic S5", "Advanced", []string{"Master", "Grandmaster"}, []float64{100, 200})
	older.Benchmarks[0].Category = "Clicking"
	older.Benchmarks[0].Group = "Static"
	newer := older
	newer.Name = "new unplayed"
	newer.Benchmarks = []BenchmarkMembership{{Name: "Voltaic S5.5 / Advanced", System: "Voltaic S5.5", NativeDifficulty: "Advanced", Category: "Clicking", Group: "Static", Ranks: []string{"Master", "Grandmaster"}, Thresholds: []float64{100, 200}}}
	unseen := levelScenario("other unplayed", "static", "Voltaic S5", "Novice", []string{"Gold"}, []float64{10})
	unseen.Benchmarks[0].Group = "Unplayed subgroup"
	levels := PlayerLevels([]Scenario{older, newer, unseen}, levelRuns(older.Name, []float64{120, 120, 120}, epoch.AddDate(0, 0, -40)), epoch)
	if inferredTier("static", levels) != "advanced" {
		t.Fatal("unplayed/latest-only roster reset established ability", levels)
	}
	p := automaticReferences([]Scenario{older, newer}, defaults())
	if selectedBenchmark(p, older.Benchmarks[0].Name) || !selectedBenchmark(p, newer.Benchmarks[0].Name) {
		t.Fatal("training references stopped preferring latest version")
	}
}

func TestOrdinaryInputSettingsDoNotHideRankButScenarioRevisionsStaySeparate(t *testing.T) {
	scene := levelScenario("A", "static", "Voltaic S5", "Novice", []string{"Gold"}, []float64{40})
	runs := levelRuns("A", []float64{45, 45, 45}, epoch)
	for i := range runs {
		runs[i].Stats.Summary.HorizSens = float64(i + 1)
		runs[i].Stats.Summary.FOV = float64(90 + i*10)
	}
	if PlayerLevels([]Scenario{scene}, runs, epoch)[0].Rank != "Gold" {
		t.Fatal("ordinary sensitivity/FOV split rank history")
	}
	runs[0].Stats.Summary.Hash = "new scene revision"
	if PlayerLevels([]Scenario{scene}, runs, epoch)[0].Status != "insufficient" {
		t.Fatal("different scene versions mixed")
	}
}

func TestMalformedRankMetadataCannotSilentlyReduceCategoryCoverage(t *testing.T) {
	a := levelScenario("A", "static", "Voltaic S5", "Novice", []string{"Gold"}, []float64{40})
	b := a
	b.Name = "B"
	b.Benchmarks = []BenchmarkMembership{{Name: a.Benchmarks[0].Name, System: "Voltaic S5", NativeDifficulty: "Novice", Ranks: []string{"Gold"}, Thresholds: []float64{10, 40}}}
	if PlayerLevels([]Scenario{a, b}, levelRuns("A", []float64{45, 45, 45}, epoch), epoch)[0].Status != "partial" {
		t.Fatal("invalid scene silently disappeared from coverage requirement")
	}
}

func TestStartupPauseIsInformationalAndExistingSessionDoesNotChange(t *testing.T) {
	s, _ := templateFixture(t)
	if _, err := s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	s.state.Plan.Status = "running"
	s.state.Plan.Recorded = 60
	s.state.Error = "应用重新启动，训练已暂停；请确认后继续。"
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reopened.state.Error != "" || !strings.Contains(reopened.state.Notice, "已暂停") || reopened.state.Plan.Status != "paused" {
		t.Fatal("normal recovery shown as error")
	}
	before := *reopened.state.Plan
	reopened.BeginStartup()
	changed, err := reopened.PrepareStartup(nil)
	if err != nil || changed || !reflect.DeepEqual(before, *reopened.state.Plan) || reopened.state.Initializing {
		t.Fatal("active plan changed during startup", err)
	}
}

func TestOldImportedBaselineMigratesOnLoad(t *testing.T) {
	s, _ := templateFixture(t)
	s.state.Catalog[0].Benchmarks = []BenchmarkMembership{{BenchmarkID: 51, Name: "Voltaic S5 / Advanced", Ranks: []string{"Master", "Grandmaster"}, Thresholds: []float64{50, 100, 200}}}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reopened.state.Catalog[0].Benchmarks[0].Thresholds, []float64{100, 200}) {
		t.Fatal("old chart baseline remained in rank cutoffs")
	}
}

func TestInvalidHistoryDurationCannotBreakStartupStateJSON(t *testing.T) {
	s, _ := templateFixture(t)
	bad := record("bad", "Static A", math.NaN(), 10, time.Now())
	snapshot := s.Snapshot([]models.RunRecord{bad})
	if snapshot.Version != 1 || len(snapshot.Catalog) == 0 {
		t.Fatal("invalid history destroyed snapshot")
	}
	if _, err := json.Marshal(snapshot); err != nil {
		t.Fatal("invalid duration escaped into state JSON", err)
	}
}

func TestTopRankOnEasyRosterIsALowerBoundNotAnArtificialDemotion(t *testing.T) {
	a := levelScenario("easy A", "static", "Voltaic S5", "Novice", []string{"Iron", "Gold"}, []float64{10, 40})
	b := a
	b.Name = "easy B"
	defs, progress := nativeFixture()
	advanced := BenchmarkScenarios(defs, progress, epoch)
	for _, s := range []*Scenario{&a, &b} {
		s.Benchmarks[0].Category = "Clicking"
		s.Benchmarks[0].Group = "Static"
	}
	scenes := append([]Scenario{a, b}, advanced...)
	runs := append(levelRuns(a.Name, []float64{50, 50, 50}, epoch), levelRuns(b.Name, []float64{50, 50, 50}, epoch)...)
	levels := PlayerLevels(scenes, runs, epoch)
	if inferredTier("static", levels) != "advanced" {
		t.Fatal("easy roster top rank treated as ability ceiling", levels)
	}
	runs = append(levelRuns(a.Name, []float64{15, 15, 15}, epoch), levelRuns(b.Name, []float64{15, 15, 15}, epoch)...)
	if inferredTier("static", PlayerLevels(scenes, runs, epoch)) != "entry" {
		t.Fatal("current unsaturated scores did not outrank undated PB")
	}
}

func TestStartupSamplingSelectsLatestEachNativeRosterWithoutHidingEarlierNovice(t *testing.T) {
	defs := []models.Benchmark{
		{BenchmarkName: "Voltaic S5", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Novice", KovaaksBenchmarkID: 459}, {DifficultyName: "Intermediate", KovaaksBenchmarkID: 460}, {DifficultyName: "Advanced", KovaaksBenchmarkID: 461}}},
		{BenchmarkName: "Voltaic S5.5", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Advanced", KovaaksBenchmarkID: 999}}},
		{BenchmarkName: "Other", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Advanced", KovaaksBenchmarkID: 1234}}},
	}
	if ids := InitialLevelReferenceIDs(defs); !reflect.DeepEqual(ids, []int{459, 460, 999}) {
		t.Fatal("startup sampled old advanced or hid novice", ids)
	}
}

func TestLegacyPausedFiveRunPlanIsArchivedWithoutLosingRecordedProgress(t *testing.T) {
	s, _ := templateFixture(t)
	s.state.Plan = &Plan{ID: "legacy", Status: "paused", Recorded: 60, Preferences: defaults(), Blocks: []Block{{Scenario: s.state.Catalog[0], PlayCount: 5, Budget: 300, Runs: 1, Recorded: 60, Outcome: "pending"}}}
	s.BeginStartup()
	changed, err := s.PrepareStartup(nil)
	if err != nil || !changed || s.state.Plan.ID == "legacy" {
		t.Fatal("legacy paused list still appeared as new policy", err)
	}
	if len(s.state.History) != 1 || s.state.History[0].Recorded != 60 || s.state.History[0].Blocks[0].Runs != 1 || s.state.History[0].Blocks[0].Outcome != "skipped" {
		t.Fatal("recorded progress lost or unfinished set marked complete")
	}
	s.FinishStartup(nil)
	if s.state.Error != "" || !strings.Contains(s.state.Notice, "已归档") {
		t.Fatal("legacy migration shown as startup error")
	}
}
