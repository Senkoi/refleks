package runs

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeScenarioHistoryRun(t *testing.T, dir, name, filename string, at time.Time, score float64) string {
	t.Helper()
	r := sampleRecord()
	r.FileName = strings.TrimSuffix(filename, ".refleks")
	r.EpochMilli = at.UnixMilli()
	r.Stats.Summary.Scenario = name
	r.Stats.Summary.DatePlayed = at.Format(time.RFC3339)
	r.Stats.Summary.Score = score
	r.Stats.Summary.Duration = 60
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeRecord(f, r); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScenarioHistoryIncludesOldAndSingleRunsWithoutLoadingOtherMaps(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(nil)
	old := time.Now().AddDate(0, 0, -100).Truncate(time.Second)
	recent := old.AddDate(0, 0, 99)
	file := func(name string, at time.Time) string {
		return name + " - Challenge - " + at.Format("2006.01.02-15.04.05") + ".refleks"
	}
	writeScenarioHistoryRun(t, dir, "Map X", file("Map X", old), old, 100)
	writeScenarioHistoryRun(t, dir, "Map X", file("Map X", recent), recent, 0)
	other := writeScenarioHistoryRun(t, dir, "Map X Small", file("Map X Small", recent), recent, 999)
	points, err := store.loadScenarioHistory(dir, "  map x  ")
	if err != nil || len(points) != 2 {
		t.Fatal(points, err)
	}
	if points[0].At != old.UnixMilli() || points[0].Score != 100 || points[1].Score != 0 {
		t.Fatalf("old/zero score or order lost: %+v", points)
	}
	if _, cached := store.index.cachedRecord(other); cached {
		t.Fatal("read unrelated map while opening one history")
	}
	encoded, err := json.Marshal(points)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"mouseTrace", "events", "filePath", "env"} {
		if strings.Contains(string(encoded), field) {
			t.Fatal("non-chart payload leaked", field)
		}
	}
	one, err := store.loadScenarioHistory(dir, "Map X Small")
	if err != nil || len(one) != 1 || one[0].Score != 999 {
		t.Fatal("single played run must be available", one, err)
	}
}

func TestScenarioHistoryRefreshesNamesAndScoresAndHandlesLegacyFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(nil)
	at := time.Now().AddDate(0, 0, -90).Truncate(time.Second)
	path := writeScenarioHistoryRun(t, dir, "Legacy Map", "legacy.refleks", at, 10)
	writeScenarioHistoryRun(t, dir, "Legacy Map Small", "legacy-small.refleks", at, 20)
	writeScenarioHistoryRun(t, dir, "Legacy Map", "invalid-score.refleks", at, math.NaN())
	names, err := store.loadPlayedScenarioNames(dir)
	if err != nil || len(names) != 2 {
		t.Fatal(names, err)
	}
	points, err := store.loadScenarioHistory(dir, "Legacy Map")
	if err != nil || len(points) != 1 || points[0].Score != 10 {
		t.Fatal(points, err)
	}
	writeScenarioHistoryRun(t, dir, "Legacy Map", "legacy.refleks", at, 42)
	store.index.add(dir, filepath.Base(path), at.UnixMilli())
	points, err = store.loadScenarioHistory(dir, "Legacy Map")
	if err != nil || len(points) != 1 || points[0].Score != 42 {
		t.Fatal("stale after save", points, err)
	}
	points[0].Score = -999
	again, _ := store.loadScenarioHistory(dir, "Legacy Map")
	if again[0].Score != 42 {
		t.Fatal("caller mutated history cache")
	}
	writeScenarioHistoryRun(t, dir, "Brand New Map", "new.refleks", at, 50)
	store.index.add(dir, "new.refleks", at.UnixMilli())
	names, err = store.loadPlayedScenarioNames(dir)
	if err != nil || len(names) != 3 {
		t.Fatal("new played map missing", names, err)
	}
	empty, err := store.loadScenarioHistory(dir, "Unplayed Map")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("empty history must be an empty list", empty, err)
	}
}
