package training

import (
	"math"
	"refleks/internal/models"
	"sort"
	"strings"
	"time"
)

type PlayerLevel struct {
	AtCeiling        bool   `json:"atCeiling,omitempty"`
	Source           string `json:"source,omitempty"`
	WindowDays       int    `json:"windowDays,omitempty"`
	LastPlayed       string `json:"lastPlayed,omitempty"`
	Theme            string `json:"theme"`
	Category         string `json:"category,omitempty"`
	Group            string `json:"group,omitempty"`
	System           string `json:"system,omitempty"`
	NativeDifficulty string `json:"nativeDifficulty,omitempty"`
	Rank             string `json:"rank,omitempty"`
	Tier             string `json:"tier"`
	Status           string `json:"status"`
	Scenarios        int    `json:"scenarios"`
	Required         int    `json:"required"`
	Samples          int    `json:"samples"`
	Evidence         string `json:"evidence"`
}

// VDIM S5 author's distribution: Bronze Complete -> Novice, Gold Complete
// -> Adept, Diamond Complete -> Intermediate, Master Complete -> Advanced.
// This is a curriculum audience mapping, not cross-system rank equivalence.
func vdimRankTier(rank string) string {
	switch strings.ToLower(rank) {
	case "iron":
		return "entry"
	case "bronze", "silver":
		return "novice"
	case "gold", "platinum":
		return "adept"
	case "diamond", "jade":
		return "intermediate"
	case "master", "grandmaster":
		return "advanced"
	case "nova", "astra", "celestial":
		return "elite"
	}
	return ""
}

var curriculumTiers = []string{"entry", "novice", "adept", "intermediate", "advanced", "elite"}

func tierIndex(t string) int {
	for i, v := range curriculumTiers {
		if v == t {
			return i
		}
	}
	return -1
}

// Choose versions per native difficulty. S5.5 Advanced must not hide S5
// Novice/Intermediate when the newest release has only an Advanced roster.
func automaticReferences(catalog []Scenario, p Preferences) Preferences {
	p.Benchmark, p.Benchmarks = "", nil
	versions := map[string][]int{}
	for _, s := range catalog {
		for _, m := range memberships(s) {
			if m.System == "" {
				continue
			}
			key, v := benchmarkSeries(m.System)
			key += "|" + m.NativeDifficulty
			old, ok := versions[key]
			if !ok || newerBenchmark(v, old) {
				versions[key] = v
			}
		}
	}
	names := map[string]bool{}
	for _, s := range catalog {
		for _, m := range memberships(s) {
			if m.System == "" {
				names[m.Name] = true
				continue
			}
			key, v := benchmarkSeries(m.System)
			if !newerBenchmark(versions[key+"|"+m.NativeDifficulty], v) {
				names[m.Name] = true
			}
		}
	}
	for n := range names {
		if n != "" {
			p.Benchmarks = append(p.Benchmarks, n)
		}
	}
	sort.Strings(p.Benchmarks)
	return p
}

// Reject altered target scales, time dilation, cheats and incomplete runs for
// benchmark rank inference. Ordinary sensitivity changes remain separate via
// comparable signatures. No PB-only rank promotion.
func levelObservations(runs []models.RunRecord, now time.Time) map[string][]observation {
	filtered := []models.RunRecord{}
	for _, r := range runs {
		v := r.Stats.Summary
		if math.IsNaN(v.Duration) || math.IsInf(v.Duration, 0) || math.IsNaN(v.TimeRemaining) || math.IsInf(v.TimeRemaining, 0) || math.IsNaN(v.AvgTargetScale) || math.IsInf(v.AvgTargetScale, 0) || math.IsNaN(v.AvgTimeDilation) || math.IsInf(v.AvgTimeDilation, 0) || v.Duration <= 0 || v.Duration > 3600 || v.TimeRemaining > 1 || v.AvgTargetScale != 0 && math.Abs(v.AvgTargetScale-1) > .001 || v.AvgTimeDilation != 0 && math.Abs(v.AvgTimeDilation-1) > .001 {
			continue
		}
		cheated := false
		for _, e := range r.Stats.Events {
			cheated = cheated || e.Cheated
		}
		if cheated {
			continue
		}
		at, err := time.Parse(time.RFC3339, v.DatePlayed)
		if err != nil || at.After(now) || at.Before(now.AddDate(0, 0, -45)) {
			continue
		}
		// Benchmark rank is independent of ordinary sensitivity/FOV choices.
		// Scene revisions remain separate; altered scale/time are rejected above.
		r.Stats.Summary.HorizSens, r.Stats.Summary.VertSens, r.Stats.Summary.FOV = 0, 0, 0
		r.Stats.Summary.SensScale = ""
		r.Stats.Summary.AvgTimeDilation = 1
		filtered = append(filtered, r)
	}
	return observed(filtered)
}

// Use the smallest recent window with three comparable scores. Older valid
// scores retain their value for up to 45 days; age does not subtract rank.
func levelScoreWindow(rows []observation, now time.Time) (float64, int, int) {
	rows = comparable(rows)
	for _, days := range []int{7, 14, 30, 45} {
		scores := []float64{}
		for _, r := range rows {
			if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -days)) {
				scores = append(scores, r.score)
				if len(scores) == 7 {
					break
				}
			}
		}
		if len(scores) >= 3 {
			return median(scores), len(scores), days
		}
	}
	return 0, 0, 0
}

func recentLevelScore(rows []observation, now time.Time) (float64, int) {
	score, n, _ := levelScoreWindow(rows, now)
	return score, n
}

func validRankCutoffs(m BenchmarkMembership) bool {
	if len(m.Ranks) == 0 || len(m.Ranks) != len(m.Thresholds) {
		return false
	}
	last := -1.0
	for i, v := range m.Thresholds {
		if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) || v <= last || strings.TrimSpace(m.Ranks[i]) == "" {
			return false
		}
		last = v
	}
	return true
}

func PlayerLevels(catalog []Scenario, runs []models.RunRecord, now time.Time) []PlayerLevel {
	obs := levelObservations(runs, now)
	type bucket struct {
		level                          PlayerLevel
		lowest                         int
		ranks                          []string
		historyScenes, benchmarkScenes int
		seen                           map[string]bool
	}
	groups := map[string]*bucket{}
	for _, s := range catalog {
		for _, m := range memberships(s) {
			key := scenarioTheme(s) + "|" + m.System + "|" + m.NativeDifficulty + "|" + m.Category + "|" + m.Group
			b := groups[key]
			if b == nil {
				b = &bucket{level: PlayerLevel{Theme: scenarioTheme(s), Category: m.Category, Group: m.Group, System: m.System, NativeDifficulty: m.NativeDifficulty, Tier: "novice", Status: "insufficient"}, lowest: len(m.Ranks), ranks: m.Ranks, seen: map[string]bool{}}
				groups[key] = b
			}
			name := strings.ToLower(s.Name)
			if b.seen[name] {
				continue
			}
			b.seen[name] = true
			b.level.Required++
			if !validRankCutoffs(m) {
				continue
			}
			ranksMatch := len(b.ranks) == len(m.Ranks)
			if ranksMatch {
				for i := range b.ranks {
					ranksMatch = ranksMatch && b.ranks[i] == m.Ranks[i]
				}
			}
			if !ranksMatch {
				continue
			}
			score, n, days := levelScoreWindow(obs[name], now)
			if n > 0 {
				b.historyScenes++
				b.level.WindowDays = max(b.level.WindowDays, days)
				if len(obs[name]) > 0 && obs[name][0].at.Format(time.RFC3339) > b.level.LastPlayed {
					b.level.LastPlayed = obs[name][0].at.Format(time.RFC3339)
				}
			} else if m.BenchmarkScore != nil && *m.BenchmarkScore > 0 && !math.IsNaN(*m.BenchmarkScore) && !math.IsInf(*m.BenchmarkScore, 0) {
				score = *m.BenchmarkScore
				b.benchmarkScenes++
			} else {
				continue
			}
			b.level.Scenarios++
			b.level.Samples += n
			rank := -1
			for i, t := range m.Thresholds {
				if t > 0 && score >= t {
					rank = i
				}
			}
			if rank < b.lowest {
				b.lowest = rank
			}
		}
	}
	out := []PlayerLevel{}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := groups[k]
		l := b.level
		l.Source = "history"
		l.Evidence = "优先 7 天可比成绩，不足三局依次扩到 14/30/45 天；窗口内不按年龄降档，需覆盖该原生分类全部场景。"
		if b.benchmarkScenes > 0 {
			l.Source = "benchmark"
			if b.historyScenes > 0 {
				l.Source = "mixed"
			}
			l.Evidence += "缺少足够本地历史的场景采用 benchmark 已有成绩；接口未提供达成日期，作为暂定参考，不伪造近期样本。"
		}
		if l.Scenarios == l.Required && l.Required >= 1 {
			l.Status = "inferred"
			l.AtCeiling = b.lowest == len(b.ranks)-1
			if b.benchmarkScenes > 0 {
				l.Status = "estimated"
			}
			if b.lowest >= 0 {
				l.Rank = b.ranks[b.lowest]
			} else {
				l.Rank = "unranked"
			}
			series, _ := benchmarkSeries(l.System)
			if series == "voltaic" {
				l.Tier = vdimRankTier(l.Rank)
				if l.Tier == "" {
					l.Tier = "entry"
				}
			}
		}
		out = append(out, l)
	}
	return out
}

func inferredTier(theme string, levels []PlayerLevel) string {
	tier := ""
	for _, t := range inferredGroupTiers(theme, levels) {
		if tier == "" || tierIndex(t) < tierIndex(tier) {
			tier = t
		}
	}
	if tier == "" {
		return "novice"
	}
	return tier
}

// Whole-category achievement remains conservative. A mixed-goal curriculum
// uses the median of established groups instead of letting one weak group
// lower every training goal. Unknown groups never manufacture a promotion.
func trainingTier(theme string, levels []PlayerLevel) string {
	tiers := []string{}
	for _, tier := range inferredGroupTiers(theme, levels) {
		tiers = append(tiers, tier)
	}
	if len(tiers) == 0 {
		return "novice"
	}
	sort.Slice(tiers, func(i, j int) bool { return tierIndex(tiers[i]) < tierIndex(tiers[j]) })
	return tiers[len(tiers)/2]
}

func inferredGroupTiers(theme string, levels []PlayerLevel) map[string]string {
	// A completed top rank on an easier roster is a lower bound, not proof
	// that the player lost an established higher rank. Unsaturated recent
	// evidence does take priority over an undated higher PB.
	local, fallback := map[string]string{}, map[string]string{}
	bounded := map[string]bool{}
	for _, l := range levels {
		series, _ := benchmarkSeries(l.System)
		if l.Theme != theme || series != "voltaic" || (l.Status != "inferred" && l.Status != "estimated") {
			continue
		}
		key := strings.ToLower(l.Category + "|" + l.Group)
		target := fallback
		if l.Status == "inferred" || l.Source == "mixed" {
			target = local
			bounded[key] = bounded[key] || !l.AtCeiling
		}
		if tierIndex(l.Tier) > tierIndex(target[key]) {
			target[key] = l.Tier
		}
	}
	best := map[string]string{}
	for k, v := range local {
		best[k] = v
	}
	for k, v := range fallback {
		if bounded[k] {
			continue
		}
		if tierIndex(v) > tierIndex(best[k]) {
			best[k] = v
		}
	}
	return best
}
