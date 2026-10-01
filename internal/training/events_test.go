package training

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"refleks/internal/models"
)

func TestGameEventsTrackRestartsWithoutInventingCompletedRuns(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.ExecutionMode = "playlist"
	if _, err := s.Action("start", epoch, nil); err != nil { t.Fatal(err) }
	start := func(kind string, seconds int, name string) {
		t.Helper()
		when := epoch.Add(time.Duration(seconds) * time.Second)
		if err := s.ApplyGameEvent(GameEvent{Type: kind, Scenario: name, At: when.UnixMilli()}, when); err != nil { t.Fatal(err) }
	}
	start("challenge_start", 1, "Smooth")
	start("challenge_restart", 41, "Smooth")
	start("challenge_restart", 81, "Smooth")
	start("challenge_restart", 121, "Smooth")
	p := s.Snapshot(nil).Plan
	if p.Game.Restarts != 3 || p.Game.AbortedSeconds != 120 || p.Recorded != 0 || p.Blocks[0].Runs != 0 || p.Index != 0 {
		t.Fatalf("restart incorrectly counted as a finished run: %+v", p)
	}
	if !strings.Contains(s.TakeReminder(), "反复重开") || s.TakeReminder() != "" {
		t.Fatal("repeat reminder should be delivered once")
	}
	start("challenge_complete", 161, "Smooth")
	if p = s.Snapshot(nil).Plan; p.Game.Phase != "between" || p.Recorded != 0 {
		t.Fatal("lifecycle event invented a score")
	}
	start("challenge_start", 170, "Other")
	if p = s.Snapshot(nil).Plan; p.Game.Scenario != "Other" || p.Index != 0 {
		t.Fatal("unrelated scenario advanced the plan")
	}
}

func TestGameEventReaderSkipsHistoryAndWaitsForCompleteLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), GameEventsFile)
	old := `{"type":"challenge_start","scenario":"Old","at":1}` + "\n"
	if err := os.WriteFile(path, []byte(old), 0600); err != nil { t.Fatal(err) }
	r := NewEventReader(path)
	if events, err := r.Read(); err != nil || len(events) != 0 { t.Fatalf("replayed old events: %v %v", events, err) }
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil { t.Fatal(err) }
	defer f.Close()
	_, _ = f.WriteString(`{"type":"challenge_restart","scenario":"Smooth","at":1000}`)
	if events, err := r.Read(); err != nil || len(events) != 0 { t.Fatalf("read incomplete line: %v %v", events, err) }
	_, _ = f.WriteString("\n")
	if events, err := r.Read(); err != nil || len(events) != 1 || events[0].Scenario != "Smooth" {
		t.Fatalf("lost completed event: %v %v", events, err)
	}
}

func TestFinishedLaterPlaylistRunRealignsProgress(t *testing.T) {
	s := fixture(t)
	s.state.Plan.Preferences.ExecutionMode = "playlist"
	s.state.Plan.Blocks[0].PlayCount = 5
	s.state.Plan.Blocks[1].PlayCount = 2
	if _, err := s.Action("start", epoch, nil); err != nil { t.Fatal(err) }
	s.Tick(epoch.Add(time.Minute), []models.RunRecord{record("later", "Benchmark", 60, 40, epoch.Add(time.Minute))})
	p := s.Snapshot(nil).Plan
	if p.Index != 1 || p.Blocks[0].Outcome != "missed" || p.Blocks[1].Runs != 1 || p.Recorded != 60 {
		t.Fatalf("finished later row was dropped: %+v", p)
	}
}
