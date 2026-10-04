package training

import "strings"

// Native difficulty is a roster reference. Only Voltaic's documented VDIM
// audience mapping is used; unrelated systems require their own score evidence.
func explorationReference(s Scenario, levels []PlayerLevel, p Preferences) (BenchmarkMembership, bool) {
	theme := scenarioTheme(s)
	for _, m := range memberships(s) {
		if !selectedBenchmark(p, m.Name) || len(m.Thresholds) == 0 {
			continue
		}
		series, _ := benchmarkSeries(m.System)
		if series == "voltaic" {
			// Prefer this native goal's evidence, retaining useful older versions.
			// Do not transfer another group's weakness (or strength) to this one.
			goalLevels := []PlayerLevel{}
			for _, l := range levels {
				if l.Theme == theme && strings.EqualFold(l.Category, m.Category) && strings.EqualFold(l.Group, m.Group) {
					goalLevels = append(goalLevels, l)
				}
			}
			tier := trainingTier(theme, goalLevels)
			if len(trainingGroupTiers(theme, goalLevels)) == 0 {
				tier = trainingTier(theme, levels)
			}
			native := "Novice"
			if tier == "adept" || tier == "intermediate" {
				native = "Intermediate"
			}
			if tier == "advanced" || tier == "elite" {
				native = "Advanced"
			}
			if strings.EqualFold(m.NativeDifficulty, native) {
				return m, true
			}
			continue // An older high PB cannot bypass the current goal's tier.
		}
		for _, l := range levels {
			if (l.Status == "inferred" || l.Status == "estimated") && l.Theme == theme && l.System == m.System && l.NativeDifficulty == m.NativeDifficulty && l.Category == m.Category && l.Group == m.Group {
				return m, true
			}
		}
	}
	return BenchmarkMembership{}, false
}
