package practice

import (
	"aimmeow/internal/models"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestRestBoundaryIncludesFirstRunAndKeepsZeroScores(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	first := now.Add(-time.Hour).UnixMilli()
	entries := []Entry{{RunID: "B", EndedAt: first + 21*60000, Seconds: 120}, {RunID: "A", EndedAt: first, Seconds: 60}}
	g := Group(entries, 20*time.Minute, now)
	if len(g) != 1 || g[0].StartedAt != first-60000 || g[0].EndedAt != entries[0].EndedAt || g[0].ID != "session-A" {
		t.Fatal("19 minutes rest should remain one interval", g)
	}
	entries[0].EndedAt += 60000
	g = Group(entries, 20*time.Minute, now)
	if len(g) != 2 {
		t.Fatal("exactly 20 minutes rest must split", g)
	}
}
func TestSessionIdentityAndLegacyAliasesAreDeterministic(t *testing.T) {
	now := time.Now()
	at := now.Add(-time.Hour).UnixMilli()
	entries := []Entry{{"b", at + 60000, 60}, {"a", at, 60}, {"a", at, 60}, {"invalid", at, math.NaN()}, {"future", now.Add(time.Hour).UnixMilli(), 60}}
	groups := Group(entries, 0, now)
	if len(groups) != 1 || !reflect.DeepEqual(groups[0].RunIDs, []string{"a", "b"}) || len(groups[0].LegacyIDs) != 4 {
		t.Fatal(groups)
	}
	original := groups[0].ID
	entries = append(entries, Entry{"c", at + 120000, 60})
	if Group(entries, 0, now)[0].ID != original {
		t.Fatal("adding new records renamed interval")
	}
	// One recorded run has its whole duration, rather than a zero-length span.
	single := Group(entries[:1], 0, now)[0]
	if single.EndedAt-single.StartedAt != 60000 {
		t.Fatal(single)
	}
}
func TestComparisonSeparatesMeasurementConditions(t *testing.T) {
	base := models.RunStatsSummary{Hash: "v1", GameVersion: "game", Duration: 60, HorizSens: 1, VertSens: 1, FOV: 103, DPI: 800, AvgTimeDilation: 1, AvgTargetScale: 1}
	key := ComparisonKey(base)
	for _, change := range []func(*models.RunStatsSummary){func(s *models.RunStatsSummary) { s.Hash = "v2" }, func(s *models.RunStatsSummary) { s.GameVersion = "game2" }, func(s *models.RunStatsSummary) { s.Duration = 120 }, func(s *models.RunStatsSummary) { s.DPI = 1600 }, func(s *models.RunStatsSummary) { s.FOVScale = "different" }, func(s *models.RunStatsSummary) { s.AvgTargetScale = .5 }} {
		s := base
		change(&s)
		if ComparisonKey(s) == key {
			t.Fatal("incompatible scores joined", s)
		}
	}
}
