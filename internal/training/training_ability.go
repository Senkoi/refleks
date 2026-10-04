package training

import (
	"fmt"
	"math"
	"refleks/internal/models"
	"strings"
	"time"
)

var voltaicRanks = []string{"Iron", "Bronze", "Silver", "Gold", "Platinum", "Diamond", "Jade", "Master", "Grandmaster", "Nova", "Astra", "Celestial"}

func rankValue(rank string) (float64, bool) {
	for i, r := range voltaicRanks {
		if strings.EqualFold(rank, r) {
			return float64(i), true
		}
	}
	return 0, false
}

// Interpolate against this scene's own rank thresholds, then compare on the
// same system's rank axis. Raw scores and different benchmark systems never mix.
func scoreValue(m BenchmarkMembership, score float64) (float64, bool) {
	series, _ := benchmarkSeries(m.System)
	if series != "voltaic" || score <= 0 || !validRankCutoffs(m) {
		return 0, false
	}
	first, ok := rankValue(m.Ranks[0])
	if !ok {
		return 0, false
	}
	if score < m.Thresholds[0] {
		return math.Max(0, first-1+score/m.Thresholds[0]), true
	}
	for i := 1; i < len(m.Thresholds); i++ {
		lo, ok1 := rankValue(m.Ranks[i-1])
		hi, ok2 := rankValue(m.Ranks[i])
		if !ok1 || !ok2 || hi <= lo {
			return 0, false
		}
		if score < m.Thresholds[i] {
			return lo + (hi-lo)*(score-m.Thresholds[i-1])/(m.Thresholds[i]-m.Thresholds[i-1]), true
		}
	}
	return rankValue(m.Ranks[len(m.Ranks)-1])
}
func valueTier(value float64) string {
	i := min(len(voltaicRanks)-1, max(0, int(math.Floor(value))))
	return vdimRankTier(voltaicRanks[i])
}

type ThemePriority struct {
	Priority float64  `json:"priority"`
	Minutes  float64  `json:"minutes"`
	Level    *float64 `json:"level,omitempty"`
	Evidence string   `json:"evidence"`
}

func ThemePriorities(catalog []Scenario, runs []models.RunRecord, now time.Time) map[string]ThemePriority {
	return themePrioritiesWithLevels(catalog, runs, now, PlayerLevels(catalog, runs, now))
}

func themePrioritiesWithLevels(catalog []Scenario, runs []models.RunRecord, now time.Time, levels []PlayerLevel) map[string]ThemePriority {
	byTheme := map[string][]float64{}
	confidence := map[string]float64{}
	// Use each subgroup once. The newest available evidence with local-history
	// precedence is already selected by trainingGroupLevels.
	for _, theme := range vdimThemes {
		for _, l := range trainingGroupLevels(theme, levels) {
			byTheme[theme] = append(byTheme[theme], l.Ability)
			c := float64(l.Scenarios) / float64(max(1, l.Required))
			if l.Source == "benchmark" {
				c *= .7
			}
			confidence[theme] += c
		}
	}
	medians := []float64{}
	for _, v := range byTheme {
		if len(v) > 0 {
			medians = append(medians, median(v))
		}
	}
	own := median(medians)
	sceneThemes := map[string]string{}
	for _, s := range catalog {
		sceneThemes[strings.ToLower(s.Name)] = scenarioTheme(s)
	}
	minutes := map[string]float64{}
	weighted := map[string]float64{}
	last := map[string]time.Time{}
	for name, rows := range observed(runs) {
		theme := sceneThemes[name]
		if theme == "" {
			continue
		}
		for _, r := range rows {
			age := now.Sub(r.at)
			if age < 0 || age > 28*24*time.Hour {
				continue
			}
			weighted[theme] += r.duration / 60 * math.Exp2(-age.Hours()/(7*24))
			if age <= 7*24*time.Hour {
				minutes[theme] += r.duration / 60
			}
			if r.at.After(last[theme]) {
				last[theme] = r.at
			}
		}
	}
	out := map[string]ThemePriority{}
	for _, theme := range vdimThemes {
		weakness := 0.0
		levelText := "有效等级证据不足，保留有限评估机会"
		var value *float64
		if v := byTheme[theme]; len(v) > 0 {
			x := median(v)
			value = &x
			weakness = min(2, math.Max(0, own-x)) * confidence[theme] / float64(len(v))
			levelText = fmt.Sprintf("同体系水平 %.2f，低于自身中位水平 %.2f 档", x, weakness)
		}
		gap := 7.0
		if !last[theme].IsZero() {
			gap = min(7, now.Sub(last[theme]).Hours()/24)
		}
		// Initial product weights: weakness and undertraining dominate; bounded
		// recency correction prevents starvation without forcing the weakest daily.
		priority := (1+weakness)/(1+weighted[theme]/30) + .3*gap/7
		if value == nil {
			priority = min(priority, .9)
		}
		out[theme] = ThemePriority{priority, minutes[theme], value, fmt.Sprintf("%s；近七天训练 %.1f 分钟，衰减训练量 %.1f 分钟，距上次训练 %.1f 天；综合优先级 %.2f。", levelText, minutes[theme], weighted[theme], gap, priority)}
	}
	return out
}

func trainingGroupLevels(theme string, levels []PlayerLevel) map[string]PlayerLevel {
	best := map[string]PlayerLevel{}
	for _, l := range levels {
		series, _ := benchmarkSeries(l.System)
		if l.Theme != theme || series != "voltaic" || l.TrainingTier == "" {
			continue
		}
		key := strings.ToLower(l.Category + "|" + l.Group)
		old, ok := best[key]
		recent := l.Source == "history" || l.Source == "mixed"
		oldRecent := old.Source == "history" || old.Source == "mixed"
		// A top rank in an easier roster establishes a lower bound; incomplete
		// coverage cannot establish a category ceiling.
		if !ok || recent && !oldRecent && !l.TrainingAtCeiling || recent == oldRecent && l.Ability > old.Ability || old.TrainingAtCeiling && l.Ability > old.Ability {
			best[key] = l
		}
	}
	return best
}

func trainingGroupTiers(theme string, levels []PlayerLevel) map[string]string {
	out := inferredGroupTiers(theme, levels)
	for key, l := range trainingGroupLevels(theme, levels) {
		out[key] = l.TrainingTier
	}
	return out
}
