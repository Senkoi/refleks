package training

import (
	"strings"
	"time"
)

func difficultyRank(source string) int {
	switch source {
	case "manual": return 5
	case "benchmark": return 4
	case "name": return 2
	case "playlist": return 1
	default: return 0
	}
}

// A score from this same scenario can assess personal fit against its published
// cutoffs. It cannot establish how hard a different community scenario is.
func assessDifficulty(s Scenario, rows []observation, now time.Time) DifficultyEvidence {
	level, source := s.Difficulty, s.DifficultySource
	if level == "" { level = "unknown" }
	if source == "" {
		if s.Classification == "benchmark" { source = "benchmark" } else if s.Classification == "manual" { source = "manual" } else if level != "unknown" { source = "name" } else { source = "unknown" }
	}
	e := DifficultyEvidence{Level: level, Source: source, Fit: "unknown"}
	valid := []observation{}
	for _, r := range rows {
		if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -30)) { valid = append(valid, r) }
	}
	valid = comparable(valid)
	e.Samples = len(valid)
	if len(valid) >= 3 {
		// Published thresholds are specific to a benchmark scenario and version.
		for _, m := range memberships(s) {
			if len(m.Thresholds) == 0 { continue }
			first, last := m.Thresholds[0], m.Thresholds[0]
			for _, v := range m.Thresholds { if v < first { first = v }; if v > last { last = v } }
			if first <= 0 { continue }
			scores := make([]float64, len(valid))
			for i, r := range valid { scores[i] = r.score }
			score := median(scores)
			switch {
			case score < first * .75: e.Fit = "challenging"
			case score >= last: e.Fit = "comfortable"
			default: e.Fit = "suitable"
			}
			break
		}
	}
	if s.PersonalDifficulty == "hard" { e.Fit = "challenging" }
	if s.PersonalDifficulty == "suitable" { e.Fit = "suitable" }
	if s.PersonalDifficulty == "easy" { e.Fit = "comfortable" }
	return e
}

// Official category/group metadata is stronger than a scenario-name guess.
func BenchmarkTechnique(category, group, skill string) string {
	hint := strings.ToLower(category + " " + group)
	if skill == "switching" {
		if strings.Contains(hint, "evas") || strings.Contains(hint, "stability") { return "switching_evasive" }
		return "switching_speed"
	}
	return skill
}
