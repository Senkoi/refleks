package training

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"aimmeow/internal/models"
)

func demandScene(name string, tags ...string) Scenario {
	return Scenario{Name: name, Skill: "static", Family: name, Seconds: 60, Difficulty: "unknown", Enabled: true,
		Mechanics: &Mechanics{Tags: tags, Status: "snapshot_unverified_runtime", Role: "unknown", GeometryStatus: "unknown"}}
}

func TestSnapshotDoesNotInventGeometryDifficultyOrRoles(t *testing.T) {
	if len(snapshotMechanics) != 58 { t.Fatal("incomplete intake") }
	s := enrichMechanics(Scenario{Name: "1w4t Pressure Revosect Int", Difficulty: "unknown"})
	if !pressureDemand(s) || s.Difficulty != "unknown" || s.Mechanics.AngularSize != nil || s.Mechanics.Role != "unknown" { t.Fatalf("invented calibration: %+v", s) }
	mixed := enrichMechanics(Scenario{Name: "StrawberryClick Revosect Int"})
	if !hasDemand(mixed, "wide_transfer") || !hasDemand(mixed, "micro_adjustment") || !hasDemand(mixed, "phased_targets") { t.Fatal("mixed mechanics lost") }
	unknown := enrichMechanics(Scenario{Name: "Pressure Wide Small Hard Fictional"})
	if unknown.Mechanics != nil { t.Fatal("mechanics inferred from title") }
	items, err := ParsePlaylist([]byte(`{"playlistName":"VDIM Intermediate","scenarioList":[{"scenarioName":"EvoClick Revosect Int"},{"scenarioName":"1w4t Pressure Revosect Int"}]}`), Source{})
	if err != nil || !items[0].Enabled || items[0].Skill != "static" || items[1].Seconds != 120 || items[0].Difficulty != "unknown" { t.Fatalf("snapshot intake not applied: %+v %v", items, err) }
}

func TestDemandPlanningAcrossSeedsKeepsPhasesBudgetAndMeasurement(t *testing.T) {
	pool := []Scenario{}
	for i := 0; i < 14; i++ {
		tag := "short_transfer"
		if i%2 == 0 { tag = "wide_transfer" }
		pool = append(pool, demandScene(fmt.Sprint("safe", i), tag))
	}
	for i := 0; i < 8; i++ { pool = append(pool, demandScene(fmt.Sprint("pressure", i), "time_pressure")) }
	b := demandScene("measurement", "wide_transfer")
	b.Benchmarks = []BenchmarkMembership{{Name: "System A", Thresholds: []float64{50, 100}}}
	pool = append(pool, b)
	p := defaults(); p.Focus = "static"; p.Benchmark = "System A"
	for seed := int64(0); seed < 100; seed++ {
		plan, err := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(seed)))
		if err != nil { t.Fatal(err) }
		warm, practice := 0, 0
		leftWarm := false
		seen := map[string]bool{}
		for i, block := range plan.Blocks {
			if seen[block.Scenario.Name] || block.PlayCount < 1 || block.Budget != block.PlayCount*block.Timing.Seconds { t.Fatalf("invalid block: %+v", block) }
			seen[block.Scenario.Name] = true
			if block.Role == "warmup" {
				if leftWarm || pressureDemand(block.Scenario) { t.Fatalf("pressure or late warmup: %+v", block) }
				warm += block.Budget
			} else { leftWarm = true; practice += block.Budget }
			if pressureDemand(block.Scenario) && block.PlayCount != 1 { t.Fatal("pressure repeated") }
			if i > 0 && pressureDemand(block.Scenario) && pressureDemand(plan.Blocks[i-1].Scenario) {
				for _, s := range pool { if !seen[s.Name] && !pressureDemand(s) && s.Name != "measurement" && s.Seconds <= 810-practice+block.Budget-60 { t.Fatal("stacked pressure with available alternative") } }
			}
		}
		if warm == 0 || warm > 810 || practice > 810 || warm+practice > 1620 { t.Fatalf("bad phase budget: %d %d", warm, practice) }
		last := plan.Blocks[len(plan.Blocks)-1]
		if last.Role != "benchmark" || last.PlayCount != 1 || last.Target != 0 || last.Benchmark != "System A" { t.Fatalf("measurement missing: %+v", last) }
	}
}

func TestRecentDemandCoverageChangesSelectionWithoutComparingRawScores(t *testing.T) {
	pool := []Scenario{}
	for i := 0; i < 12; i++ {
		pool = append(pool, demandScene(fmt.Sprint("wide", i), "wide_transfer"), demandScene(fmt.Sprint("short", i), "short_transfer"))
	}
	runs := []models.RunRecord{record("exposure", "wide0", 3600, 100000, epoch.Add(-time.Hour))}
	p := defaults(); p.Minutes = 5; p.Focus = "static"; p.Variety = 0
	shortWith, shortWithout := 0, 0
	for seed := int64(0); seed < 100; seed++ {
		with, err := Generate(pool, runs, p, epoch, rand.New(rand.NewSource(seed))); if err != nil { t.Fatal(err) }
		without, err := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(seed))); if err != nil { t.Fatal(err) }
		if hasDemand(with.Blocks[0].Scenario, "short_transfer") { shortWith++ }
		if hasDemand(without.Blocks[0].Scenario, "short_transfer") { shortWithout++ }
	}
	if shortWith <= shortWithout+15 { t.Fatalf("no meaningful coverage response: %d %d", shortWith, shortWithout) }
}

func TestWarmupRoleAndLegacyTierArePreservedConservatively(t *testing.T) {
	pool := []Scenario{demandScene("prep", "short_transfer"), demandScene("practice", "wide_transfer")}
	pool[0].Mechanics.Role = "warmup"
	pool[0].Benchmarks = []BenchmarkMembership{{Name: "System A"}}
	p := defaults(); p.Benchmark = "System A"
	plan, err := Generate(pool, nil, p, epoch, rand.New(rand.NewSource(1))); if err != nil { t.Fatal(err) }
	for _, b := range plan.Blocks { if b.Scenario.Name == "prep" && b.Role != "warmup" { t.Fatal("warmup exercise used as measurement") } }
	legacy := demandScene("legacy"); legacy.Difficulty = "novice"; legacy.DifficultySource = "playlist"
	merged := mergeCatalog([]Scenario{legacy}, nil)
	if merged[0].Difficulty != "unknown" { t.Fatal("playlist tier still inherited") }
}
