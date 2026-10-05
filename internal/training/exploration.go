package training

import (
	"math/rand"
	"sort"
	"strings"
	"time"

	"aimmeow/internal/models"
)

type ExplorationReason struct {
	Code   string   `json:"code"`
	Count  int      `json:"count"`
	Scenes []string `json:"scenes"`
}
type ExplorationReport struct {
	Status         string              `json:"status"`
	Eligible       int                 `json:"eligible"`
	Selected       []string            `json:"selected"`
	LimitSeconds   int                 `json:"limitSeconds"`
	UsedSeconds    int                 `json:"usedSeconds"`
	UnknownSeconds int                 `json:"unknownSeconds"`
	Reasons        []ExplorationReason `json:"reasons"`
	rejected       map[string]map[string]bool
}

func newExplorationReport() *ExplorationReport {
	return &ExplorationReport{Status: "no_candidates", Selected: []string{}, Reasons: []ExplorationReason{}, rejected: map[string]map[string]bool{}}
}
func (r *ExplorationReport) reject(name, reason string) {
	if r == nil {
		return
	}
	if r.rejected[reason] == nil {
		r.rejected[reason] = map[string]bool{}
	}
	r.rejected[reason][name] = true
}
func (r *ExplorationReport) finish(eligible []progressionCandidate, selected []progressionCandidate) {
	r.Eligible = len(eligible)
	accepted := map[string]bool{}
	for _, c := range eligible {
		accepted[c.scenario.Name] = true
	}
	for _, c := range selected {
		r.Selected = append(r.Selected, c.scenario.Name)
		if c.kind == "explore" {
			r.UsedSeconds += c.seconds
		}
		if c.fit.Fit == "unknown" {
			r.UnknownSeconds += c.seconds
		}
	}
	if len(selected) > 0 {
		r.Status = "selected"
	} else if len(eligible) > 0 || len(r.rejected["whole_run_budget"]) > 0 {
		r.Status = "budget_limited"
	}
	codes := []string{}
	for code := range r.rejected {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		names := []string{}
		for name := range r.rejected[code] {
			if accepted[name] && code != "whole_run_budget" && code != "budget_or_priority" {
				continue
			}
			names = append(names, name)
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		r.Reasons = append(r.Reasons, ExplorationReason{Code: code, Count: len(names), Scenes: names[:min(3, len(names))]})
	}
	r.rejected = nil
}

// Familiarity is local to the actual task, not completion of a whole routine.
// Three completed comparable runs permit a bounded trial; they do not grant rank.
func familiarPractice(rows []observation, now time.Time) bool {
	recent := []observation{}
	for _, r := range rows {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -45)) {
			recent = append(recent, r)
		}
	}
	return len(comparable(recent)) >= 3
}

func familiarityObservations(runs []models.RunRecord, now time.Time) map[string][]observation {
	valid := []models.RunRecord{}
	for _, r := range runs {
		if validPractice(r, now) {
			valid = append(valid, r)
		}
	}
	return observed(valid)
}

// A trusted category can support an unknown-difficulty probe before its first
// SCE arrives. Name inference alone cannot establish a matching training goal.
func explorationGoalCompatible(anchor, candidate Scenario) bool {
	if goalCompatible(anchor, candidate) {
		return true
	}
	if !candidate.Enabled || anchor.Skill != candidate.Skill || scenarioTheme(anchor) != scenarioTheme(candidate) || candidate.Mechanics != nil {
		return false
	}
	if candidate.LocalAssessment != nil && candidate.LocalAssessment.Status != "file_parsed_model_unfitted" {
		return false
	}
	if candidate.Classification == "manual" && candidate.Technique != "" {
		return true
	}
	if candidate.Classification == "benchmark" {
		for _, m := range memberships(candidate) {
			if m.System != "" && (m.Category != "" || m.Group != "") {
				return true
			}
		}
	}
	return false
}

// The same allocator owns ordinary variants and controlled trials. It reserves
// whole runs before the ordered main window, leaving at least one main run.
func selectTrainingExtras(t Curriculum, catalog []Scenario, runs []models.RunRecord, contexts map[string]RunContext, studies []TrainingStudy, p Preferences, now time.Time, baseline bool, firstSeconds int, rng *rand.Rand, templates []Curriculum) (personalizationBundle, *ExplorationReport, *ProgressionBudget) {
	if p.ExplorationMode == "off" {
		p.Variety = 0
	}
	report := newExplorationReport()
	usable := p.Minutes * 60 * 9 / 10
	report.LimitSeconds = int(float64(usable) * min(p.Variety, .1))
	candidates := progressionCandidatesWithReport(t, catalog, runs, p, now, templates, baseline, report)
	personal := preparePersonalization(t, catalog, runs, contexts, studies, now, p, baseline, firstSeconds, rng, report)
	if personal.study != nil && len(personal.after) > 0 {
		b := personal.after[0]
		fit := assessDifficultyFor(b.Scenario, levelObservations(runs, now)[strings.ToLower(b.Scenario.Name)], now, p)
		candidates = append(candidates, progressionCandidate{scenario: b.Scenario, kind: "explore", seconds: personal.seconds(), weight: 2, fit: fit, personal: &personal})
	}
	candidates = deduplicateProgression(candidates)
	chosen, limits := selectProgression(candidates, usable, max(0, usable-firstSeconds), p, rng)
	selected := map[string]bool{}
	result := personalizationBundle{}
	for _, c := range chosen {
		selected[c.scenario.Name] = true
		if c.personal != nil {
			result.before = append(result.before, c.personal.before...)
			result.after = append(result.after, c.personal.after...)
			result.study = c.personal.study
		} else {
			result.after = append(result.after, progressionBlock(c, catalog, runs, p, now))
		}
	}
	for _, c := range candidates {
		if selected[c.scenario.Name] {
			continue
		}
		budget := limits.ExplorationLimit
		if c.kind == "challenge" {
			budget = limits.ChallengeLimit
		}
		if c.seconds > budget || c.seconds > max(0, usable-firstSeconds) {
			report.reject(c.scenario.Name, "whole_run_budget")
		} else {
			report.reject(c.scenario.Name, "budget_or_priority")
		}
	}
	report.finish(candidates, chosen)
	if p.Variety <= 0 || p.ExplorationMode == "off" {
		report.Status = "disabled"
	}
	return result, report, &limits
}

func deduplicateProgression(candidates []progressionCandidate) []progressionCandidate {
	index := map[string]int{}
	out := []progressionCandidate{}
	for _, c := range candidates {
		key := strings.ToLower(c.scenario.Name)
		if i, ok := index[key]; ok {
			if c.personal != nil || c.weight > out[i].weight {
				out[i] = c
			}
			continue
		}
		index[key] = len(out)
		out = append(out, c)
	}
	return out
}

func progressionBlock(c progressionCandidate, catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time) Block {
	obs := observed(runs)
	timing := estimateTiming(c.scenario, obs[strings.ToLower(c.scenario.Name)], now)
	reason := "匹配已练习的训练目标，以整局预算尝试另一种练习；难度未知时先小量试练。"
	cue := "完成一次，结果只影响下一份列表。"
	if c.kind == "challenge" {
		reason = "依据近期稳定表现或持续改善，试探同目标的相邻档位；不提升正式 benchmark 等级。"
		cue = "保持控制质量，单次低分不触发回退。"
	}
	var decision *SceneDecision
	if c.neighbor != nil {
		personal := personalAnchors(catalog, runs, nil, now)[strings.ToLower(c.anchor.Name)]
		decision = &SceneDecision{Source: "requirement_neighbor", Anchor: personal, Requirements: c.neighbor}
	}
	return Block{Scenario: c.scenario, Timing: timing, DifficultyEvidence: c.fit, AnchorScenario: c.anchor.Name, Role: c.kind, Benchmark: c.reference.Name, Budget: c.seconds, PlayCount: 1, Outcome: "pending", Reason: reason, Cue: cue, Personalization: decision}
}
