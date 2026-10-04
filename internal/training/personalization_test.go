package training

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aimmeow/internal/models"
)

func personalScene(t *testing.T, name string, radius int, change string, now time.Time) Scenario {
	t.Helper()
	data := strings.Replace(localSCEFixture, "Name=Static A", "Name="+name, 1)
	data = strings.Replace(data, "MainBBRadius=10", fmt.Sprintf("MainBBRadius=%d", radius), 1)
	if change != "" {
		data = strings.Replace(data, "Velocity=3500", change, 1)
	}
	n, m, a, err := ParseLocalSCE([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	a.FilePath = filepath.Join(t.TempDir(), name+".sce")
	a.ObservedAt = now.AddDate(0, 0, -10).Format(time.RFC3339)
	return scenarioFromLocal(n, a.FilePath, m, a)
}

func anchorRuns(name string, now time.Time) []models.RunRecord {
	out := []models.RunRecord{}
	for day := 1; day <= 3; day++ {
		for i := 0; i < 3; i++ {
			r := record(fmt.Sprintf("%s-%d-%d", name, day, i), name, 60, 100, now.Add(-time.Duration(day)*24*time.Hour-time.Duration(i)*time.Minute))
			r.Stats.Summary.GameVersion = "fixture"
			out = append(out, r)
		}
	}
	return out
}

func TestControlledComparisonChecksMapMechanismsAndReachableSize(t *testing.T) {
	a := personalScene(t, "anchor", 10, "", epoch)
	b := personalScene(t, "smaller", 8, "", epoch)
	r := precisionRelation(a, b)
	if r == nil || !r.Uniform || r.Direction != "higher_precision" || math.Abs(r.UniformDelta-math.Log2(1.25)) > 1e-6 {
		t.Fatal("controlled pair missing", r)
	}
	for _, change := range []string{"Velocity=4000", "Velocity=3500\nExtraMechanism=1"} {
		c := personalScene(t, "confounded", 8, change, epoch)
		if precisionRelation(a, c) != nil {
			t.Fatal("mechanism change admitted", change)
		}
	}
	data := strings.Replace(localSCEFixture, "Name=Static A", "Name=changed map", 1)
	data = strings.Replace(data, "MainBBRadius=10", "MainBBRadius=8", 1)
	for _, tc := range []string{
		strings.Replace(data, "Name=this is map text not a root config", "Name=other layout", 1),
		strings.Replace(data, "MainBBRadius=9999", "MainBBRadius=99", 1),
		strings.Replace(data, "Timelimit=60", "Timelimit=60\nTimelimit=60", 1),
	} {
		n, m, assess, err := ParseLocalSCE([]byte(tc))
		if err != nil {
			t.Fatal(err)
		}
		assess.FilePath = "local"
		c := scenarioFromLocal(n, "local", m, assess)
		if precisionRelation(a, c) != nil {
			t.Fatal("map/unused/default ambiguity admitted")
		}
	}
	b.LocalAssessment.Status = "ambiguous_local_versions"
	if precisionRelation(a, b) != nil {
		t.Fatal("conflicting local files admitted")
	}
}

func TestPersonalAnchorBalancesSessionsAndPreservesVersionUncertainty(t *testing.T) {
	now := epoch
	scene := personalScene(t, "anchor", 10, "", now)
	runs := []models.RunRecord{
		record("day1", "anchor", 60, 100, now.Add(-48*time.Hour)),
		record("day2", "anchor", 60, 110, now.Add(-24*time.Hour)),
	}
	for i := 0; i < 75; i++ {
		runs = append(runs, record(fmt.Sprintf("burst-%d", i), "anchor", 60, 200, now.Add(-time.Duration(i)*time.Minute/4)))
	}
	a := personalAnchors([]Scenario{scene}, runs, nil, now)["anchor"]
	if a.MedianScore != 110 || a.Days != 3 || a.Evidence != "history_unbound" || a.FileSHA256 != "" {
		t.Fatal("retries or current file rewrote baseline", a)
	}
	if len(a.Points) != 3 || a.Points[0].Score != 100 || a.Points[1].Score != 110 || a.Points[2].Score != 200 || a.Points[2].Samples != 12 || a.Points[0].At >= a.Points[1].At || a.Points[1].At >= a.Points[2].At {
		t.Fatal("chart did not preserve balanced, chronological anchor evidence", a.Points)
	}
	r := record("new revision", "anchor", 60, 50, now.Add(time.Second))
	r.Stats.Summary.Hash = "v2"
	runs = append(runs, r)
	a = personalAnchors([]Scenario{scene}, runs, nil, now.Add(2*time.Second))["anchor"]
	if a.Samples != 1 || a.Status != "provisional" {
		t.Fatal("different revisions mixed", a)
	}
	if len(a.Points) != 1 || a.Points[0].Score != 50 {
		t.Fatal("chart connected different revisions", a.Points)
	}
	runs = anchorRuns("anchor", now)
	runs[0].Stats.Summary.TimeRemaining = 20
	runs[1].Stats.Events = []models.RunStatsEvent{{Cheated: true}}
	runs[2].Stats.Summary.AvgTargetScale = .5
	a = personalAnchors([]Scenario{scene}, runs, nil, now)["anchor"]
	if a.Samples != 6 {
		t.Fatal("invalid observations entered baseline", a)
	}
}

func TestFasterEffectiveHitsDoNotBecomeAccuracyRegression(t *testing.T) {
	runs := sessionRuns("A", []float64{100, 100, 100, 110, 110, 110})
	for i := range runs {
		runs[i].Stats.Summary.HitCount = 80
		runs[i].Stats.Summary.MissCount = 20
		if i >= 3 {
			runs[i].Stats.Summary.HitCount = 120
			runs[i].Stats.Summary.MissCount = 120
		}
	}
	trend, _ := performanceTrend(levelObservations(runs, epoch)["a"], epoch)
	if trend != "improving" {
		t.Fatal("speed/accuracy improvement was downgraded", trend)
	}
}

func TestBoundAnchorDoesNotRelabelUnboundHistory(t *testing.T) {
	scene := personalScene(t, "anchor", 10, "", epoch)
	runs := []models.RunRecord{record("old-unbound", "anchor", 60, 500, epoch.Add(-48*time.Hour))}
	contexts := map[string]RunContext{}
	for i := 0; i < 3; i++ {
		r := record(fmt.Sprintf("bound-%d", i), "anchor", 60, 100, epoch.Add(-time.Duration(i+1)*time.Minute))
		runs = append(runs, r)
		p := practiceSample(r)
		contexts[p.RunID] = RunContext{Scenario: "anchor", Signature: p.Signature, FileSHA256: scene.LocalAssessment.FileSHA256, At: p.At}
	}
	a := personalAnchors([]Scenario{scene}, runs, contexts, epoch)["anchor"]
	if a.FileSHA256 != "" || a.Evidence != "history_unbound" {
		t.Fatal("one-day bindings relabeled cross-day history", a)
	}
	runs[3].Stats.Summary.DatePlayed = epoch.Add(-25 * time.Hour).Format(time.RFC3339)
	p := practiceSample(runs[3])
	contexts[p.RunID] = RunContext{Scenario: "anchor", Signature: p.Signature, FileSHA256: scene.LocalAssessment.FileSHA256, At: p.At}
	a = personalAnchors([]Scenario{scene}, runs, contexts, epoch)["anchor"]
	if a.FileSHA256 == "" || a.Evidence != "local_execution_context" || a.MedianScore != 100 || a.Samples != 3 || a.Days != 2 {
		t.Fatal("bound anchor mixed unbound observations", a)
	}
	for _, point := range a.Points {
		if point.Score != 100 {
			t.Fatal("chart relabeled old unbound scores as current-file evidence", a.Points)
		}
	}
}

func TestPersonalTrialUsesHistoryWithoutMandatoryPretest(t *testing.T) {
	now := epoch
	a := personalScene(t, "anchor", 10, "", now)
	b := personalScene(t, "near", 9, "", now)
	c := personalScene(t, "far", 5, "", now)
	template := Curriculum{ID: "pilot", Theme: "static", Tier: "novice", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	catalog := []Scenario{a, b, c}
	p := defaults()
	bundle := preparePersonalization(template, catalog, anchorRuns(a.Name, now), nil, nil, now, p, true, 60, rand.New(rand.NewSource(1)))
	if bundle.study == nil || bundle.study.TrainingScenario != "near" || len(bundle.before) != 0 || bundle.after[0].PlayCount != 2 || bundle.seconds() != 120 || bundle.study.ProtocolID != dailyTrialProtocol || bundle.study.Baseline.ProtocolID != "history_summary_v2" {
		t.Fatal("nearest daily trial or history reference missing", bundle)
	}
	for _, tc := range []struct {
		minutes  int
		variety  float64
		explored bool
	}{{5, .1, true}, {30, 0, true}, {30, .1, false}} {
		p.Minutes, p.Variety = tc.minutes, tc.variety
		got := preparePersonalization(template, catalog, anchorRuns(a.Name, now), nil, nil, now, p, tc.explored, 60, rand.New(rand.NewSource(1)))
		if got.study != nil {
			t.Fatal("first-run, disabled or short budget bypassed", tc)
		}
	}
	p = defaults()
	st := *bundle.study
	st.Status = "measured"
	st.Feedback = "hard"
	st.TrainedAt = now.Add(-time.Hour).UnixMilli()
	st.Trial = &MeasurementResult{Score: 90}
	got := preparePersonalization(template, catalog, anchorRuns(a.Name, now), nil, []TrainingStudy{st}, now, p, true, 60, rand.New(rand.NewSource(1)))
	if got.study != nil && got.study.TrainingScenario == "near" {
		t.Fatal("hard feedback ignored")
	}
	st.Feedback = "easy"
	got = preparePersonalization(template, catalog, anchorRuns(a.Name, now), nil, []TrainingStudy{st}, now, p, true, 60, rand.New(rand.NewSource(1)))
	if got.study == nil || got.study.TrainingScenario != "far" {
		t.Fatal("easy response never advances to next controlled change")
	}
}

func TestTrialCaptureRetestPersistenceAndFoundationProgress(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	a := personalScene(t, "anchor", 10, "", now)
	b := personalScene(t, "variant", 8, "", now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	template := Curriculum{ID: "pilot", ContentSHA256: "template-hash", Theme: "static", Tier: "novice", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	s.state.Catalog = []Scenario{a, b}
	s.state.Curricula = []Curriculum{template}
	s.state.CurriculumProgress = map[string]RoutineProgress{progressKey(template): {Cycle: 1, EverCompleted: true, Rows: []RowProgress{{Target: 2, Completed: 2}}}}
	runs := anchorRuns(a.Name, now)
	stLegacy := TrainingStudy{ID: "legacy-study", PlanID: "legacy-plan", Theme: "static", AnchorScenario: a.Name, TrainingScenario: b.Name,
		AnchorHash: a.LocalAssessment.FileSHA256, TrainingHash: b.LocalAssessment.FileSHA256, Status: "planned", CreatedAt: now.UnixMilli()}
	plan, err := GenerateCurriculum(template, []Scenario{a, b}, runs, defaults(), now, rand.New(rand.NewSource(1)), false)
	if err != nil {
		t.Fatal(err)
	}
	plan.ID, plan.PlannerVersion, plan.CurriculumCycle, plan.CurriculumTotal = "legacy-plan", 7, 2, 1
	rowIndex := 0
	plan.Blocks[0].CurriculumRow = &rowIndex
	trial := Block{Scenario: b, Role: "explore", PlayCount: 2, Budget: 120, Outcome: "pending", Measurement: &MeasurementSpec{StudyID: stLegacy.ID, Phase: "trial"}}
	plan.Blocks = append([]Block{measurementBlock(stLegacy, a, "baseline", nil, now)}, append(plan.Blocks, trial)...)
	s.state.Plan, s.state.TrainingStudies = plan, []TrainingStudy{stLegacy}
	if _, err = s.Action("start", now, runs); err != nil {
		t.Fatal(err)
	}
	end := now
	for _, tc := range []struct {
		name  string
		score float64
	}{{a.Name, 9999}, {a.Name, 100}, {a.Name, 100}, {a.Name, 100}, {a.Name, 100}, {b.Name, 80}, {b.Name, 80}} {
		end = end.Add(time.Minute)
		r := record(fmt.Sprintf("executed-%d", end.Unix()), tc.name, 60, tc.score, end)
		r.Stats.Summary.GameVersion = "fixture"
		runs = append(runs, r)
		s.Tick(end, runs)
		if len(runs) == 12 && s.state.CurriculumProgress[progressKey(template)].Rows[0].Completed != 0 {
			t.Fatal("measurement counted as foundation")
		}
	}
	st := s.state.TrainingStudies[0]
	if st.Status != "waiting" || st.Baseline == nil || st.Baseline.Score != 100 || st.Trial == nil || st.Trial.Score != 80 || s.state.CurriculumProgress[progressKey(template)].Rows[0].Completed != 2 {
		t.Fatal("capture or progress incorrect", st)
	}
	due := end.Add(25 * time.Hour)
	s.updateStudiesLocked(due)
	if s.state.TrainingStudies[0].Status != "due" {
		t.Fatal("legacy window changed")
	}
	bundle := personalizationBundle{before: []Block{measurementBlock(s.state.TrainingStudies[0], a, "retest", nil, due)}}
	s.state.Plan = &Plan{ID: "retest-plan", Status: "running", Preferences: defaults(), Blocks: bundle.before, AcceptAfter: due.UnixMilli(), LastTick: due.UnixMilli()}
	for i, score := range []float64{9999, 102, 104} {
		at := due.Add(time.Duration(i+1) * time.Minute)
		r := record(fmt.Sprintf("retest-%d", i), a.Name, 60, score, at)
		r.Stats.Summary.GameVersion = "fixture"
		runs = append(runs, r)
		s.Tick(at, runs)
	}
	st = s.state.TrainingStudies[0]
	if st.Status != "measured" || st.Retest == nil || st.Retest.Score != 103 || st.RetentionChange == nil || math.Abs(*st.RetentionChange-.03) > 1e-9 {
		t.Fatal("warmup excluded/retention incorrect", st)
	}
	if err = s.TrialFeedback(st.ID, "suitable"); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reopened.state.TrainingStudies[0].Status != "measured" || reopened.state.TrainingStudies[0].Feedback != "suitable" {
		t.Fatal("study lost across restart")
	}
}

func TestResponseModelRequiresIndependentDaysValidationAndRange(t *testing.T) {
	now := epoch
	a := PersonalAnchor{Scenario: "A", MedianScore: 100, Signature: "v1|settings", FileSHA256: "ha"}
	relation := &PrecisionRelation{FamilyFingerprint: "family", Uniform: true, UniformDelta: .3}
	studies := []TrainingStudy{}
	for day := 0; day < 3; day++ {
		for i, x := range []float64{.2, .4} {
			at := now.Add(-time.Duration(3-day)*24*time.Hour + time.Duration(i)*time.Hour).UnixMilli()
			studies = append(studies, TrainingStudy{ID: fmt.Sprintf("%d-%d", day, i), AnchorScenario: "A", AnchorHash: "ha", TrainingHash: "hb",
				Relation: PrecisionRelation{FamilyFingerprint: "family", Uniform: true, UniformDelta: x},
				Baseline: &MeasurementResult{Score: 100, Signature: a.Signature, Settings: "settings", FileSHA256: "ha"},
				Trial:    &MeasurementResult{Score: 100 * math.Exp(-.5*x), At: at, Settings: "settings", FileSHA256: "hb"}})
		}
	}
	p := responsePrediction(studies, a, relation)
	if p.Status != "local_backtest" || p.ExpectedScore <= 0 || p.ExpectedScore >= 100 || p.ValidationMAE >= p.BaselineMAE {
		t.Fatal("local response never validated", p)
	}
	relation.UniformDelta = .6
	if got := responsePrediction(studies, a, relation); got.Status != "out_of_range" {
		t.Fatal("model extrapolated", got)
	}
	relation.UniformDelta = .3
	if got := responsePrediction(studies[:4], a, relation); got.Status != "insufficient" {
		t.Fatal("too few trials fitted", got)
	}
	for _, tc := range []string{"one_per_day", "baseline_better", "zero_score"} {
		t.Run(tc, func(t *testing.T) {
			copy := append([]TrainingStudy(nil), studies...)
			for i := range copy {
				trial := *copy[i].Trial
				copy[i].Trial = &trial
				if tc == "one_per_day" {
					trial.At = now.Add(-time.Duration(6-i) * 24 * time.Hour).UnixMilli()
				} else if tc == "baseline_better" && i >= 4 {
					trial.Score = 100.1
				} else if tc == "zero_score" && i == 5 {
					trial.Score = 0
				}
			}
			want := map[string]string{"one_per_day": "local_backtest", "baseline_better": "baseline_better", "zero_score": "non_positive"}[tc]
			if got := responsePrediction(copy, a, relation); got.Status != want {
				t.Fatalf("%s: %s, want %s", tc, got.Status, want)
			}
		})
	}
	for i := range studies {
		studies[i].Trial.At = now.Add(time.Duration(i) * time.Minute).UnixMilli()
	}
	if got := responsePrediction(studies, a, relation); got.Status != "insufficient" {
		t.Fatal("same-day retries masqueraded as independent days", got)
	}
	relation.FamilyFingerprint = "unseen family"
	if got := responsePrediction(studies, a, relation); got.Status != "insufficient" {
		t.Fatal("cross-family transfer fabricated", got)
	}
}

func TestReviewCategoryNeverOverridesRecentContinuation(t *testing.T) {
	a := Scenario{Name: "A", Skill: "static", Technique: "static", Seconds: 60, Enabled: true}
	b := Scenario{Name: "B", Skill: "dynamic", Technique: "dynamic", Seconds: 60, Enabled: true}
	ts := []Curriculum{{ID: "a", Theme: "static", Tier: "novice", Rows: []CurriculumRow{{Name: "A", Count: 2}}}, {ID: "b", Theme: "dynamic", Tier: "novice", Rows: []CurriculumRow{{Name: "B", Count: 2}}}}
	progress := map[string]RoutineProgress{progressKey(ts[0]): {LastPracticed: epoch.Add(-2 * time.Hour).UnixMilli(), Rows: []RowProgress{{Target: 2, Completed: 1}}}}
	p := defaults()
	p.ReviewTheme = "dynamic"
	chosen, err := selectCurriculumWithHistory(ts, []Scenario{a, b}, p, nil, epoch, nil, progress)
	if err != nil || chosen.ID != "a" {
		t.Fatal("review overrode continuation", chosen, err)
	}
	r := progress[progressKey(ts[0])]
	r.LastPracticed = epoch.Add(-25 * time.Hour).UnixMilli()
	progress[progressKey(ts[0])] = r
	chosen, err = selectCurriculumWithHistory(ts, []Scenario{a, b}, p, nil, epoch, nil, progress)
	baseline := p
	baseline.ReviewTheme = ""
	expected, expectedErr := selectCurriculumWithHistory(ts, []Scenario{a, b}, baseline, nil, epoch, nil, progress)
	if err != nil || expectedErr != nil || chosen.ID != expected.ID {
		t.Fatal("review changed weighted category selection", chosen, expected, err)
	}
}

func TestChangedSettingsRetestAndTransferExposureDoNotProduceBenefit(t *testing.T) {
	now := epoch
	a := personalScene(t, "anchor", 10, "", now)
	b := personalScene(t, "variant", 8, "", now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Catalog = []Scenario{a, b}
	baseRun := record("base", "anchor", 60, 100, now.Add(-26*time.Hour))
	baseRun.Stats.Summary.GameVersion = "fixture"
	sample := practiceSample(baseRun)
	st := TrainingStudy{ID: "study", Theme: "static", AnchorScenario: "anchor", TrainingScenario: "variant", AnchorHash: a.LocalAssessment.FileSHA256, TrainingHash: b.LocalAssessment.FileSHA256, Status: "waiting",
		Baseline: &MeasurementResult{Score: 100, Signature: sample.Signature, Settings: sample.Settings, FileSHA256: a.LocalAssessment.FileSHA256, At: sample.At},
		Trial:    &MeasurementResult{Score: 80, Settings: sample.Settings, FileSHA256: b.LocalAssessment.FileSHA256, At: now.Add(-25 * time.Hour).UnixMilli()}}
	s.state.TrainingStudies = []TrainingStudy{st}
	s.updateStudiesLocked(now)
	st = s.state.TrainingStudies[0]
	block := measurementBlock(st, a, "retest", nil, now)
	for i := 0; i < 3; i++ {
		r := record(fmt.Sprintf("changed-%d", i), "anchor", 60, 120, now.Add(time.Duration(i)*time.Minute))
		r.Stats.Summary.GameVersion = "fixture"
		r.Stats.Summary.HorizSens = 2
		p := practiceSample(r)
		p.FileSHA256 = st.AnchorHash
		block.Observations = append(block.Observations, p)
	}
	s.state.Plan = &Plan{Blocks: []Block{block}}
	s.updateStudiesLocked(now.Add(3 * time.Minute))
	if s.state.TrainingStudies[0].Retest != nil || s.state.TrainingStudies[0].Status != "settings_changed" {
		t.Fatal("different settings produced retention", s.state.TrainingStudies[0])
	}
	value := .2
	s.state.TrainingStudies = []TrainingStudy{{ID: "transfer", TransferScenario: "holdout", TransferBaseline: &MeasurementResult{At: now.Add(-48 * time.Hour).UnixMilli()}, TransferChange: &value}}
	r := record("outside-plan", "holdout", 60, 50, now.Add(-24*time.Hour))
	s.auditTransferExposureLocked([]models.RunRecord{r}, now)
	if !s.state.TrainingStudies[0].TransferContaminated || s.state.TrainingStudies[0].TransferChange != nil {
		t.Fatal("direct holdout practice was called transfer")
	}
	// A changed warmup cannot be silently omitted from comparability checks.
	block.Observations[0].Signature = "different warmup"
	if measured(block.Observations, 1, 2, st.AnchorHash) != nil {
		t.Fatal("warmup setting change ignored")
	}
}

func TestDynamicLocalReferencesRemainConditionalAndImmutable(t *testing.T) {
	a := personalScene(t, "anchor", 10, "", epoch)
	b := personalScene(t, "variant", 8, "", epoch)
	old := a.LocalAssessment
	catalog := []Scenario{a, b}
	refreshLocalRelations(catalog)
	if len(old.PrecisionComparisons) != 0 || len(catalog[1].LocalAssessment.PrecisionComparisons) != 1 {
		t.Fatal("live evidence was mutated or local reference missing")
	}
	e := assessCatalog(catalog[1], nil, epoch, defaults())
	if e.DifficultyStatus != "precision_reference" || !e.HasPrecisionReference {
		t.Fatal("local comparison not exposed", e)
	}
	catalog[0].LocalAssessment = &LocalAssessment{Status: "local_file_unavailable"}
	refreshLocalRelations(catalog)
	if len(catalog[1].LocalAssessment.PrecisionComparisons) != 0 {
		t.Fatal("deleted anchor left a dynamic reference")
	}
}

func TestUploadedControlledPrecisionPairs(t *testing.T) {
	path := os.Getenv("REFLEKS_TEST_SCE_ARCHIVE")
	if path == "" {
		t.Skip("uploaded archive not available")
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	index := map[string]Scenario{}
	for _, file := range z.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".sce") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		n, m, a, err := ParseLocalSCE(data)
		if err != nil {
			t.Fatal(file.Name, err)
		}
		a.FilePath = file.Name
		index[n] = scenarioFromLocal(n, file.Name, m, a)
	}
	data, err := os.ReadFile("../../docs/research/difficulty-feature-audit-2026-10-02/controlled-precision-pairs.json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct{ Pairs []struct{ A, B string } }
	if err = json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	for _, pair := range report.Pairs {
		r := precisionRelation(index[pair.A], index[pair.B])
		if r == nil {
			t.Fatalf("audited controlled pair unavailable: %s / %s; issues %v / %v", pair.A, pair.B, index[pair.A].LocalAssessment.Issues, index[pair.B].LocalAssessment.Issues)
		}
		if r.Direction != "higher_precision" {
			t.Fatal("audited direction changed", pair, r)
		}
	}
	t.Logf("checked %d real scenarios and %d controlled pairs", len(index), len(report.Pairs))
}
