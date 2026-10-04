package training

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"refleks/internal/models"
)

func sessionRuns(name string, scores []float64) []models.RunRecord {
	runs := []models.RunRecord{}
	for i, score := range scores {
		runs = append(runs, record(fmt.Sprintf("session-%s-%d", name, i), name, 60, score, epoch.Add(-time.Duration(len(scores)-i)*time.Hour)))
	}
	return runs
}

func TestProgressionTrendIgnoresSingleBadRunAndRetryCluster(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scores []float64
		want   string
	}{
		{"improving", []float64{50, 50, 50, 65, 65, 65}, "improving"},
		{"single low", []float64{100, 100, 100, 100, 100, 20}, "stable"},
		{"repeated decline", []float64{100, 100, 100, 70, 70, 70}, "declining"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := levelObservations(sessionRuns("A", tc.scores), epoch)["a"]
			trend, n := performanceTrend(rows, epoch)
			if trend != tc.want || n != 6 {
				t.Fatal(trend, n)
			}
		})
	}
	runs := sessionRuns("A", []float64{100, 100, 100, 20, 20, 20})
	for i := 3; i < 6; i++ {
		runs[i].Stats.Summary.DatePlayed = epoch.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
	}
	if trend, n := performanceTrend(levelObservations(runs, epoch)["a"], epoch); trend == "declining" || n >= 6 {
		t.Fatal("retry cluster became sustained decline", trend, n)
	}
}

func TestAccuracyRegressionStopsProgressionDespiteHigherScore(t *testing.T) {
	runs := sessionRuns("A", []float64{100, 100, 100, 110, 110, 110})
	for i := range runs {
		runs[i].Stats.Summary.HitCount = 80
		runs[i].Stats.Summary.MissCount = 20
		if i >= 3 {
			runs[i].Stats.Summary.HitCount = 50
			runs[i].Stats.Summary.MissCount = 50
		}
	}
	if trend, _ := performanceTrend(levelObservations(runs, epoch)["a"], epoch); trend != "declining" {
		t.Fatal("accuracy loss ignored", trend)
	}
}

func progressionFixture() (Scenario, Scenario, []models.RunRecord) {
	base := levelScenario("foundation", "static", "Voltaic S5", "Novice", []string{"Bronze", "Silver", "Gold"}, []float64{20, 30, 40})
	base.Mechanics = demandScene("", "precision").Mechanics
	base.Benchmarks[0].Category, base.Benchmarks[0].Group = "Clicking", "Small"
	next := levelScenario("next roster", "static", "Voltaic S5", "Intermediate", []string{"Platinum", "Diamond"}, []float64{100, 200})
	next.Mechanics = base.Mechanics
	next.Benchmarks[0].Category, next.Benchmarks[0].Group = "Clicking", "Small"
	runs := sessionRuns(base.Name, []float64{36, 36, 36})
	for i := range runs {
		runs[i].Stats.Summary.DatePlayed = epoch.Add(-time.Duration(3-i) * 24 * time.Hour).Format(time.RFC3339)
	}
	return base, next, runs
}

func TestNearThresholdTrialsDoNotGrantRankOrCrossGoals(t *testing.T) {
	base, next, runs := progressionFixture()
	far := next
	far.Name = "two tiers away"
	far.Benchmarks = append([]BenchmarkMembership(nil), next.Benchmarks...)
	far.Benchmarks[0].NativeDifficulty = "Advanced"
	far.Benchmarks[0].Name = "Voltaic S5 / Advanced"
	wrong := next
	wrong.Name = "wrong group"
	wrong.Benchmarks = append([]BenchmarkMembership(nil), next.Benchmarks...)
	wrong.Benchmarks[0].Group = "Wide"
	catalog := []Scenario{base, next, far, wrong}
	p := defaults()
	p.Minutes = 15
	p.Variety = 0
	plan, err := GenerateCurriculum(Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: base.Name, Count: 1}}}, catalog, runs, p, epoch, rand.New(rand.NewSource(0)), true)
	if err != nil || len(plan.Blocks) != 2 || plan.Blocks[1].Role != "challenge" || plan.Blocks[1].Scenario.Name != next.Name {
		t.Fatal("adjacent trial missing or unrelated goal raised", err, plan)
	}
	if plan.PlayerTier != "novice" {
		t.Fatal("trial granted a formal promotion", plan.PlayerTier)
	}
	for _, l := range PlayerLevels(catalog, runs, epoch) {
		if l.Status == "inferred" && l.Rank != "Silver" {
			t.Fatal("trial changed official evidence", l)
		}
	}
	// A PB without sessions cannot prove readiness.
	base.Benchmarks[0].BenchmarkScore = new(float64)
	*base.Benchmarks[0].BenchmarkScore = 36
	plan, err = GenerateCurriculum(Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: base.Name, Count: 1}}}, []Scenario{base, next}, nil, p, epoch, rand.New(rand.NewSource(0)), true)
	if err != nil || len(plan.Blocks) != 1 {
		t.Fatal("undated PB triggered trial", err, plan)
	}
}

func TestImprovingHardSceneIsTrainableAndPersistentDeclineRollsBack(t *testing.T) {
	base := demandScene("base", "precision")
	c := levelScenario("challenge", "static", "Voltaic S5", "Novice", []string{"Bronze", "Gold"}, []float64{100, 200})
	c.Mechanics = base.Mechanics
	p := defaults()
	p.Minutes = 15
	for _, tc := range []struct {
		scores []float64
		want   bool
	}{{[]float64{50, 50, 50, 65, 65, 65}, true}, {[]float64{100, 100, 100, 60, 60, 60}, false}, {[]float64{10, 10, 10, 20, 20, 20}, false}} {
		runs := sessionRuns(c.Name, tc.scores)
		for i := range runs {
			runs[i].Stats.Summary.DatePlayed = epoch.Add(-time.Duration(len(runs)-i) * 24 * time.Hour).Format(time.RFC3339)
		}
		plan, err := GenerateCurriculum(Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: base.Name, Count: 1}}}, []Scenario{base, c}, runs, p, epoch, rand.New(rand.NewSource(0)), true)
		if err != nil {
			t.Fatal(err)
		}
		if (len(plan.Blocks) > 1) != tc.want {
			t.Fatal("hard-scene policy", tc, plan)
		}
		if tc.want && (plan.Blocks[1].Role != "challenge" || plan.Blocks[1].DifficultyEvidence.Fit != "challenging") {
			t.Fatal("trainable challenge hidden", plan.Blocks)
		}
	}
}

func TestAdjacentOfficialTemplateTrialsPreserveFoundationAndBudget(t *testing.T) {
	base, _, runs := progressionFixture()
	pool := []Scenario{base}
	templates := []Curriculum{}
	for i := 0; i < 12; i++ {
		c := demandScene(fmt.Sprintf("higher-%d", i), "precision")
		c.PersonalDifficulty = "suitable"
		pool = append(pool, c)
		templates = append(templates, Curriculum{Theme: "static", Tier: "adept", OfficialCode: "verified", Rows: []CurriculumRow{{Name: c.Name, Count: 1}}})
	}
	far := demandScene("far template", "precision")
	pool = append(pool, far)
	templates = append(templates, Curriculum{Theme: "static", Tier: "elite", OfficialCode: "verified", Rows: []CurriculumRow{{Name: far.Name, Count: 1}}})
	p := defaults()
	p.Minutes = 30
	template := Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: base.Name, Count: 1}}}
	plan, err := GenerateCurriculum(template, pool, runs, p, epoch, rand.New(rand.NewSource(0)), true, templates...)
	if err != nil {
		t.Fatal(err)
	}
	challenge, unknown, total := 0, 0, 0
	for _, b := range plan.Blocks {
		total += b.Budget
		if b.Role == "challenge" {
			challenge += b.Budget
		}
		if b.Role != "practice" && b.DifficultyEvidence.Fit == "unknown" {
			unknown += b.Budget
		}
		if b.Scenario.Name == far.Name {
			t.Fatal("far template bypassed challenge limits")
		}
	}
	if challenge != 480 || unknown > 162 || total > 1620 || plan.Blocks[0].Scenario.Name != base.Name || plan.Blocks[0].PlayCount != 1 {
		t.Fatal("budget or original sequence changed", challenge, unknown, total)
	}
	first, err := GenerateCurriculum(template, pool, runs, p, epoch, rand.New(rand.NewSource(0)), false, templates...)
	if err != nil || len(first.Blocks) != 1 {
		t.Fatal("first foundation session changed", err)
	}
	// Strict native alignment must not re-admit a stale higher PB after local decline.
	levels := []PlayerLevel{{Theme: "static", System: "Voltaic S5", NativeDifficulty: "Novice", Category: "Clicking", Group: "Small", Tier: "novice", Status: "inferred"}, {Theme: "static", System: "Voltaic S5", NativeDifficulty: "Advanced", Category: "Clicking", Group: "Small", Tier: "advanced", Status: "estimated"}}
	farB := levelScenario("old PB", "static", "Voltaic S5", "Advanced", []string{"Master"}, []float64{100})
	farB.Benchmarks[0].Category, farB.Benchmarks[0].Group = "Clicking", "Small"
	if _, ok := explorationReference(farB, levels, automaticReferences([]Scenario{farB}, p)); ok {
		t.Fatal("higher PB bypassed current goal tier")
	}
}
