package training

import (
	"math"
	"refleks/internal/models"
	"sort"
	"strings"
	"time"
)

type PlayerLevel struct {
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
		if v.Duration <= 0 || v.Duration > 3600 || v.TimeRemaining > 1 || v.AvgTargetScale != 0 && math.Abs(v.AvgTargetScale-1) > .001 || v.AvgTimeDilation != 0 && math.Abs(v.AvgTimeDilation-1) > .001 {
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
		if err != nil || at.After(now) || at.Before(now.AddDate(0, 0, -30)) {
			continue
		}
		filtered = append(filtered, r)
	}
	return observed(filtered)
}

// Recent weighted median: seven-day half life, minimum three comparable runs
// and a recent sample. Every scenario in a category/tier roster must establish
// the threshold before that roster can promote the category.
func recentLevelScore(rows []observation, now time.Time) (float64, int) {
	rows = comparable(rows)
	if len(rows) < 3 || now.Sub(rows[0].at) > 7*24*time.Hour {
		return 0, 0
	}
	type weighted struct{ score, weight float64 }
	values := []weighted{}
	total := 0.0
	for _, r := range rows {
		w := math.Exp2(-now.Sub(r.at).Hours() / (7 * 24))
		total += w
		values = append(values, weighted{r.score, w})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].score < values[j].score })
	acc := 0.0
	for _, v := range values {
		acc += v.weight
		if acc >= total/2 {
			return v.score, len(rows)
		}
	}
	return 0, 0
}

func PlayerLevels(catalog []Scenario, runs []models.RunRecord, now time.Time) []PlayerLevel {
	obs := levelObservations(runs, now)
	p := automaticReferences(catalog, Preferences{})
	type bucket struct {
		level  PlayerLevel
		lowest int
		ranks  []string
	}
	groups := map[string]*bucket{}
	for _, s := range catalog {
		for _, m := range memberships(s) {
			if !selectedBenchmark(p, m.Name) || len(m.Ranks) != len(m.Thresholds) || len(m.Ranks) == 0 {
				continue
			}
			key := scenarioTheme(s) + "|" + m.System + "|" + m.NativeDifficulty + "|" + m.Category + "|" + m.Group
			b := groups[key]
			if b == nil {
				b = &bucket{level: PlayerLevel{Theme: scenarioTheme(s), Category: m.Category, Group: m.Group, System: m.System, NativeDifficulty: m.NativeDifficulty, Tier: "novice", Status: "insufficient"}, lowest: len(m.Ranks), ranks: m.Ranks}
				groups[key] = b
			}
			b.level.Required++
			score, n := recentLevelScore(obs[strings.ToLower(s.Name)], now)
			if n == 0 {
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
		l.Evidence = "每张场景至少 3 局同设置成绩，30 天窗口、7 天半衰期；需覆盖该分类全部场景。"
		if l.Scenarios == l.Required && l.Required >= 1 {
			l.Status = "inferred"
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
	// Use the weakest covered subcategory after taking its strongest established
	// native tier. Missing subcategory evidence remains provisional Novice.
	best := map[string]string{}
	for _, l := range levels {
		series, _ := benchmarkSeries(l.System)
		if l.Theme != theme || series != "voltaic" {
			continue
		}
		key := l.Category + "|" + l.Group
		if _, ok := best[key]; !ok {
			best[key] = ""
		}
		if l.Status == "inferred" && tierIndex(l.Tier) > tierIndex(best[key]) {
			best[key] = l.Tier
		}
	}
	tier := ""
	for _, t := range best {
		if t == "" {
			t = "novice"
		}
		if tier == "" || tierIndex(t) < tierIndex(tier) {
			tier = t
		}
	}
	if tier == "" {
		return "novice"
	}
	return tier
}
