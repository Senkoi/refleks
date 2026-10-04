package training

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"refleks/internal/models"
	"testing"
	"time"
)

func TestAutomaticManyBenchmarksStillGenerateAndInstall(t *testing.T) {
	s, _ := templateFixture(t)
	for i := 0; i < 40; i++ {
		s.state.Catalog[0].Benchmarks = append(s.state.Catalog[0].Benchmarks, BenchmarkMembership{Name: fmt.Sprintf("System %d / Novice", i), System: fmt.Sprintf("System %c%c", 'A'+i/26, 'A'+i%26), NativeDifficulty: "Novice", Thresholds: []float64{1}})
	}
	p, err := s.Generate(defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Preferences.Benchmarks) != 40 {
		t.Fatal("automatic references were silently truncated", len(p.Preferences.Benchmarks))
	}
	path, err := s.Install(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Rows []CurriculumRow `json:"scenarioList"`
	}
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i, row := range out.Rows {
		if row.Count != p.Blocks[i].PlayCount {
			t.Fatal("installed counts differ from plan")
		}
	}
}

func TestOfficialWindowsUseShortSetsAndContinueOnlyCompletedRows(t *testing.T) {
	s, template := templateFixture(t)
	template.OfficialCode = "official"
	template.Rows = []CurriculumRow{{Name: "Static A", Count: 5}, {Name: "Static B", Count: 5}, {Name: "Static Measure", Count: 5}}
	p := defaults()
	p.Minutes = 5
	p.Variety = 0
	window, start, end, err := curriculumWindow(template, s.state.Catalog, nil, p, epoch, nil)
	if err != nil || start != 0 || end != 2 {
		t.Fatal(start, end, err)
	}
	if window.Rows[0].Count != 2 || window.Rows[1].Count != 2 || template.Rows[0].Count != 5 {
		t.Fatal("fixed five-minute sets or source mutation", window.Rows)
	}
	plan, err := GenerateCurriculum(window, s.state.Catalog, nil, p, epoch, rand.New(rand.NewSource(1)), false)
	if err != nil {
		t.Fatal(err)
	}
	plan.CurriculumStart, plan.CurriculumEnd, plan.CurriculumTotal = 0, 2, 3
	plan.Status = "completed"
	for i := range plan.Blocks {
		b := &plan.Blocks[i]
		b.Runs = b.PlayCount
		b.Recorded = float64(b.PlayCount * 60)
		b.Outcome = "list_complete"
	}
	_, start, end, err = curriculumWindow(template, s.state.Catalog, nil, p, epoch, []Plan{*plan})
	if err != nil || start != 2 || end != 3 {
		t.Fatal("adaptive completion did not advance cursor", start, end, err)
	}
	plan.Blocks[0].Runs--
	_, start, _, err = curriculumWindow(template, s.state.Catalog, nil, p, epoch, []Plan{*plan})
	if err != nil || start != 0 {
		t.Fatal("unfinished short set advanced cursor", start, err)
	}
}

func TestShortSetsUseComparableDurationsAndRecentExposure(t *testing.T) {
	s, template := templateFixture(t)
	template.Rows = []CurriculumRow{{Name: "Static B", Count: 5}}
	// Three old comparable 45-second records allow three whole runs; new
	// 60-second records and enough recent exposure reduce the same row to one.
	old := []models.RunRecord{}
	recent := []models.RunRecord{}
	for i := 0; i < 3; i++ {
		old = append(old, record(fmt.Sprint(i), "Static B", 45, 1, epoch.Add(-48*time.Hour)))
		recent = append(recent, record(fmt.Sprint(i), "Static B", 60, 1, epoch.Add(-time.Hour)))
	}
	a, _, _, err := curriculumWindow(template, s.state.Catalog, old, defaults(), epoch, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _, err := curriculumWindow(template, s.state.Catalog, recent, defaults(), epoch, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Rows[0].Count != 3 || b.Rows[0].Count != 1 {
		t.Fatal("duration/exposure policy missing", a.Rows, b.Rows)
	}
}

func TestCatalogVerificationSeparatesEvidenceFromLabelsAndBrokenFiles(t *testing.T) {
	_, m, a, err := ParseLocalSCE([]byte(localSCEFixture))
	if err != nil {
		t.Fatal(err)
	}
	a.FilePath = "download.sce"
	scene := Scenario{Name: "Static A", Mechanics: m, LocalAssessment: a, Difficulty: "advanced", DifficultySource: "manual"}
	e := assessCatalog(scene, nil, epoch, Preferences{Benchmark: "native"})
	if e.FileStatus != "verified" || e.HasDifficultyEvidence || e.DifficultyStatus != "unfitted" {
		t.Fatal("label counted as evaluation", e)
	}
	scene.Benchmarks = []BenchmarkMembership{{Name: "native", NativeDifficulty: "Novice", Thresholds: []float64{50}}}
	e = assessCatalog(scene, nil, epoch, Preferences{Benchmark: "native"})
	if !e.HasBenchmarkReference || !e.HasDifficultyEvidence || e.DifficultyStatus == "calibrated" {
		t.Fatal("native reference missing or called calibrated", e)
	}
	for _, issue := range []string{"unresolved:Bot Profile:target", "unsupported_ability:x"} {
		a.Issues = []string{issue}
		e = assessCatalog(scene, nil, epoch, Preferences{Benchmark: "native"})
		if e.FileStatus == "verified" || e.HasDifficultyEvidence {
			t.Fatal("broken configuration verified", e)
		}
	}
	a.Issues = nil
	a.Status = "ambiguous_local_versions"
	if assessCatalog(scene, nil, epoch, Preferences{Benchmark: "native"}).HasDifficultyEvidence {
		t.Fatal("conflicting file accepted")
	}
}

func TestDeletedDownloadedSceneInvalidatesPersistedAssessment(t *testing.T) {
	s, _ := templateFixture(t)
	root := t.TempDir()
	path := filepath.Join(root, "download.sce")
	if err := os.WriteFile(path, []byte(localSCEFixture), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, epoch.Add(-time.Minute), epoch.Add(-time.Minute))
	s.PollLocal([]string{root}, "", epoch)
	s.PollLocal([]string{root}, "", epoch.Add(15*time.Second))
	if s.state.Catalog[0].LocalAssessment == nil {
		t.Fatal("fixture not assessed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	reopened.PollLocal([]string{root}, "", epoch.Add(time.Minute))
	c := reopened.state.Catalog[0]
	if c.Mechanics != nil || c.LocalAssessment.Status != "local_file_unavailable" || assessCatalog(c, nil, epoch, Preferences{Benchmark: "native"}).FileStatus == "verified" {
		t.Fatal("deleted file stayed assessed", c.LocalAssessment)
	}
}

func TestNonFiniteSettingsDoNotInferPlayerRank(t *testing.T) {
	for _, field := range []string{"duration", "remaining", "scale", "dilation"} {
		r := record("bad", "Static A", 60, 100, epoch)
		switch field {
		case "duration":
			r.Stats.Summary.Duration = math.NaN()
		case "remaining":
			r.Stats.Summary.TimeRemaining = math.NaN()
		case "scale":
			r.Stats.Summary.AvgTargetScale = math.NaN()
		case "dilation":
			r.Stats.Summary.AvgTimeDilation = math.Inf(1)
		}
		if len(levelObservations([]models.RunRecord{r}, epoch)) != 0 {
			t.Fatal("invalid settings counted", field)
		}
	}
}

func TestSameBroadSkillIsNotProofOfExplorationGoal(t *testing.T) {
	a := Scenario{Name: "base", Skill: "static", Technique: "static", Classification: "manual", Enabled: true}
	b := a
	b.Name = "unrelated"
	if goalCompatible(a, b) {
		t.Fatal("broad category treated as shared goal")
	}
	b.VariantOf = a.Name
	if !goalCompatible(a, b) {
		t.Fatal("explicit variant rejected")
	}
}

func TestBundledTemplatesGenerateShortSetsAcrossBudgetsAndThemes(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InitializeTraining(); err != nil {
		t.Fatal(err)
	}
	for _, minutes := range []int{5, 15, 30, 60, 120} {
		for _, theme := range vdimThemes {
			p := defaults()
			p.Minutes = minutes
			p.Variety = 0
			if theme == "switching_speed" || theme == "switching_evasive" {
				p.Focus = "switching"
			} else {
				p.Focus = theme
			}
			plan, err := s.Generate(p, nil)
			if err != nil {
				t.Fatalf("%s / %d: %v", theme, minutes, err)
			}
			cost := 0
			for _, b := range plan.Blocks {
				if b.PlayCount < 1 || b.PlayCount > 2 || b.SourcePlayCount < b.PlayCount || b.Budget != b.PlayCount*b.Timing.Seconds {
					t.Fatal("official path missed short sets", b)
				}
				cost += b.Budget
			}
			if cost > minutes*60*9/10 {
				t.Fatal("budget overflow")
			}
		}
	}
}

func TestAutomaticSelectionChecksPendingWindowInsteadOfBeginning(t *testing.T) {
	s, base := templateFixture(t)
	base.Name = "AAA first"
	base.Rows = []CurriculumRow{{Name: "Static A", Count: 1}, {Name: "Static B", Count: 1}}
	alternative := base
	alternative.ID = "alternative"
	alternative.Name = "ZZZ alternative"
	alternative.Rows = []CurriculumRow{{Name: "Static A", Count: 1}}
	s.state.Curricula = []Curriculum{base, alternative}
	for i := range s.state.Catalog {
		if s.state.Catalog[i].Name == "Static B" {
			s.state.Catalog[i].Seconds = 600
		}
	}
	s.state.Plan = &Plan{CurriculumID: base.ID, Status: "completed", CurriculumStart: 0, CurriculumEnd: 1, CurriculumTotal: 2, Blocks: []Block{{Scenario: Scenario{Name: "Static A"}, Role: "practice", PlayCount: 1, SourcePlayCount: 1, Runs: 1, Recorded: 60, Outcome: "list_complete"}}}
	p := defaults()
	p.Minutes = 5
	p.Focus = "static"
	plan, err := s.Generate(p, nil)
	if err != nil || plan.CurriculumID != alternative.ID {
		t.Fatal("selected template beginning fits but pending row does not", err)
	}
}
