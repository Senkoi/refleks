package training

import (
	"math"
	"strings"
	"time"
)

var vdimThemes = []string{"static", "dynamic", "smooth", "reactive", "switching_speed", "switching_evasive"}

func scenarioTheme(s Scenario) string {
	if s.Technique != "" { return s.Technique }
	return technique(s.Name, s.Skill)
}

// Start from the author's six specialist categories. Recent completed practice
// automatically breaks the calendar rotation when a category is undertrained.
func chooseTheme(pool []Scenario, obs map[string][]observation, now time.Time) string {
	available := map[string]bool{}
	minutes := map[string]float64{}
	for _, s := range pool {
		key := scenarioTheme(s)
		available[key] = true
		for _, r := range obs[strings.ToLower(s.Name)] {
			if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -7)) && r.duration > 0 && r.duration <= 3600 {
				minutes[key] += r.duration / 60
			}
		}
	}
	// Time of generation uses the player's local time on the desktop.
	preferred := (int(now.Weekday()) + 6) % 7
	if preferred >= len(vdimThemes) { preferred = 0 }
	best, bestScore := "", math.Inf(1)
	for offset := range vdimThemes {
		key := vdimThemes[(preferred+offset)%len(vdimThemes)]
		if available[key] && minutes[key] < bestScore {
			best, bestScore = key, minutes[key]
		}
	}
	return best
}
