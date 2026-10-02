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
			tier := inferredTier(theme, levels)
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
		}
		for _, l := range levels {
			if (l.Status == "inferred" || l.Status == "estimated") && l.Theme == theme && l.System == m.System && l.NativeDifficulty == m.NativeDifficulty && l.Category == m.Category && l.Group == m.Group {
				return m, true
			}
		}
	}
	return BenchmarkMembership{}, false
}
