package training

import (
	"reflect"
	"strings"
	"time"
)

func difficultyRank(source string) int {
	switch source {
	case "manual":
		return 5
	case "benchmark":
		return 4
	case "name":
		return 2
	case "playlist":
		return 1
	default:
		return 0
	}
}

// A score from this same scenario can assess personal fit against its published
// cutoffs. It cannot establish how hard a different community scenario is.
func assessDifficulty(s Scenario, rows []observation, now time.Time) DifficultyEvidence {
	return assessDifficultyFor(s, rows, now, Preferences{})
}

func assessDifficultyFor(s Scenario, rows []observation, now time.Time, p Preferences) DifficultyEvidence {
	level, source := s.Difficulty, s.DifficultySource
	if level == "" {
		level = "unknown"
	}
	if source == "" {
		if s.Classification == "benchmark" {
			source = "benchmark"
		} else if s.Classification == "manual" {
			source = "manual"
		} else if level != "unknown" {
			source = "name"
		} else {
			source = "unknown"
		}
	}
	eligible := []BenchmarkMembership{}
	for _, m := range memberships(s) {
		if p.Benchmark == "" && len(p.Benchmarks) == 0 || selectedBenchmark(p, m.Name) {
			eligible = append(eligible, m)
		}
	}
	e := DifficultyEvidence{Level: level, Source: source, Fit: "unknown", Benchmarks: eligible}
	// Native tiers describe a benchmark roster, not calibrated mechanism difficulty.
	if source == "benchmark" {
		for _, m := range memberships(s) {
			if m.BenchmarkID != 0 {
				e.Level, e.Source = "unknown", "unknown"
				break
			}
		}
	}
	// A score must not silently choose among incompatible versions/tier cutoffs.
	thresholds := []float64(nil)
	ambiguous := false
	for _, m := range eligible {
		if len(m.Thresholds) == 0 {
			continue
		}
		if thresholds == nil {
			thresholds = m.Thresholds
		} else if !reflect.DeepEqual(thresholds, m.Thresholds) {
			ambiguous = true
		}
	}
	valid := []observation{}
	for _, r := range rows {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -45)) {
			valid = append(valid, r)
		}
	}
	valid = comparable(valid)
	score, samples, days := levelScoreWindow(valid, now)
	e.Samples = samples
	e.WindowDays = days
	e.RecentScore = score
	e.Trend, e.TrendSessions = performanceTrend(valid, now)
	if samples >= 3 && !ambiguous {
		// Published thresholds are specific to a benchmark scenario and version.
		for _, m := range eligible {
			if len(m.Thresholds) == 0 {
				continue
			}
			first, last := m.Thresholds[0], m.Thresholds[0]
			for _, v := range m.Thresholds {
				if v < first {
					first = v
				}
				if v > last {
					last = v
				}
			}
			if first <= 0 {
				continue
			}
			switch {
			case score < first*.75:
				e.Fit = "challenging"
			case score >= last:
				e.Fit = "comfortable"
			default:
				e.Fit = "suitable"
			}
			break
		}
	}
	if s.PersonalDifficulty == "hard" {
		e.Fit = "challenging"
	}
	if s.PersonalDifficulty == "suitable" {
		e.Fit = "suitable"
	}
	if s.PersonalDifficulty == "easy" {
		e.Fit = "comfortable"
	}
	return e
}

// Official category/group metadata is stronger than a scenario-name guess.
func BenchmarkTechnique(category, group, skill string) string {
	hint := strings.ToLower(category + " " + group)
	if skill == "switching" {
		if strings.Contains(hint, "evas") || strings.Contains(hint, "stability") {
			return "switching_evasive"
		}
		return "switching_speed"
	}
	return skill
}
