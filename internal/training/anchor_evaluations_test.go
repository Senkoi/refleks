package training

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"refleks/internal/models"
)

func evaluationFixture(t *testing.T, now time.Time, minutes, count, seconds int) (*Service, Curriculum, []models.RunRecord) {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := personalScene(t, "anchor", 10, "", now)
	a.Seconds = seconds
	s.state.Catalog = []Scenario{a}
	s.state.RunContexts = map[string]RunContext{}
	c := Curriculum{ID: "foundation", ContentSHA256: "template", Theme: "static", Tier: "novice", Rows: []CurriculumRow{{Name: a.Name, Count: count}}}
	s.state.Curricula = []Curriculum{c}
	index := 0
	p := defaults()
	p.Minutes = minutes
	s.state.Plan = &Plan{ID: "plan", PlannerVersion: 8, CurriculumID: c.ID, CurriculumHash: c.ContentSHA256, CurriculumTotal: 1,
		Created: now.Format(time.RFC3339), Status: "draft", Theme: c.Theme, Preferences: p,
		Blocks: []Block{{Scenario: a, Role: "practice", PlayCount: count, SourcePlayCount: count, CurriculumRow: &index,
			Budget: count * seconds, Timing: TimingEstimate{Seconds: seconds}, Outcome: "pending"}}}
	// Three natural training days justify updating a shared anchor, not testing
	// every individual probe. No study receives the later window's score gain.
	for day := 1; day <= 3; day++ {
		at := now.Add(-time.Duration(day) * 24 * time.Hour).UnixMilli()
		s.state.TrainingStudies = append(s.state.TrainingStudies,
			TrainingStudy{ID: fmt.Sprint(day), ProtocolID: dailyTrialProtocol, AnchorScenario: a.Name, Trial: &MeasurementResult{At: at}, Status: "observed"})
	}
	return s, c, anchorRuns(a.Name, now)
}

func TestAssessmentBudgetsAndRollingFrequency(t *testing.T) {
	for _, tc := range []struct {
		minutes, seconds int
		want             bool
	}{{5, 30, false}, {10, 30, true}, {30, 60, true}, {60, 180, false}} {
		t.Run(fmt.Sprintf("%dmin-%dsec", tc.minutes, tc.seconds), func(t *testing.T) {
			s, _, runs := evaluationFixture(t, epoch, tc.minutes, 2, tc.seconds)
			s.scheduleAnchorEvaluationLocked(s.state.Plan, runs, epoch)
			b := s.state.Plan.Blocks[0]
			if (b.Assessment != nil) != tc.want {
				t.Fatalf("assessment=%v, want=%v", b.Assessment, tc.want)
			}
			if tc.want && (b.PlayCount != 4 || mainPlayCount(b) != 2 || b.Budget > tc.minutes*60*9/10 || b.Assessment.ExtraRuns*tc.seconds > min(180, tc.minutes*60/10)) {
				t.Fatal("budget/share accounting", b)
			}
		})
	}
	for _, tc := range []struct {
		name    string
		records []AnchorEvaluation
		want    bool
	}{
		{"same category", []AnchorEvaluation{{Theme: "static", ExtraRuns: 2, StartedAt: epoch.Add(-24 * time.Hour).UnixMilli()}}, false},
		{"global two", []AnchorEvaluation{{Theme: "smooth", ExtraRuns: 2, StartedAt: epoch.Add(-24 * time.Hour).UnixMilli()}, {Theme: "reactive", ExtraRuns: 1, StartedAt: epoch.Add(-48 * time.Hour).UnixMilli()}}, false},
		{"old", []AnchorEvaluation{{Theme: "static", ExtraRuns: 2, StartedAt: epoch.Add(-8 * 24 * time.Hour).UnixMilli()}}, true},
		{"passive", []AnchorEvaluation{{Theme: "static", ExtraRuns: 0, StartedAt: epoch.Add(-24 * time.Hour).UnixMilli()}}, true},
		{"unstarted", []AnchorEvaluation{{Theme: "static", ExtraRuns: 2, Status: "planned"}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, runs := evaluationFixture(t, epoch, 30, 2, 60)
			s.state.AnchorEvaluations = tc.records
			s.scheduleAnchorEvaluationLocked(s.state.Plan, runs, epoch)
			if (s.state.Plan.Blocks[0].Assessment != nil) != tc.want {
				t.Fatal("rolling limit", tc.name)
			}
		})
	}
	s, _, runs := evaluationFixture(t, epoch, 30, 2, 60)
	s.state.Plan.Blocks = append(s.state.Plan.Blocks, Block{Role: "practice", Budget: 1500})
	s.scheduleAnchorEvaluationLocked(s.state.Plan, runs, epoch)
	if s.state.Plan.Blocks[0].Assessment != nil {
		t.Fatal("assessment exceeded total budget")
	}
}

func TestSharedAssessmentExecutionCountsOnceAndRetainsLowScores(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	s, c, runs := evaluationFixture(t, now, 30, 2, 60)
	s.scheduleAnchorEvaluationLocked(s.state.Plan, runs, now)
	if _, err := s.Action("start", now, runs); err != nil {
		t.Fatal(err)
	}
	for i, score := range []float64{9999, 102, 0, 100, 9000} {
		at := now.Add(time.Duration(i+1) * time.Minute)
		r := record(fmt.Sprintf("fixed-%d", i), "anchor", 60, score, at)
		r.Stats.Summary.GameVersion = "fixture"
		runs = append(runs, r)
		s.Tick(at, runs)
		progress := s.state.CurriculumProgress[progressKey(c)]
		if progress.Rows[0].Completed > 2 {
			t.Fatal("assessment advanced foundation twice", progress)
		}
	}
	e := s.state.AnchorEvaluations[0]
	if e.Result == nil || e.Result.Score != 100 || len(e.Result.Scores) != 3 || e.Result.Scores[0] != 102 || e.Result.Scores[1] != 0 || len(e.Result.RunIDs) != 4 {
		t.Fatal("low scores/retries/familiarization mishandled", e)
	}
	if s.state.CurriculumProgress[progressKey(c)].Rows[0].Completed != 2 {
		t.Fatal("original main counts missing")
	}
	if err := s.persist(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if reopened.state.AnchorEvaluations[0].Result.Score != 100 || reopened.state.Plan.Blocks[0].Assessment.MainPlayCount != 2 {
		t.Fatal("observation or accounting lost on restart")
	}
	view, err := s.WorkbenchJSON(runs)
	if err != nil || strings.Contains(view, "runIds") || strings.Contains(view, "runContexts") || len(s.state.AnchorEvaluations[0].Result.RunIDs) != 4 {
		t.Fatal("workbench leaked IDs or mutated stored observations", err)
	}
}

func TestAssessmentInvalidAttemptCannotBeReplacedWithRetry(t *testing.T) {
	for _, issue := range []string{"pause", "scale", "settings", "file", "incomplete", "first_incomplete"} {
		t.Run(issue, func(t *testing.T) {
			now := time.Now().Truncate(time.Second)
			s, _, runs := evaluationFixture(t, now, 30, 2, 60)
			s.scheduleAnchorEvaluationLocked(s.state.Plan, runs, now)
			_, _ = s.Action("start", now, runs)
			for i := 0; i < 5; i++ {
				at := now.Add(time.Duration(i+1) * time.Minute)
				r := record(fmt.Sprintf("%s-%d", issue, i), "anchor", 60, 100, at)
				r.Stats.Summary.GameVersion = "fixture"
				if i == 0 && issue == "first_incomplete" {
					r.Stats.Summary.TimeRemaining = 20
				}
				if i == 1 {
					switch issue {
					case "pause":
						r.Stats.Summary.PauseCount = 1
					case "scale":
						r.Stats.Summary.AvgTargetScale = .5
					case "settings":
						r.Stats.Summary.HorizSens = 2
					case "file":
						a := *s.state.Catalog[0].LocalAssessment
						a.FileSHA256 = "changed"
						s.state.Catalog[0].LocalAssessment = &a
					case "incomplete":
						r.Stats.Summary.TimeRemaining = 20
					}
				}
				runs = append(runs, r)
				s.Tick(at, runs)
			}
			if s.state.AnchorEvaluations[0].Result != nil {
				t.Fatal("invalid/omitted attempt replaced with a high retry")
			}
		})
	}
}

func addOrdinaryObservation(t *testing.T, s *Service, id string, at time.Time, score float64, position int) []models.RunRecord {
	t.Helper()
	a := s.state.Catalog[0]
	plan := Plan{ID: id, PlannerVersion: 8, Status: "completed", Created: at.Format(time.RFC3339), Theme: "static", Preferences: defaults(), Blocks: make([]Block, position+1)}
	b := Block{Scenario: a, Role: "practice", PlayCount: 4, Runs: 4, Recorded: 240, Outcome: "list_complete"}
	runs := []models.RunRecord{}
	for i := 0; i < 4; i++ {
		end := at.Add(time.Duration(i+1) * time.Minute)
		r := record(fmt.Sprintf("%s-%d", id, i), a.Name, 60, score+float64(i-2), end)
		r.Stats.Summary.GameVersion = "fixture"
		p := practiceSample(r)
		p.FileSHA256 = a.LocalAssessment.FileSHA256
		b.Observations = append(b.Observations, p)
		runs = append(runs, r)
		s.state.RunContexts[p.RunID] = RunContext{ContextVersion: 2, Scenario: a.Name, At: p.At, Signature: p.Signature, FileSHA256: p.FileSHA256, PlanID: id, ProtocolID: anchorObservationProtocol, BlockPosition: position, BlockRun: i + 1}
	}
	plan.Blocks[position] = b
	s.state.History = append(s.state.History, plan)
	return runs
}

func TestNaturalObservationsHaveNoDeadlineAndSeparateContext(t *testing.T) {
	for _, hours := range []int{20, 23, 24, 72, 144} {
		t.Run(fmt.Sprint(hours), func(t *testing.T) {
			s, _, _ := evaluationFixture(t, epoch, 30, 2, 60)
			s.state.Plan = nil
			runs := addOrdinaryObservation(t, s, "before", epoch.Add(-time.Duration(hours)*time.Hour), 100, 0)
			runs = append(runs, addOrdinaryObservation(t, s, "after", epoch, 110, 0)...)
			now := epoch.Add(5 * time.Minute)
			s.syncSessionContextsLocked(runs, now)
			s.updateAnchorEvaluationsLocked(runs, now)
			e := s.state.AnchorEvaluations[1]
			if e.Previous == nil || math.Abs(e.IntervalHours-float64(hours)) > 1e-9 || e.Change == nil || math.Abs(*e.Change-.1) > 1e-9 {
				t.Fatal("natural interval expired or hidden", e)
			}
			if e.ExtraRuns != 0 {
				t.Fatal("ordinary history charged additional tests")
			}
		})
	}
	s, _, _ := evaluationFixture(t, epoch, 30, 2, 60)
	s.state.Plan = nil
	runs := addOrdinaryObservation(t, s, "before", epoch.Add(-24*time.Hour), 100, 0)
	runs = append(runs, addOrdinaryObservation(t, s, "after", epoch, 110, 1)...)
	s.syncSessionContextsLocked(runs, epoch.Add(5*time.Minute))
	s.updateAnchorEvaluationsLocked(runs, epoch.Add(5*time.Minute))
	if s.state.AnchorEvaluations[1].Previous != nil {
		t.Fatal("different run position claimed comparable change")
	}
}

func TestExposureIncludesInvalidScoresAndGlobalSessions(t *testing.T) {
	s, _, _ := evaluationFixture(t, epoch, 30, 2, 60)
	s.state.Plan = nil
	s.SetSessionGap(40 * time.Minute)
	runs := addOrdinaryObservation(t, s, "before", epoch.Add(-24*time.Hour), 100, 0)
	runs = append(runs, addOrdinaryObservation(t, s, "after", epoch, 110, 0)...)
	paused := record("paused", "anchor", 60, 0, epoch.Add(-time.Hour))
	paused.Stats.Summary.PauseCount = 1
	other := record("other", "other", 60, 10, epoch.Add(-30*time.Minute))
	runs = append(runs, paused, other)
	positions := sessionPositions(s.state.Catalog, runs, 40*time.Minute, epoch.Add(5*time.Minute))
	if positions[runKey(paused)].SessionID != positions[runKey(other)].SessionID || positions[runKey(other)].PriorSeconds != 60 || positions[runKey(paused)].ScoreValid {
		t.Fatal("global session/exposure/score validity mixed")
	}
	// Use the default gap for the fixed observations so their position matches.
	s.SetSessionGap(20 * time.Minute)
	s.syncSessionContextsLocked(runs, epoch.Add(5*time.Minute))
	s.updateAnchorEvaluationsLocked(runs, epoch.Add(5*time.Minute))
	e := s.state.AnchorEvaluations[1]
	if e.Previous == nil || e.Exposure.SameSceneSeconds != 60 || e.Exposure.RecordedSeconds != 120 {
		t.Fatal("invalid score removed practice exposure", e)
	}
}

func TestPendingLegacyStudiesDoNotBlockDailyProbesOrMixProtocols(t *testing.T) {
	a := personalScene(t, "anchor", 10, "", epoch)
	b := personalScene(t, "variant", 8, "", epoch)
	c := Curriculum{Theme: "static", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	for _, status := range []string{"waiting", "due", "transfer_due"} {
		st := TrainingStudy{Theme: "static", Status: status, DueAt: epoch.Add(-time.Hour).UnixMilli(), ExpiresAt: epoch.Add(time.Hour).UnixMilli()}
		bundle := preparePersonalization(c, []Scenario{a, b}, anchorRuns(a.Name, epoch), nil, []TrainingStudy{st}, epoch, defaults(), true, 60, rand.New(rand.NewSource(1)))
		if bundle.study == nil || len(bundle.before) > 0 || bundle.study.ProtocolID != dailyTrialProtocol {
			t.Fatal("pending legacy study blocked daily probe", status)
		}
	}
	studies := []TrainingStudy{}
	for i := 0; i < 6; i++ {
		studies = append(studies, TrainingStudy{ID: fmt.Sprint(i), ProtocolID: dailyTrialProtocol, Relation: PrecisionRelation{FamilyFingerprint: "family", Uniform: true}, Baseline: &MeasurementResult{Score: 100}, Trial: &MeasurementResult{Score: 80}})
	}
	if got := responsePrediction(studies, PersonalAnchor{MedianScore: 100, FileSHA256: "file"}, &PrecisionRelation{Uniform: true, FamilyFingerprint: "family"}); got.Status != "insufficient" || got.Samples != 0 {
		t.Fatal("daily/legacy protocols mixed", got)
	}
}

func TestLegacyPlansRemainImmutableAtStartup(t *testing.T) {
	for _, status := range []string{"draft", "paused"} {
		s, _, _ := evaluationFixture(t, epoch, 30, 5, 60)
		s.state.Plan.PlannerVersion = 7
		s.state.Plan.Status = status
		before, _ := json.Marshal(s.state.Plan)
		changed, err := s.PrepareStartup(nil)
		after, _ := json.Marshal(s.state.Plan)
		if err != nil || changed || string(before) != string(after) {
			t.Fatal("v1 plan rewritten at startup", status, err)
		}
	}
}

func TestDailyTrialPersistsWithoutExpirationOrAttribution(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	a := personalScene(t, "anchor", 10, "", now)
	b := personalScene(t, "variant", 8, "", now)
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.state.Catalog = []Scenario{a, b}
	c := Curriculum{ID: "pilot", ContentSHA256: "hash", Theme: "static", Tier: "novice", Rows: []CurriculumRow{{Name: a.Name, Count: 2}}}
	s.state.Curricula = []Curriculum{c}
	s.state.CurriculumProgress = map[string]RoutineProgress{progressKey(c): {Cycle: 1, EverCompleted: true, Rows: []RowProgress{{Target: 2, Completed: 2}}}}
	runs := anchorRuns(a.Name, now)
	plan, err := s.Generate(defaults(), runs)
	if err != nil || len(plan.Blocks) != 2 || plan.Blocks[0].Role != "practice" || plan.Blocks[1].Role != "explore" {
		t.Fatal("daily trial integration", err)
	}
	_, _ = s.Action("start", now, runs)
	for i, name := range []string{a.Name, a.Name, b.Name, b.Name} {
		at := now.Add(time.Duration(i+1) * time.Minute)
		r := record(fmt.Sprint(i), name, 60, 80, at)
		r.Stats.Summary.GameVersion = "fixture"
		runs = append(runs, r)
		s.Tick(at, runs)
	}
	s.updateStudiesLocked(now.Add(6 * 24 * time.Hour))
	st := s.state.TrainingStudies[0]
	if st.Status != "observed" || st.Trial == nil || st.DueAt != 0 || st.ExpiresAt != 0 || st.RetentionChange != nil || st.TransferScenario != "" {
		t.Fatal("daily probe became a compulsory retest", st)
	}
	view, err := s.WorkbenchJSON(runs)
	if err != nil || strings.Contains(view, "runContexts") {
		t.Fatal("raw context leaked into workbench", err)
	}
}

func TestNeededAssessmentFitsFullRoutineWithoutReorderingOrOverbudget(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	s, c, runs := evaluationFixture(t, now, 30, 2, 60)
	s.state.Plan = nil
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("later-%d", i)
		s.state.Catalog = append(s.state.Catalog, Scenario{Name: name, Skill: "static", Technique: "static", Seconds: 60, Enabled: true})
		c.Rows = append(c.Rows, CurriculumRow{Name: name, Count: 2})
	}
	s.state.Curricula = []Curriculum{c}
	p := defaults()
	p.Variety = 0
	plan, err := s.Generate(p, runs)
	if err != nil {
		t.Fatal(err)
	}
	used, assessments := 0, 0
	for i, b := range plan.Blocks {
		used += b.Budget
		if b.Assessment != nil {
			assessments++
			if b.Assessment.ExtraRuns != 2 {
				t.Fatal("shared main counts changed", b)
			}
		}
		if b.Scenario.Name != c.Rows[i].Name {
			t.Fatal("main reordered to fit assessment")
		}
	}
	if assessments != 1 || used > 1620 || plan.CurriculumStart != 0 || plan.CurriculumEnd >= len(c.Rows) {
		t.Fatal("assessment starvation/budget/continuation", assessments, used, plan.CurriculumEnd)
	}
}

func TestCrossingMidnightPreservesShortIntervalWithoutSleepInference(t *testing.T) {
	zone := time.FixedZone("Asia/Shanghai", 8*3600)
	before := time.Date(2026, 9, 29, 23, 40, 0, 0, zone)
	after := before.Add(time.Hour)
	s, _, _ := evaluationFixture(t, after, 30, 2, 60)
	s.state.Plan = nil
	runs := addOrdinaryObservation(t, s, "before", before, 100, 0)
	runs = append(runs, addOrdinaryObservation(t, s, "after", after, 105, 0)...)
	now := after.Add(5 * time.Minute)
	s.syncSessionContextsLocked(runs, now)
	s.updateAnchorEvaluationsLocked(runs, now)
	e := s.state.AnchorEvaluations[1]
	if e.Previous == nil || e.IntervalHours != 1 || e.IntervalKind != "cross_day" || e.ComparableDays != 2 {
		t.Fatal("midnight changed actual interval", e)
	}
}

func TestSameDayRetrySessionsCannotJustifyNextTier(t *testing.T) {
	base, _, _ := progressionFixture()
	rows := levelObservations(sessionRuns(base.Name, []float64{40, 40, 40, 40, 40, 40}), epoch)[strings.ToLower(base.Name)]
	if readyForNextTier(base.Benchmarks[0], rows, "novice", epoch) {
		t.Fatal("same-day retry sessions granted advancement")
	}
}

func TestReparsedCSVDoesNotInheritAnOldExecutionBinding(t *testing.T) {
	s, _, runs := evaluationFixture(t, epoch, 30, 2, 60)
	r := runs[0]
	p := practiceSample(r)
	s.state.RunContexts[p.RunID] = RunContext{Scenario: r.Stats.Summary.Scenario, At: p.At, Signature: p.Signature, FileSHA256: "observed-file", ProtocolID: anchorObservationProtocol, PlanID: "old"}
	r.Stats.Summary.Hash = "changed-csv-version"
	s.syncSessionContextsLocked([]models.RunRecord{r}, epoch)
	if c := s.state.RunContexts[p.RunID]; c.FileSHA256 != "" || c.ProtocolID != "" || c.PlanID != "" {
		t.Fatal("derived context rewrote an execution fact", c)
	}
}

func TestPassiveHistoryTrimmingDoesNotResetAssessmentQuota(t *testing.T) {
	s, _, _ := evaluationFixture(t, epoch, 30, 2, 60)
	s.state.AnchorEvaluations = []AnchorEvaluation{{ID: "started-test", Theme: "static", ExtraRuns: 2, StartedAt: epoch.Add(-time.Hour).UnixMilli()}}
	for i := 0; i < 200; i++ {
		s.state.AnchorEvaluations = append(s.state.AnchorEvaluations, AnchorEvaluation{ID: fmt.Sprintf("passive-%d", i)})
	}
	s.updateAnchorEvaluationsLocked(nil, epoch)
	if len(s.state.AnchorEvaluations) > 128 || s.assessmentAllowed("static", epoch) {
		t.Fatal("trimming free observations reset paid assessment quota")
	}
}
