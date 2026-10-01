package training

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"refleks/internal/models"
)

func TestDifficultyEvidencePreservesUnknownAndManualOverride(t *testing.T) {
	playlist := []byte(`{"playlistName":"VDIM Novice Static","scenarioList":[{"scenarioName":"1w2ts"},{"scenarioName":"Static Advanced"}]}`)
	items, err := ParsePlaylist(playlist, Source{URL: "https://example.com/a.json"})
	if err != nil { t.Fatal(err) }
	if items[0].Difficulty != "novice" || items[0].DifficultySource != "playlist" || items[1].DifficultySource != "name" {
		t.Fatalf("lost source provenance: %+v", items)
	}
	unknown, _ := ParsePlaylist([]byte(`{"scenarioList":[{"scenarioName":"Smoothbot"}]}`), Source{})
	if unknown[0].Difficulty != "unknown" || unknown[0].DifficultySource != "unknown" { t.Fatalf("invented difficulty: %+v", unknown[0]) }
	items[0].Difficulty = "intermediate"
	items[0].DifficultySource = "manual"
	items[0].Classification = "manual"
	refresh, _ := ParsePlaylist(playlist, Source{URL: "https://example.com/a.json"})
	merged := mergeCatalog(items, refresh)
	if merged[0].Difficulty != "intermediate" || merged[0].DifficultySource != "manual" { t.Fatal("refresh overwrote correction") }
}

func TestPersonalFitUsesSameScenarioComparableScores(t *testing.T) {
	s := Scenario{Name: "Benchmark", Difficulty: "novice", DifficultySource: "benchmark", Benchmarks: []BenchmarkMembership{{Name: "VT / Novice", Thresholds: []float64{100, 200}}}}
	runs := []models.RunRecord{}
	for i := 0; i < 4; i++ { runs = append(runs, record(string(rune('a'+i)), s.Name, 60, 70, epoch.Add(-time.Duration(i+1)*time.Hour))) }
	e := assessDifficulty(s, observed(runs)["benchmark"], epoch)
	if e.Fit != "challenging" || e.Samples != 4 || e.Level != "novice" { t.Fatalf("unexpected fit: %+v", e) }
	other := assessDifficulty(Scenario{Name: "Unknown"}, observed(runs)["benchmark"], epoch)
	if other.Fit != "unknown" { t.Fatal("nonbenchmark score labeled an unrelated scenario") }
}

func TestVDIMRotatesUndertrainedThemeAndWeightsPlan(t *testing.T) {
	pool := []Scenario{}
	for _, theme := range vdimThemes {
		for i := 0; i < 5; i++ {
			skill := theme
			if strings.HasPrefix(theme, "switching_") { skill = "switching" }
			name := theme + string(rune('A'+i))
			pool = append(pool, Scenario{Name: name, Skill: skill, Technique: theme, Family: name, Seconds: 60, Enabled: true, Difficulty: "novice"})
		}
	}
	// Monday would start with static, but static had enough recent practice.
	monday := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	runs := []models.RunRecord{record("static-1", "staticA", 600, 100, monday.Add(-time.Hour))}
	plan, err := Generate(pool, runs, defaults(), monday, rand.New(rand.NewSource(4)))
	if err != nil { t.Fatal(err) }
	if plan.Theme != "dynamic" { t.Fatalf("undertrained theme not selected: %s", plan.Theme) }
	focused := 0
	for _, b := range plan.Blocks { if scenarioTheme(b.Scenario) == plan.Theme { focused++ } }
	if focused < 2 { t.Fatalf("no specialist block emphasis: %+v", plan.Blocks) }
}
