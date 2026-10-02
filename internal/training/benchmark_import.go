package training

import (
	"refleks/internal/models"
	"strings"
	"time"
)

// BenchmarkScenarios joins the existing RefleK's definitions and progress by
// native ID. Keep system/version names and every native tier verbatim.
// Rosters supply reference metadata, never playlist ordering or repeat counts.
func BenchmarkScenarios(catalog []models.Benchmark, progress map[int]models.BenchmarkProgress, now time.Time) []Scenario {
	items := []Scenario{}
	for _, b := range catalog {
		for _, d := range b.Difficulties {
			p, ok := progress[d.KovaaksBenchmarkID]
			if !ok {
				continue
			}
			for _, c := range p.Categories {
				for _, g := range c.Groups {
					for _, s := range g.Scenarios {
						skill := "unknown"
						hint := strings.ToLower(c.Name + " " + g.Name)
						switch {
						case strings.Contains(hint, "switch"):
							skill = "switching"
						case strings.Contains(hint, "static"):
							skill = "static"
						case strings.Contains(hint, "dynamic"), strings.Contains(hint, "linear"), strings.Contains(hint, "timing"):
							skill = "dynamic"
						case strings.Contains(hint, "reactiv"):
							skill = "reactive"
						case strings.Contains(hint, "smooth"), strings.Contains(hint, "precis"), strings.Contains(hint, "control"):
							skill = "smooth"
						}
						m := BenchmarkMembership{Name: b.BenchmarkName + " / " + d.DifficultyName, BenchmarkID: d.KovaaksBenchmarkID, System: b.BenchmarkName, NativeDifficulty: d.DifficultyName, Category: c.Name, Group: g.Name, Thresholds: append([]float64(nil), s.Thresholds...)}
						items = append(items, Scenario{Name: s.Name, Skill: skill, Technique: BenchmarkTechnique(c.Name, g.Name, skill), Family: strings.ToLower(s.Name), Difficulty: "unknown", DifficultySource: "unknown", Seconds: 60, Benchmarks: []BenchmarkMembership{m}, Classification: "benchmark", Enabled: skill != "unknown", Sources: []Source{{URL: b.SpreadsheetURL, Title: m.Name + " / " + g.Name, Retrieved: now.UTC().Format(time.RFC3339)}}})
					}
				}
			}
		}
	}
	return items
}
