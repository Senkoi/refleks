package training

import (
	"math/rand"
	"sort"
	"strings"
	"time"

	"refleks/internal/models"
)

// Experimental ceilings, not claims about an optimal scientific ratio. Keep
// whole runs, current audience foundations and limited unassessed trials.
type ProgressionBudget struct {
	ChallengeLimit   int `json:"challengeLimit"`
	ExplorationLimit int `json:"explorationLimit"`
	UnknownLimit     int `json:"unknownLimit"`
}

// Compare three recent practice sessions with three previous sessions. Runs
// less than 30 minutes apart belong to one session, so a single bad run (or
// three consecutive retries) cannot cause a progression rollback.
func performanceTrend(rows []observation, now time.Time) (string, int) {
	valid := []observation{}
	for _, r := range rows {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -45)) {
			valid = append(valid, r)
		}
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].at.After(valid[j].at) })
	valid = comparable(valid)
	type session struct{ scores, accuracy, speed []float64 }
	sessions := []session{}
	for i, r := range valid {
		if i == 0 || valid[i-1].at.Sub(r.at) >= 30*time.Minute {
			if len(sessions) == 6 {
				break
			}
			sessions = append(sessions, session{})
		}
		s := &sessions[len(sessions)-1]
		s.scores = append(s.scores, r.score)
		if r.accuracyKnown {
			s.accuracy = append(s.accuracy, r.accuracy)
		}
		if r.speedKnown {
			s.speed = append(s.speed, r.hitsPerSecond)
		}
	}
	if len(sessions) < 6 {
		return "insufficient", len(sessions)
	}
	newScores, oldScores, newAccuracy, oldAccuracy := []float64{}, []float64{}, []float64{}, []float64{}
	newSpeed, oldSpeed := []float64{}, []float64{}
	for i, s := range sessions {
		if i < 3 {
			newScores = append(newScores, median(s.scores))
			if len(s.accuracy) > 0 {
				newAccuracy = append(newAccuracy, median(s.accuracy))
			}
			if len(s.speed) > 0 {
				newSpeed = append(newSpeed, median(s.speed))
			}
		} else {
			oldScores = append(oldScores, median(s.scores))
			if len(s.accuracy) > 0 {
				oldAccuracy = append(oldAccuracy, median(s.accuracy))
			}
			if len(s.speed) > 0 {
				oldSpeed = append(oldSpeed, median(s.speed))
			}
		}
	}
	previous := median(oldScores)
	declining := previous > 0
	for _, v := range newScores {
		declining = declining && v < previous*.9
	}
	if declining {
		return "declining", 6
	}
	if len(newAccuracy) == 3 && len(oldAccuracy) == 3 {
		previousAccuracy := median(oldAccuracy)
		declining = previousAccuracy > 0
		for _, v := range newAccuracy {
			declining = declining && v < previousAccuracy*.9
		}
		faster := len(newSpeed) == 3 && len(oldSpeed) == 3 && median(newSpeed) >= median(oldSpeed)*1.05 && median(newScores) >= previous
		if declining && !faster {
			return "declining", 6
		}
	}
	if previous > 0 && median(newScores) >= previous*1.05 {
		return "improving", 6
	}
	return "stable", 6
}

func nativeForTier(tier string) string {
	switch tier {
	case "adept", "intermediate":
		return "Intermediate"
	case "advanced", "elite":
		return "Advanced"
	}
	return "Novice"
}

// A local near-threshold result can justify a training trial without granting
// an official category rank. Undated PBs never establish stable readiness.
func readyForNextTier(m BenchmarkMembership, rows []observation, tier string, now time.Time) bool {
	if practiceDayCount(rows, now) < 3 {
		return false
	}
	score, n, _ := levelScoreWindow(rows, now)
	if n < 3 {
		return false
	}
	trend, sessions := performanceTrend(rows, now)
	if sessions < 3 || trend == "declining" {
		return false
	}
	series, _ := benchmarkSeries(m.System)
	if series != "voltaic" || !validRankCutoffs(m) {
		return false
	}
	target := m.Thresholds[len(m.Thresholds)-1]
	if tierIndex(vdimRankTier(m.Ranks[len(m.Ranks)-1])) < tierIndex(tier) {
		return false
	}
	for i, rank := range m.Ranks {
		if tierIndex(vdimRankTier(rank)) > tierIndex(tier) {
			target = m.Thresholds[i]
			break
		}
	}
	stable := true
	recent := []observation{}
	for _, r := range rows {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -45)) {
			recent = append(recent, r)
		}
	}
	sort.Slice(recent, func(i, j int) bool { return recent[i].at.After(recent[j].at) })
	recent = comparable(recent)
	recentSessions := [][]float64{}
	for i, r := range recent {
		if i == 0 || recent[i-1].at.Sub(r.at) >= 30*time.Minute {
			if len(recentSessions) == 3 {
				break
			}
			recentSessions = append(recentSessions, []float64{})
		}
		recentSessions[len(recentSessions)-1] = append(recentSessions[len(recentSessions)-1], r.score)
	}
	for _, scores := range recentSessions {
		stable = stable && median(scores) >= target*.9
	}
	return stable || trend == "improving" && score >= target*.85
}

type progressionCandidate struct {
	scenario  Scenario
	anchor    Scenario
	fit       DifficultyEvidence
	reference BenchmarkMembership
	kind      string
	seconds   int
	weight    float64
}

func progressionCandidates(t Curriculum, catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time, templates []Curriculum) []progressionCandidate {
	levels := PlayerLevels(catalog, runs, now)
	obs, fitObs := observed(runs), levelObservations(runs, now)
	p = automaticReferences(catalog, p)
	tier := trainingTier(t.Theme, levels)
	adjacent := map[string]bool{}
	audience := map[string]int{}
	for _, v := range templates {
		if v.Theme == t.Theme && v.OfficialCode != "" && tierIndex(v.Tier) >= 0 {
			for _, row := range v.Rows {
				name := strings.ToLower(row.Name)
				old, ok := audience[name]
				if !ok || tierIndex(v.Tier) < old {
					audience[name] = tierIndex(v.Tier)
				}
			}
		}
		if v.Theme == t.Theme && v.OfficialCode != "" && tierIndex(v.Tier) == tierIndex(tier)+1 {
			for _, row := range v.Rows {
				adjacent[strings.ToLower(row.Name)] = true
			}
		}
	}
	byName := map[string]Scenario{}
	baseNames := map[string]bool{}
	for _, s := range catalog {
		byName[strings.ToLower(s.Name)] = enrichMechanics(s)
	}
	for _, r := range t.Rows {
		baseNames[strings.ToLower(r.Name)] = true
	}
	for _, full := range templates {
		if t.ID != "" && full.ID == t.ID {
			for _, row := range full.Rows {
				baseNames[strings.ToLower(row.Name)] = true
			}
		}
	}
	result := []progressionCandidate{}
	exposure := map[string]float64{}
	for _, s := range catalog {
		addDemandTime(s, estimateTiming(s, obs[strings.ToLower(s.Name)], now).WeeklySeconds, exposure)
	}
	groupTiers := trainingGroupTiers(t.Theme, levels)
	for _, row := range t.Rows {
		if row.Role == "warmup" || row.Role == "benchmark" {
			continue
		}
		anchor, ok := byName[strings.ToLower(row.Name)]
		if !ok {
			continue
		}
		readyGroups := map[string]bool{}
		templateReady := false
		for _, source := range catalog {
			if !strings.EqualFold(source.Name, anchor.Name) && !goalCompatible(anchor, enrichMechanics(source)) {
				continue
			}
			for _, m := range memberships(source) {
				key := strings.ToLower(m.Category + "|" + m.Group)
				goalTier := groupTiers[key]
				if goalTier == "" {
					goalTier = tier
				}
				if readyForNextTier(m, fitObs[strings.ToLower(source.Name)], goalTier, now) {
					readyGroups[key] = true
				}
				if readyForNextTier(m, fitObs[strings.ToLower(source.Name)], tier, now) {
					templateReady = true
				}
			}
		}
		for _, raw := range catalog {
			c := enrichMechanics(raw)
			if precisionRelation(anchor, c) != nil {
				continue
			} // Dedicated fixed-protocol personal trials own controlled variants.
			if lowest, ok := audience[strings.ToLower(c.Name)]; ok && lowest > tierIndex(tier)+1 {
				continue
			}
			if baseNames[strings.ToLower(c.Name)] || !goalCompatible(anchor, c) || c.Preference == "disliked" {
				continue
			}
			fit := assessDifficultyFor(c, fitObs[strings.ToLower(c.Name)], now, p)
			if fit.Trend == "declining" {
				continue
			}
			ref, aligned := explorationReference(c, levels, p)
			nextRoster := false
			for _, m := range memberships(c) {
				series, _ := benchmarkSeries(m.System)
				key := strings.ToLower(m.Category + "|" + m.Group)
				if series != "voltaic" || !selectedBenchmark(p, m.Name) || !readyGroups[key] {
					continue
				}
				goalTier := groupTiers[key]
				if goalTier == "" {
					goalTier = tier
				}
				i := tierIndex(goalTier)
				if i >= 0 && i+1 < len(curriculumTiers) && nativeForTier(curriculumTiers[i+1]) != nativeForTier(goalTier) && strings.EqualFold(m.NativeDifficulty, nativeForTier(curriculumTiers[i+1])) {
					nextRoster = true
					ref = m
				}
			}
			trainable := fit.Fit == "challenging" && fit.Trend == "improving" && practiceDayCount(fitObs[strings.ToLower(c.Name)], now) >= 3
			aboveFloor := fit.Fit != "challenging"
			for _, m := range fit.Benchmarks {
				if len(m.Thresholds) > 0 && fit.RecentScore >= m.Thresholds[0]*.4 {
					aboveFloor = true
				}
			}
			kind := "explore"
			if aboveFloor && (trainable || nextRoster || templateReady && adjacent[strings.ToLower(c.Name)]) {
				kind = "challenge"
			}
			if fit.Fit == "challenging" && kind != "challenge" {
				continue
			}
			if len(memberships(c)) > 0 && !aligned && !(kind == "challenge" && (nextRoster || templateReady && adjacent[strings.ToLower(c.Name)])) {
				continue
			}
			result = append(result, progressionCandidate{scenario: c, anchor: anchor, fit: fit, reference: ref, kind: kind, seconds: estimateTiming(c, obs[strings.ToLower(c.Name)], now).Seconds, weight: demandWeight(c, exposure) / (1 + float64(len(obs[strings.ToLower(c.Name)])))})
		}
	}

	return result
}

func selectProgression(candidates []progressionCandidate, usable, available int, p Preferences, rng *rand.Rand) ([]progressionCandidate, ProgressionBudget) {
	limits := ProgressionBudget{ChallengeLimit: usable * 3 / 10, ExplorationLimit: int(float64(usable) * min(p.Variety, .1)), UnknownLimit: usable / 10}
	selected := []progressionCandidate{}
	used := map[string]bool{}
	unknown := 0
	for _, kind := range []string{"challenge", "explore"} {
		budget := limits.ChallengeLimit
		if kind == "explore" {
			budget = limits.ExplorationLimit
		}
		for budget > 0 && available > 0 {
			best := -1
			weight := -1.0
			for i, c := range candidates {
				if c.kind != kind || used[strings.ToLower(c.scenario.Name)] || c.seconds > budget || c.seconds > available || c.fit.Fit == "unknown" && unknown+c.seconds > limits.UnknownLimit {
					continue
				}
				w := c.weight + rng.Float64()*.01
				if c.fit.Fit == "suitable" {
					w += 10
				}
				if c.fit.Trend == "improving" {
					w += 5
				}
				if c.fit.Fit == "comfortable" {
					w *= .25
				}
				if w > weight {
					best, weight = i, w
				}
			}
			if best < 0 {
				break
			}
			c := candidates[best]
			selected = append(selected, c)
			used[strings.ToLower(c.scenario.Name)] = true
			budget -= c.seconds
			available -= c.seconds
			if c.fit.Fit == "unknown" {
				unknown += c.seconds
			}
		}
	}
	return selected, limits
}
