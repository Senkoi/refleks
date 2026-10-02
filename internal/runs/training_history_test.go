package runs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrainingHistoryReadsFullWindowAndRefreshesAfterNewRecords(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(nil)
	write := func(name string, age int, score float64) string {
		t.Helper()
		r := sampleRecord()
		at := time.Now().AddDate(0, 0, -age)
		r.EpochMilli = at.UnixMilli()
		r.Stats.Summary.DatePlayed = at.Format(time.RFC3339)
		r.Stats.Summary.Score = score
		path := filepath.Join(dir, name+".refleks")
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
	write("recent", 2, 1)
	write("forty days", 40, 2)
	write("expired", 46, 3)
	rows, err := store.loadTrainingRuns(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(store.index.recent(0, 7, 0)) != 1 {
		t.Fatal("training inherited narrow history-display window", len(rows))
	}
	// Add/re-save uses the index revision, so the summary cache cannot hide
	// newly arrived history or a changed score until a timer expires.
	path := write("recent", 2, 99)
	store.index.add(dir, filepath.Base(path), time.Now().AddDate(0, 0, -2).UnixMilli())
	rows, err = store.loadTrainingRuns(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		found = found || r.Stats.Summary.Score == 99
	}
	if !found {
		t.Fatal("new score hidden by training cache")
	}
	rows[0].Stats.Summary.Score = -999
	again, err := store.loadTrainingRuns(dir)
	if err != nil || again[0].Stats.Summary.Score == -999 {
		t.Fatal("caller mutated cached history")
	}
}
