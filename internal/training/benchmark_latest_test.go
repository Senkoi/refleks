package training

import "testing"

func TestLatestBenchmarkPreference(t *testing.T) {
	pool := []Scenario{}
	for _, m := range []BenchmarkMembership{
		{Name: "VT S4 / Novice", System: "Voltaic S4"},
		{Name: "VT S5 / Novice", System: "Voltaic S5"},
		{Name: "VT S5.5 / Novice", System: "Voltaic S5.5"},
		{Name: "VT S5.5 / Advanced", System: "Voltaic S5.5"},
		{Name: "RA S5 / All", System: "Revosect S5"},
		{Name: "AS / Easy", System: "AimSpeed Benchmarks"},
		{Name: "AS2 / Easy", System: "AimSpeed Benchmarks 2.0"},
		{Name: "Special / All", System: "Special Other"},
	} {
		pool = append(pool, Scenario{Benchmarks: []BenchmarkMembership{m}})
	}
	p := latestBenchmarkPreferences(pool, Preferences{})
	if len(p.Benchmarks) != 5 {
		t.Fatalf("incorrect latest series: %+v", p)
	}
	for _, old := range []string{"VT S4 / Novice", "VT S5 / Novice", "AS / Easy"} {
		if selectedBenchmark(p, old) {
			t.Fatalf("selected old %s", old)
		}
	}
	for _, latest := range []string{"VT S5.5 / Novice", "VT S5.5 / Advanced", "RA S5 / All", "AS2 / Easy", "Special / All"} {
		if !selectedBenchmark(p, latest) {
			t.Fatalf("missing %s", latest)
		}
	}
	explicit := Preferences{Benchmark: "VT S4 / Novice"}
	if got := latestBenchmarkPreferences(pool, explicit); got.Benchmark != explicit.Benchmark || len(got.Benchmarks) != 0 {
		t.Fatal("overrode explicit old version")
	}
	if !newerBenchmark([]int{5, 10}, []int{5, 5}) || newerBenchmark([]int{5}, []int{5, 0}) {
		t.Fatal("incorrect numeric ordering")
	}
}
