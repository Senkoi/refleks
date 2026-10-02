package training

import (
	"refleks/internal/models"
	"testing"
	"time"
)

func TestNativeBenchmarkVersionsAndTiersRemainIndependent(t *testing.T) {
	definitions := []models.Benchmark{
		{BenchmarkName: "Voltaic S4", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Novice", KovaaksBenchmarkID: 41}}},
		{BenchmarkName: "Voltaic S5", Difficulties: []models.BenchmarkDifficulty{{DifficultyName: "Elite", KovaaksBenchmarkID: 51}}},
	}
	makeProgress := func(cutoffs []float64) models.BenchmarkProgress {
		return models.BenchmarkProgress{Categories: []models.ProgressCategory{{Name: "Precise Tracking", Groups: []models.ProgressGroup{{Name: "Smooth", Scenarios: []models.ScenarioProgress{{Name: "Shared", Thresholds: cutoffs}}}}}}}
	}
	imported := BenchmarkScenarios(definitions, map[int]models.BenchmarkProgress{41: makeProgress([]float64{50, 100}), 51: makeProgress([]float64{200, 400})}, epoch)
	catalog := mergeCatalog(nil, imported)
	if len(catalog) != 1 || len(catalog[0].Benchmarks) != 2 {
		t.Fatalf("lost memberships: %+v", catalog)
	}
	s := catalog[0]
	if s.Benchmarks[1].BenchmarkID != 51 || s.Benchmarks[1].NativeDifficulty != "Elite" || s.Benchmarks[1].System != "Voltaic S5" || s.Benchmarks[1].Category != "Precise Tracking" || s.ImportedCurriculum != nil {
		t.Fatalf("lost native identity: %+v", s)
	}
	runs := []models.RunRecord{}
	for i := 0; i < 3; i++ {
		runs = append(runs, record(string(rune('a'+i)), "Shared", 60, 120, epoch.Add(-time.Duration(i+1)*time.Hour)))
	}
	rows := observed(runs)["shared"]
	if e := assessDifficulty(s, rows, epoch); e.Fit != "unknown" || e.Level != "unknown" {
		t.Fatalf("arbitrary version selected: %+v", e)
	}
	for _, tc := range []struct{ name, fit string }{{"Voltaic S4 / Novice", "comfortable"}, {"Voltaic S5 / Elite", "challenging"}, {"Missing", "unknown"}} {
		e := assessDifficultyFor(s, rows, epoch, Preferences{Benchmark: tc.name})
		if e.Fit != tc.fit {
			t.Fatalf("%s: %+v", tc.name, e)
		}
	}
	s.Difficulty, s.DifficultySource, s.Classification = "advanced", "manual", "manual"
	s.PersonalDifficulty = "suitable"
	if e := assessDifficulty(s, rows, epoch); e.Level != "advanced" || e.Fit != "suitable" {
		t.Fatalf("manual correction lost: %+v", e)
	}
	// Re-sync replaces thresholds for the same ID rather than growing the list.
	imported[1].Benchmarks[0].Thresholds = []float64{220, 440}
	catalog = mergeCatalog(catalog, imported)
	if len(catalog[0].Benchmarks) != 2 || catalog[0].Benchmarks[1].Thresholds[0] != 220 {
		t.Fatal("resync duplicated identity")
	}
}
