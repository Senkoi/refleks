package training

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"aimmeow/internal/models"
)

func TestTrainingTierKeepsGoalEvidenceWithoutChangingAchievementTier(t *testing.T) {
	levels := []PlayerLevel{
		{Theme: "static", System: "Voltaic S5", Category: "Clicking", Group: "Small", Status: "inferred", Tier: "advanced"},
		{Theme: "static", System: "Voltaic S5", Category: "Clicking", Group: "Wide", Status: "inferred", Tier: "novice"},
		{Theme: "static", System: "Voltaic S5", Category: "Clicking", Group: "Pressure", Status: "insufficient", Tier: "entry"},
		{Theme: "static", System: "Revosect", Status: "inferred", Tier: "elite"},
	}
	if inferredTier("static", levels) != "novice" || trainingTier("static", levels) != "advanced" || trainingTier("reactive", levels) != "novice" {
		t.Fatal("achievement, curriculum audience, or missing evidence conflated")
	}
	s := levelScenario("Small variant", "static", "Voltaic S5.5", "Advanced", []string{"Master"}, []float64{100})
	s.Benchmarks[0].Category, s.Benchmarks[0].Group = "Clicking", "Small"
	p := automaticReferences([]Scenario{s}, defaults())
	if _, ok := explorationReference(s, levels, p); !ok {
		t.Fatal("weak wide-transfer group lowered precision exploration")
	}
	s.Benchmarks[0].Group = "Wide"
	if _, ok := explorationReference(s, levels, p); ok {
		t.Fatal("strong precision group raised weak wide-transfer exploration")
	}
}

func TestFitUsesRecentWindowAndFortyFiveDayFallback(t *testing.T) {
	s := levelScenario("Fit scene", "static", "Voltaic S5", "Novice", []string{"Bronze", "Gold"}, []float64{100, 200})
	old := levelRuns(s.Name, []float64{250, 250, 250, 250}, epoch.AddDate(0, 0, -40))
	fit := assessDifficulty(s, levelObservations(old, epoch)[strings.ToLower(s.Name)], epoch)
	if fit.Fit != "comfortable" || fit.WindowDays != 45 || fit.Samples != 4 {
		t.Fatal("older valid fit hidden or decayed", fit)
	}
	recent := levelRuns(s.Name, []float64{125, 125, 125}, epoch)
	fit = assessDifficulty(s, levelObservations(append(old, recent...), epoch)[strings.ToLower(s.Name)], epoch)
	if fit.Fit != "suitable" || fit.WindowDays != 7 || fit.Samples != 3 {
		t.Fatal("old PB-like runs overwhelmed recent fit", fit)
	}
	outside := levelRuns(s.Name, []float64{250, 250, 250}, epoch.AddDate(0, 0, -46))
	if fit = assessDifficulty(s, levelObservations(outside, epoch)[strings.ToLower(s.Name)], epoch); fit.Fit != "unknown" {
		t.Fatal("expired scores became fit", fit)
	}
}

func TestExplorationPrefersSuitableFitAndBoundsUnknownTime(t *testing.T) {
	base := demandScene("base", "precision")
	pool := []Scenario{base}
	var runs []models.RunRecord
	for _, tc := range []struct {
		name  string
		score float64
	}{{"too hard", 20}, {"too easy", 250}, {"suitable", 150}} {
		s := demandScene(tc.name, "precision")
		s.Benchmarks = []BenchmarkMembership{{Name: "Voltaic S5 / Novice", System: "Voltaic S5", NativeDifficulty: "Novice", Thresholds: []float64{100, 200}}}
		pool = append(pool, s)
		runs = append(runs, levelRuns(s.Name, []float64{tc.score, tc.score, tc.score}, epoch)...)
	}
	pool = append(pool, demandScene("unknown", "precision"))
	p := defaults()
	p.Minutes, p.Variety = 15, .5
	for seed := int64(0); seed < 20; seed++ {
		plan, err := GenerateCurriculum(Curriculum{ID: "fit", Rows: []CurriculumRow{{Name: "base", Count: 1}}}, pool, runs, p, epoch, rand.New(rand.NewSource(seed)), true)
		if err != nil || len(plan.Blocks) != 2 || plan.Blocks[1].Scenario.Name != "suitable" || plan.Blocks[1].DifficultyEvidence.Fit != "suitable" {
			t.Fatal("fit did not control actual generated scenario", err, plan)
		}
	}
	// Several compatible anchors share the 10% uncertainty budget.
	p.Minutes = 30
	template := Curriculum{ID: "unknown"}
	pool = nil
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("base %d", i)
		pool = append(pool, demandScene(name, "precision"), demandScene(fmt.Sprintf("unknown %d", i), "precision"))
		template.Rows = append(template.Rows, CurriculumRow{Name: name, Count: 1})
	}
	plan, err := GenerateCurriculum(template, pool, nil, p, epoch, rand.New(rand.NewSource(0)), true)
	if err != nil {
		t.Fatal(err)
	}
	unknownSeconds := 0
	for _, b := range plan.Blocks {
		if b.Role == "explore" {
			unknownSeconds += b.Budget
		}
	}
	if unknownSeconds == 0 || unknownSeconds > 162 {
		t.Fatal("unknown exploration exceeded 10% of the usable session", unknownSeconds)
	}
}

func TestTemplateFallbackChoosesClosestTierAndExplainsIt(t *testing.T) {
	s, novice := templateFixture(t)
	defs, progress := nativeFixture()
	_, _ = s.Add(BenchmarkScenarios(defs, progress, epoch))
	higher := novice
	higher.ID, higher.Name, higher.Tier = "elite", "Elite S5 - Clicking I", "elite"
	p := defaults()
	p.Focus = "static"
	chosen, err := selectCurriculum([]Curriculum{novice, higher}, s.state.Catalog, p, nil, epoch)
	if err != nil || chosen.ID != higher.ID {
		t.Fatal("distant novice selected over neighbouring elite", err, chosen)
	}
	plan, err := GenerateCurriculum(*chosen, s.state.Catalog, nil, p, epoch, rand.New(rand.NewSource(0)), false)
	if err != nil || plan.PlayerTier != "advanced" || plan.TierReason == "" || len(plan.Warnings) < 4 {
		t.Fatal("fallback has no evidence or warning", err, plan)
	}
}
