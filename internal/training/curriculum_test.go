package training

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func templateFixture(t *testing.T) (*Service, Curriculum) {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := []byte(`{"playlistName":"VDIM Novice Clicking I","scenarioList":[{"scenarioName":"Static A","playCount":2,"role":"warmup"},{"scenarioName":"Static B","playCount":3},{"scenarioName":"Static A","playCount":1},{"scenarioName":"Static Measure","playCount":1,"role":"benchmark"}]}`)
	if _, err = s.Import(b, Source{URL: "fixture:vdim"}); err != nil {
		t.Fatal(err)
	}
	if len(s.state.Curricula) != 1 || len(s.state.Catalog) != 3 {
		t.Fatal("template dropped or catalog duplicates")
	}
	return s, s.state.Curricula[0]
}

func TestCurriculumFirstGenerationPreservesEveryRowAndCount(t *testing.T) {
	s, template := templateFixture(t)
	p := defaults()
	p.Variety = .5
	for seed := int64(0); seed < 100; seed++ {
		plan, err := GenerateCurriculum(template, s.state.Catalog, nil, p, epoch, rand.New(rand.NewSource(seed)), false)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Blocks) != len(template.Rows) {
			t.Fatal("first template altered")
		}
		for i, b := range plan.Blocks {
			if b.Scenario.Name != template.Rows[i].Name || b.PlayCount != template.Rows[i].Count || b.Target != 0 {
				t.Fatal("original sequence/count changed")
			}
		}
		if plan.Blocks[0].Role != "warmup" || plan.Blocks[3].Role != "benchmark" {
			t.Fatal("explicit roles lost")
		}
	}
	plan, err := s.Generate(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Rows []CurriculumRow `json:"scenarioList"`
	}
	_ = json.Unmarshal(data, &exported)
	for i, r := range exported.Rows {
		if r.Name != template.Rows[i].Name || r.Count != plan.Blocks[i].PlayCount || plan.Blocks[i].SourcePlayCount != template.Rows[i].Count || r.Count > template.Rows[i].Count {
			t.Fatal("export diverged from the actual adaptive plan")
		}
	}
	if plan.CurriculumID != template.ID {
		t.Fatal("default service bypasses template")
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil || !reflect.DeepEqual(s.state.Curricula, reopened.state.Curricula) {
		t.Fatal("template does not survive restart", err)
	}
}

func TestCurriculumExplorationKeepsGoalAllowsExtraDimensions(t *testing.T) {
	a := demandScene("base", "short_transfer", "precision")
	a.Classification = "manual"
	good := demandScene("novel subtype", "short_transfer", "precision", "phased_targets")
	wrong := demandScene("wrong goal", "wide_transfer", "precision")
	other := demandScene("other skill", "short_transfer", "precision")
	other.Skill = "reactive"
	unknown := demandScene("unknown mechanics")
	template := Curriculum{ID: "t", Name: "VDIM", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	p := defaults()
	p.Minutes = 15
	p.Variety = .5
	for seed := int64(0); seed < 100; seed++ {
		plan, err := GenerateCurriculum(template, []Scenario{a, wrong, other, unknown, good}, nil, p, epoch, rand.New(rand.NewSource(seed)), true)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Blocks) != 2 || plan.Blocks[0].Scenario.Name != a.Name || plan.Blocks[0].PlayCount != 2 || plan.Blocks[1].Scenario.Name != good.Name || plan.Blocks[1].AnchorScenario != a.Name || plan.Blocks[1].PlayCount != 1 {
			t.Fatalf("lost goal: %+v", plan.Blocks)
		}
		if !hasDemand(plan.Blocks[1].Scenario, "phased_targets") {
			t.Fatal("new dimension excluded")
		}
	}
	p.Minutes = 5
	template.Rows[0].Count = 5
	if _, err := GenerateCurriculum(template, []Scenario{a, good}, nil, p, epoch, rand.New(rand.NewSource(1)), true); err == nil {
		t.Fatal("over-budget template silently truncated")
	}
}

func TestOnlyRecordedCompletedCurriculumUnlocksExploration(t *testing.T) {
	s, template := templateFixture(t)
	for i := range s.state.Catalog {
		s.state.Catalog[i].Classification = "manual"
	}
	_, _ = s.Add([]Scenario{{Name: "extra", VariantOf: "Static B", Skill: "static", Technique: "static", Classification: "manual", Enabled: true, Seconds: 60, Difficulty: "unknown"}})
	p := defaults()
	p.Variety = .5
	plan, err := s.Generate(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.state.Plan.Status = "completed"
	s.state.Plan.Recorded = 0
	plan, err = s.Generate(p, nil)
	if err != nil || len(plan.Blocks) != len(template.Rows) {
		t.Fatal("empty/abandoned session unlocked exploration")
	}
	s.state.Plan.Status = "completed"
	s.state.Plan.Recorded = 60
	for i := range template.Rows {
		b := &s.state.Plan.Blocks[i]
		b.Runs, b.Recorded, b.Outcome = b.PlayCount, float64(b.PlayCount*60), "list_complete"
	}
	plan, err = s.Generate(p, nil)
	if err != nil || len(plan.Blocks) != len(template.Rows)+1 {
		t.Fatal("recorded baseline not used", err)
	}
	if _, err = s.Generate(Preferences{Minutes: 30}, nil); err == nil {
		t.Fatal("invalid request accepted")
	}
}

func TestDefaultDoesNotPretendRandomPlanIsVDIM(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Add([]Scenario{demandScene("candidate")})
	if _, err = s.Generate(defaults(), nil); err == nil {
		t.Fatal("invented original VDIM without a template")
	}
	if _, err := parseCurriculum([]byte(`{"playlistName":"VDIM","scenarioList":[{"scenarioName":"A","playCount":-1}]}`), Source{}); err == nil {
		t.Fatal("invalid count")
	}
}

func TestManagedPlaylistCountOwnershipAndActiveProtection(t *testing.T) {
	s, _ := templateFixture(t)
	dir := t.TempDir()
	p := defaults()
	p.Variety = 0
	userPath := filepath.Join(dir, "User.json")
	user := []byte(`{"playlistName":"User"}`)
	_ = os.WriteFile(userPath, user, 0600)
	for i := 0; i < 40; i++ {
		if _, err := s.Generate(p, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Install(dir); err != nil {
			t.Fatal(err)
		}
	}
	files, _ := os.ReadDir(dir)
	count := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("generated list accumulation: %d", count)
	}
	got, _ := os.ReadFile(userPath)
	if !reflect.DeepEqual(got, user) {
		t.Fatal("user playlist touched")
	}
	path, err := s.Install(dir)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	s.state.Plan.Status = "running"
	if _, err = s.Generate(p, nil); err == nil {
		t.Fatal("active plan replaced")
	}
	if _, err = s.Install(dir); err != nil {
		t.Fatal("same active export should be idempotent")
	}
	s.state.Plan.ID = "different"
	if _, err = s.Install(dir); err == nil {
		t.Fatal("active managed slot overwritten")
	}
	after, _ := os.ReadFile(path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("active file changed")
	}
	s.state.Plan.Status = "draft"
	_ = os.WriteFile(path, user, 0600)
	if _, err = s.Install(dir); err == nil {
		t.Fatal("foreign file overwritten")
	}
}

func TestLocalTemplateAutoloadUsesRealOrderedJSON(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "User.json"), []byte(`{"playlistName":"My routine","scenarioList":[{"scenarioName":"Static User","playCount":2}]}`), 0600)
	_ = os.WriteFile(filepath.Join(dir, "VDIM.json"), []byte(`{"playlistName":"VDIM Novice Clicking I","scenarioList":[{"scenarioName":"Unknown author scene","playCount":4}]}`), 0600)
	s.PollLocal(nil, dir, epoch)
	if len(s.state.Curricula) != 1 {
		t.Fatal("saved VDIM not found")
	}
	if _, err = s.Generate(defaults(), nil); err != nil {
		t.Fatal("unknown SCE blocked first author template run", err)
	}
	s.PollLocal(nil, dir, epoch.Add(15*time.Second))
	if len(s.state.Curricula) != 1 {
		t.Fatal("repeated scan duplicated template")
	}
}

func TestCanonicalVDIMTitleAndInvalidBudgetLeaveSourceIntact(t *testing.T) {
	data := []byte(`{"playlistName":"Intermediate S5 - Clicking I","scenarioList":[{"scenarioName":"Static A","playCount":5}]}`)
	template, err := parseCurriculum(data, Source{})
	if err != nil || template == nil || template.Theme != "static" {
		t.Fatal("canonical saved title rejected", err)
	}
	p := defaults()
	p.Minutes = 5
	if _, err = GenerateCurriculum(*template, []Scenario{demandScene("Static A")}, nil, p, epoch, rand.New(rand.NewSource(1)), false); err == nil {
		t.Fatal("too-short budget was silently adapted")
	}
	if template.Rows[0].Count != 5 {
		t.Fatal("source altered")
	}
}
