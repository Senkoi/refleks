package training

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Only explicit trailing season/version numbers establish version ordering.
// dateAdded is a catalog insertion date, not a release date. Do not merge
// differently named products or use benchmark IDs as chronology.
var benchmarkVersion = regexp.MustCompile(`(?i)^(.*?)\s+(?:s|v)?(\d+(?:\.\d+)*)$`)

func benchmarkSeries(system string) (string, []int) {
	name := strings.ToLower(strings.TrimSpace(system))
	parts := benchmarkVersion.FindStringSubmatch(name)
	if parts == nil {
		return name, nil
	}
	version := []int{}
	for _, n := range strings.Split(parts[2], ".") {
		v, _ := strconv.Atoi(n)
		version = append(version, v)
	}
	return strings.TrimSpace(parts[1]), version
}

func newerBenchmark(a, b []int) bool {
	for i := 0; i < len(a) || i < len(b); i++ {
		av, bv := 0, 0
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

// Resolve only the automatic choice; explicit selections remain unchanged.
// Include every native tier of the latest available version in each series.
func latestBenchmarkPreferences(catalog []Scenario, p Preferences) Preferences {
	if p.Benchmark != "" || len(p.Benchmarks) > 0 {
		return p
	}
	versions := map[string][]int{}
	for _, s := range catalog {
		for _, m := range memberships(s) {
			if m.System == "" {
				continue
			}
			key, v := benchmarkSeries(m.System)
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
			if !newerBenchmark(versions[key], v) {
				names[m.Name] = true
			}
		}
	}
	p.Benchmarks = []string{}
	for name := range names {
		if name != "" {
			p.Benchmarks = append(p.Benchmarks, name)
		}
	}
	sort.Strings(p.Benchmarks)
	return p
}
