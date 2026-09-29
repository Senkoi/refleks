package training

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Labels inferred from names are deliberately marked inferred, never expert-verified.
func classify(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "switch"), strings.Contains(n, "evats"), strings.Contains(n, "voxts"), strings.Contains(n, "dotts"), strings.Contains(n, "kints"):
		return "switching"
	case strings.Contains(n, "smooth"), strings.Contains(n, "centering"), strings.Contains(n, "pgt"), strings.Contains(n, "precise"), strings.Contains(n, "steady"):
		return "smooth"
	case strings.Contains(n, "reactiv"), strings.Contains(n, "air "), strings.Contains(n, "ground"), strings.Contains(n, "lddh"), strings.Contains(n, "fuglaaxy"), strings.Contains(n, "strafetrack"):
		return "reactive"
	case strings.Contains(n, "pasu"), strings.Contains(n, "popcorn"), strings.Contains(n, "bounce"), strings.Contains(n, "dynamic"), strings.Contains(n, "b180"):
		return "dynamic"
	case strings.Contains(n, "static"), strings.Contains(n, "1w"), strings.Contains(n, "wall"), strings.Contains(n, "sphere hipfire"), strings.Contains(n, "micro flick"), strings.Contains(n, "multiclick"):
		return "static"
	}
	return "unknown"
}

var variantWords = regexp.MustCompile(`(?i)\b(easy|easier|hard|harder|small|smaller|large|larger|novice|intermediate|advanced|reloaded|reload|v\d+|\d+%|\d+s)\b`)
var space = regexp.MustCompile(`\s+`)

func family(name string) string {
	return strings.TrimSpace(space.ReplaceAllString(variantWords.ReplaceAllString(strings.ToLower(name), ""), " "))
}
func difficulty(name string) string {
	n := strings.ToLower(name)
	for _, pair := range [][2]string{{"novice", "novice"}, {"easy", "novice"}, {"intermediate", "intermediate"}, {"advanced", "advanced"}, {"hard", "advanced"}} {
		if strings.Contains(n, pair[0]) {
			return pair[1]
		}
	}
	return "unknown"
}

func memberships(s Scenario) []BenchmarkMembership {
	if len(s.Benchmarks) > 0 {
		return s.Benchmarks
	}
	if s.Benchmark != "" {
		return []BenchmarkMembership{{Name: s.Benchmark, Thresholds: s.Thresholds}}
	}
	return nil
}

func member(s Scenario, benchmark string) (BenchmarkMembership, bool) {
	for _, m := range memberships(s) {
		if m.Name == benchmark {
			return m, true
		}
	}
	return BenchmarkMembership{}, false
}

func related(s Scenario, benchmark string) bool {
	for _, name := range s.RelatedBenchmarks {
		if name == benchmark {
			return true
		}
	}
	return false
}

func mergeStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range append(append([]string{}, a...), b...) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func mergeMemberships(a, b []BenchmarkMembership) []BenchmarkMembership {
	out := append([]BenchmarkMembership{}, a...)
	for _, item := range b {
		if item.Name == "" {
			continue
		}
		found := false
		for i := range out {
			if out[i].Name == item.Name {
				if len(item.Thresholds) > 0 {
					out[i].Thresholds = item.Thresholds
				}
				found = true
				break
			}
		}
		if !found {
			out = append(out, item)
		}
	}
	return out
}

// ParsePlaylist accepts actual KovaaK's playlist JSON, not arbitrary text or invented scenario names.
func ParsePlaylist(data []byte, source Source) ([]Scenario, error) {
	var p struct {
		Name      string `json:"playlistName"`
		Scenarios []struct {
			Name string `json:"scenarioName"`
		} `json:"scenarioList"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("需要 KovaaK's playlist JSON（scenarioList）：%w", err)
	}
	if len(p.Scenarios) == 0 || len(p.Scenarios) > 2000 {
		return nil, fmt.Errorf("列表为空或超过 2000 张关卡")
	}
	if p.Name != "" {
		source.Title = p.Name
	}
	if source.Retrieved == "" {
		source.Retrieved = time.Now().UTC().Format(time.RFC3339)
	}
	result := []Scenario{}
	seen := map[string]bool{}
	for _, row := range p.Scenarios {
		name := strings.TrimSpace(row.Name)
		if name == "" || len(name) > 300 || strings.ContainsAny(name, "\r\n\x00") || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		skill := classify(name)
		result = append(result, Scenario{Name: name, Skill: skill, Family: family(name), Difficulty: difficulty(name), Seconds: 60, Sources: []Source{source}, Classification: "inferred", Enabled: validSkill(skill)})
	}
	return result, nil
}

func mergeCatalog(existing, incoming []Scenario) []Scenario {
	index := map[string]int{}
	for i, s := range existing {
		index[strings.ToLower(s.Name)] = i
	}
	for _, s := range incoming {
		if i, ok := index[strings.ToLower(s.Name)]; ok {
			e := &existing[i]
			for _, src := range s.Sources {
				found := false
				for j, old := range e.Sources {
					if old.URL == src.URL && old.Title == src.Title {
						e.Sources[j] = src
						found = true
						break
					}
				}
				if !found {
					e.Sources = append(e.Sources, src)
				}
			}
			e.Benchmarks = mergeMemberships(memberships(*e), memberships(s))
			e.RelatedBenchmarks = mergeStrings(e.RelatedBenchmarks, s.RelatedBenchmarks)
			if e.Classification != "manual" && s.VariantOf != "" {
				e.VariantOf = s.VariantOf
			}
			if e.Classification != "manual" && s.Classification == "benchmark" {
				e.Skill = s.Skill
				e.Classification = s.Classification
				e.Enabled = validSkill(s.Skill)
				e.Difficulty = s.Difficulty
			}
		} else if len(existing) < 10000 {
			s.Benchmarks = mergeMemberships(nil, memberships(s))
			index[strings.ToLower(s.Name)] = len(existing)
			existing = append(existing, s)
		}
	}
	return existing
}
