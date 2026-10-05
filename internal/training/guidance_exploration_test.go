package training

import (
	"bytes"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"aimmeow/internal/models"
)

func explorationFixture(now time.Time) (Curriculum, []Scenario, []models.RunRecord) {
	a := demandScene("foundation", "precision")
	a.Classification = "manual"
	b := demandScene("unplayed row", "precision")
	b.Classification = "manual"
	novel := Scenario{Name: "author classified new scene", Skill: a.Skill, Technique: scenarioTheme(a), Enabled: true, Classification: "manual", Seconds: 60, Sources: []Source{{Title: "Other author routine"}}}
	guess := novel
	guess.Name = "name-only guess"
	guess.Classification = "inferred"
	wrong := novel
	wrong.Name = "other theme"
	wrong.Technique = "reactive"
	t := Curriculum{ID: "foundation-template", Theme: scenarioTheme(a), Tier: "novice", Rows: []CurriculumRow{{Name: a.Name, Count: 2}, {Name: b.Name, Count: 2}}}
	runs := []models.RunRecord{}
	for i := 0; i < 3; i++ {
		runs = append(runs, record(fmt.Sprintf("familiar-%d", i), a.Name, 60, 100, now.Add(-time.Duration(i+1)*time.Minute)))
	}
	return t, []Scenario{a, b, novel, guess, wrong}, runs
}

func TestFamiliarTaskUnlocksTrustedUnknownSceneBeforeWholeTemplate(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	template, catalog, runs := explorationFixture(now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Catalog = catalog
	s.state.Curricula = []Curriculum{template}
	plan, err := s.Generate(defaults(), runs)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Exploration == nil || plan.Exploration.Status != "selected" || plan.Exploration.UsedSeconds != 60 {
		t.Fatal(plan.Exploration)
	}
	if plan.Blocks[len(plan.Blocks)-1].Scenario.Name != catalog[2].Name || plan.Blocks[len(plan.Blocks)-1].Role != "explore" {
		t.Fatal(plan.Blocks)
	}
	if plan.Blocks[0].Scenario.Name != template.Rows[0].Name || plan.Blocks[1].Scenario.Name != template.Rows[1].Name {
		t.Fatal("main sequence changed")
	}
	for _, b := range plan.Blocks {
		if b.Scenario.Name == catalog[3].Name || b.Scenario.Name == catalog[4].Name {
			t.Fatal("untrusted or unrelated scene entered", b)
		}
	}
	if s.state.CurriculumProgress[progressKey(template)].EverCompleted {
		t.Fatal("generation invented a baseline")
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reopened.state.Plan.Exploration, plan.Exploration) || !reflect.DeepEqual(definePlan(reopened.state.Plan), definePlan(s.state.Plan)) {
		t.Fatal("exploration decision did not survive reload")
	}
}

func TestIncompleteCheatedAndMixedSettingsDoNotUnlockProbes(t *testing.T) {
	for _, invalid := range []string{"partial", "cheated", "mixed", "too_few"} {
		t.Run(invalid, func(t *testing.T) {
			template, catalog, runs := explorationFixture(epoch)
			switch invalid {
			case "partial":
				for i := range runs {
					runs[i].Stats.Summary.TimeRemaining = 20
				}
			case "cheated":
				for i := range runs {
					runs[i].Stats.Events = []models.RunStatsEvent{{Cheated: true}}
				}
			case "mixed":
				for i := range runs {
					runs[i].Stats.Summary.FOV += float64(i)
				}
			case "too_few":
				runs = runs[:2]
			}
			extra, report, _ := selectTrainingExtras(template, catalog, runs, nil, nil, defaults(), epoch, false, 60, rand.New(rand.NewSource(1)), nil)
			if extra.seconds() != 0 || report.Status == "selected" {
				t.Fatal("invalid familiarity unlocked exploration", report)
			}
			found := false
			for _, r := range report.Reasons {
				found = found || r.Code == "practice_evidence"
			}
			if !found {
				t.Fatal("missing evidence explanation", report)
			}
		})
	}
}

func TestControlledAndOrdinaryTrialsShareOneBudget(t *testing.T) {
	a := personalScene(t, "anchor", 10, "", epoch)
	b := personalScene(t, "controlled", 9, "", epoch)
	ordinary := demandScene("other author variant", a.Mechanics.Tags...)
	ordinary.Classification = "manual"
	ordinary.VariantOf = a.Name
	template := Curriculum{ID: "t", Theme: "static", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	runs := anchorRuns(a.Name, epoch)
	for _, minutes := range []int{30, 40} {
		p := defaults()
		p.Minutes = minutes
		bundle, report, limits := selectTrainingExtras(template, []Scenario{a, b, ordinary}, runs, nil, nil, p, epoch, false, 60, rand.New(rand.NewSource(1)), nil)
		if report.UsedSeconds > limits.ExplorationLimit || report.UnknownSeconds > limits.UnknownLimit || bundle.seconds() > p.Minutes*60*9/10-60 {
			t.Fatal("shared quota exceeded", report, limits)
		}
		if minutes == 40 && (bundle.study == nil || len(bundle.after) != 2 || report.UsedSeconds != 180) {
			t.Fatal("controlled trial suppressed ordinary exploration", bundle, report)
		}
	}
}

func TestGeneratedPlanReservesBothTrialsBeforeFillingMainWindow(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	a := personalScene(t, "anchor", 10, "", now)
	b := personalScene(t, "controlled", 9, "", now)
	ordinary := demandScene("other author variant", a.Mechanics.Tags...)
	ordinary.Classification, ordinary.VariantOf = "manual", a.Name
	template := Curriculum{ID: "t", Theme: "static", Rows: []CurriculumRow{{Name: a.Name, Count: 100}}}
	runs := anchorRuns(a.Name, now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Catalog = []Scenario{a, b, ordinary}
	s.state.Curricula = []Curriculum{template}
	p := defaults()
	p.Minutes = 40
	plan, err := s.Generate(p, runs)
	if err != nil {
		t.Fatal(err)
	}
	seconds, trials := 0, 0
	for _, block := range plan.Blocks {
		seconds += block.Budget
		if block.Role == "explore" {
			trials++
		}
	}
	if trials != 2 || len(s.state.TrainingStudies) != 1 || plan.Exploration.UsedSeconds != 180 || seconds > p.Minutes*60*9/10 {
		t.Fatal("main window starved or exceeded the shared trial budget", trials, seconds, plan.Exploration, s.state.TrainingStudies)
	}
	if s.state.TrainingStudies[0].PlanID != plan.ID || s.state.TrainingStudies[0].TrainingScenario != b.Name || plan.Blocks[0].Scenario.Name != a.Name {
		t.Fatal("trial attribution or main order changed")
	}
	p.ExplorationMode = "off" // Explicit off also works before preference normalization.
	_, report, _ := selectTrainingExtras(template, s.state.Catalog, runs, nil, nil, p, now, false, 60, rand.New(rand.NewSource(1)), nil)
	if report.Status != "disabled" || report.UsedSeconds != 0 {
		t.Fatal(report)
	}
	p = defaults()
	p.Minutes = 5
	_, report, _ = selectTrainingExtras(template, []Scenario{a, b}, runs, nil, nil, p, now, false, 60, rand.New(rand.NewSource(1)), nil)
	if report.Status != "budget_limited" || len(report.Reasons) == 0 || report.Reasons[0].Code != "whole_run_budget" {
		t.Fatal("controlled trial lacked a whole-run exclusion explanation", report)
	}
}

func TestWholeRunBudgetAndExplicitOffRemainVisible(t *testing.T) {
	template, catalog, runs := explorationFixture(epoch)
	p := defaults()
	p.Minutes = 5
	_, report, _ := selectTrainingExtras(template, catalog, runs, nil, nil, p, epoch, false, 60, rand.New(rand.NewSource(1)), nil)
	if report.Status != "budget_limited" || len(report.Reasons) == 0 {
		t.Fatal(report)
	}
	p = automaticTrainingPreferences(Preferences{Variety: 0})
	if p.Variety != 0 || p.ExplorationMode != "off" {
		t.Fatal("legacy zero preference was silently enabled", p)
	}
	p = defaults()
	p.ExplorationMode = "off"
	p = automaticTrainingPreferences(p)
	_, report, _ = selectTrainingExtras(template, catalog, runs, nil, nil, p, epoch, false, 60, rand.New(rand.NewSource(1)), nil)
	if report.Status != "disabled" || report.UsedSeconds != 0 {
		t.Fatal(report)
	}
	p.ExplorationMode = "auto"
	p = automaticTrainingPreferences(p)
	if p.Variety != .1 {
		t.Fatal("explicitly enabled auto mode remained zero", p)
	}
}

func TestGuidanceSharesPlanAndDoesNotGenerateOrRewriteIt(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	template, catalog, runs := explorationFixture(now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Catalog = catalog
	s.state.Curricula = []Curriculum{template}
	guide := s.Guidance(runs)
	if guide.Mode != "next" || guide.Theme != template.Theme || s.state.Plan != nil || len(s.state.TrainingStudies) != 0 || len(s.state.AnchorEvaluations) != 0 {
		t.Fatal("advice persisted a plan or reserved a study", guide)
	}
	plan, err := s.Generate(defaults(), runs)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.Export()
	definition := definePlan(s.state.Plan)
	actual := s.Guidance(runs)
	progress := s.Progress()
	if actual.Mode != "current" || actual.PlanID != plan.ID || !reflect.DeepEqual(actual, *progress.Guidance) || actual.Items[0].Scenario != plan.Blocks[0].Scenario.Name {
		t.Fatal("overview and workbench differ", actual, progress)
	}
	actual.Items[0].Scenario = "external edit"
	actual.Exploration.Selected[0] = "external edit"
	again := s.Guidance(runs)
	if again.Items[0].Scenario == "external edit" || again.Exploration.Selected[0] == "external edit" {
		t.Fatal("advice exposed mutable state")
	}
	after, _ := s.Export()
	if !bytes.Equal(before, after) || !reflect.DeepEqual(definition, definePlan(s.state.Plan)) {
		t.Fatal("reading advice rewrote the fixed list")
	}
	if _, err = s.Action("finish", now, runs); err != nil {
		t.Fatal(err)
	}
	if next := s.Guidance(runs); next.Mode != "next" || next.Theme != guide.Theme {
		t.Fatal(next)
	}
}
