package training

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestWorkbenchSummaryAndOnDemandVersionedInspection(t *testing.T) {
	s, _ := templateFixture(t)
	_, m, a, err := ParseLocalSCE([]byte(localSCEFixture))
	if err != nil {
		t.Fatal(err)
	}
	a.FilePath = "fixture.sce"
	s.state.Catalog[0].LocalAssessment = a
	s.state.Catalog[0].Mechanics = m
	s.state.Plan = &Plan{ID: "fixed", Blocks: []Block{{Scenario: s.state.Catalog[0]}}}
	view, err := s.Workbench(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Catalog[0].LocalAssessment.Requirements.Axes) != 0 || len(view.Catalog[0].LocalAssessment.Fields) != 0 {
		t.Fatal("full definitions leaked into workbench")
	}
	detailed, err := s.SceneRequirements(s.state.Catalog[0].Name, a.FileSHA256)
	if err != nil || len(detailed.Axes) != 6 || len(detailed.Definitions) != 0 {
		t.Fatal("inspection", err)
	}
	detailed.Targets[0].Facts[0].Text = "mutated"
	if reflect.DeepEqual(detailed.Targets[0].Facts, a.Requirements.Targets[0].Facts) {
		t.Fatal("inspection shares backend memory")
	}
	s.state.Catalog[0].LocalAssessment = &LocalAssessment{Status: "invalid_local_file"}
	old, err := s.SceneRequirements(s.state.Plan.Blocks[0].Scenario.Name, a.FileSHA256)
	if err != nil || old.FileSHA256 != a.FileSHA256 {
		t.Fatal("fixed version no longer inspectable", err)
	}
	if _, err = s.SceneRequirements(s.state.Plan.Blocks[0].Scenario.Name, "unknown version"); err == nil {
		t.Fatal("incorrect content binding accepted")
	}
}
func TestGenerationWriteFailureRollsBackPlanAndReservations(t *testing.T) {
	s, _ := templateFixture(t)
	old, err := s.Generate(defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(s.state)
	revision := s.dataRevision
	if err = os.Mkdir(s.path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Generate(defaults(), nil); err == nil {
		t.Fatal("failed write not reported")
	}
	after, _ := json.Marshal(s.state)
	if string(before) != string(after) || s.state.Plan.ID != old.ID || s.dataRevision != revision {
		t.Fatal("failed generation left a new draft or consumed assessment")
	}
}
func TestAssessmentDoesNotAssignCurrentFileToNameOnlyHistory(t *testing.T) {
	now := epoch.Add(10 * 24 * time.Hour)
	a := personalScene(t, "anchor", 10, "", epoch)
	runs := anchorRuns(a.Name, now)
	in := AssessmentInput{Catalog: []Scenario{a}, Runs: runs, Now: now, Preferences: defaults()}
	before, _ := json.Marshal(in.Catalog)
	result := AssessPlayer(in)
	if len(result.Anchors) != 1 || result.Anchors[0].Status != "stable" || len(result.Coverage) != 0 {
		t.Fatal("name-only history was relabeled with current SCE", result.Anchors, result.Coverage)
	}
	after, _ := json.Marshal(in.Catalog)
	if string(before) != string(after) {
		t.Fatal("assessment mutated catalog input")
	}
}
func TestTaskContextUsesConfirmedTaxonomyWithoutMutatingSCE(t *testing.T) {
	a := personalScene(t, "a", 10, "", epoch)
	b := personalScene(t, "b", 10, "", epoch)
	// Legacy fixture has no AimTypeTag; only manual/native classifications may
	// provide the contextual task, not a name-derived static classification.
	a.Classification = "inferred"
	if contextualRequirements(a).Task != "unknown" {
		t.Fatal("name guess completed task")
	}
	a.Classification = "manual"
	if contextualRequirements(a).Task != "clicking" || a.LocalAssessment.Requirements.Task != "unknown" {
		t.Fatal("task resolution mutated content descriptor")
	}
	b.Skill = "reactive"
	if CompareScenarios(a, b).Result.Kind != "incompatible" {
		t.Fatal("different training goal ordered")
	}
}

func TestNonRadiusNeighborIsOnlyABoundedExplorationPreference(t *testing.T) {
	makeScene := func(name string, speed int) Scenario {
		data := fmt.Sprintf(`Name=%s
AimTypeTag=Clicking
Timelimit=60
PlayerProfile=player
AddedBots=target.bot;target.bot
IsTimeDilationActive=false
IsTargetSizeActive=false
[Character Profile]
Name=player
[Bot Profile]
Name=target
DisableScoring=false
NoDodging=false
DodgeProfileNames=move.dodge
CharacterProfile=target
[Character Profile]
Name=target
MainBBType=Sphere
MainBBRadius=10
MainBBHeight=10
MaxSpeed=%d
Acceleration=500
Gravity=0
[Dodge Profile]
Name=move
ToggleLeftRight=true
MinLRTimeChange=0.2
MaxLRTimeChange=0.3
[Map Data]
{"objects":[]}
`, name, speed)
		n, m, a, e := ParseLocalSCE([]byte(data))
		if e != nil {
			t.Fatal(e)
		}
		a.FilePath = "fixture.sce"
		return Scenario{Name: n, Skill: "static", Family: "fixture", Seconds: 60, Enabled: true, Classification: "manual", Mechanics: m, LocalAssessment: a}
	}
	a, b := makeScene("anchor", 100), makeScene("neighbor", 150)
	b.VariantOf = a.Name
	runs := anchorRuns(a.Name, epoch)
	options := progressionCandidates(Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}, []Scenario{a, b}, runs, defaults(), epoch, nil)
	if len(options) != 1 || options[0].neighbor == nil || options[0].kind != "explore" || options[0].neighbor.Result.PlannerUse != "neighbor_tiebreak_only" {
		t.Fatal("neighbor missing or created a challenge", options)
	}
	if options[0].neighbor.Result.RankMargin != nil {
		t.Fatal("uncalibrated margin entered planning")
	}
}
