package training

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aimmeow/internal/models"
)

func TestAdjacentSameScenarioRowsTrackCountsAndRestart(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Import([]byte(`{"playlistName":"VDIM Novice Clicking I","scenarioList":[{"scenarioName":"Static A","playCount":2},{"scenarioName":"Static A","playCount":1},{"scenarioName":"Static B","playCount":1}]}`), Source{URL: "fixture:adjacent"})
	if err != nil {
		t.Fatal(err)
	}
	prefs := defaults()
	_, err = s.Generate(prefs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("start", epoch, nil); err != nil {
		t.Fatal(err)
	}
	rows := []models.RunRecord{
		record("a1", "Static A", 60, 1, epoch.Add(time.Minute)),
		record("a2", "Static A", 60, 1, epoch.Add(2*time.Minute)),
		record("a3", "Static A", 60, 1, epoch.Add(3*time.Minute)),
	}
	// Out-of-order watcher batch is sorted; the third A belongs to row two.
	s.Tick(epoch.Add(3*time.Minute), []models.RunRecord{rows[2], rows[0], rows[1]})
	p := s.Snapshot(nil).Plan
	if p.Index != 1 || p.Blocks[0].Runs != 2 || p.Blocks[1].Runs != 1 || p.Recorded != 180 {
		t.Fatalf("adjacent repetitions misattributed: %+v", p)
	}
	s, err = New(filepath.Dir(s.path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("start", epoch.Add(3*time.Minute), rows); err != nil {
		t.Fatal(err)
	}
	rows = append(rows, record("b1", "Static B", 60, 1, epoch.Add(4*time.Minute)))
	s.Tick(epoch.Add(4*time.Minute), rows)
	s.Tick(epoch.Add(4*time.Minute), rows)
	p = s.Snapshot(nil).Plan
	if p.Status != "completed" || p.Recorded != 240 || p.Blocks[2].Runs != 1 || !completedCurriculum(*p, s.state.Curricula[0]) {
		t.Fatalf("restart/duplicate ingestion broke baseline: %+v", p)
	}
}

func TestAutomaticTemplateSelectionFitsBudgetAndIgnoresStaleManualTier(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{
		`{"playlistName":"VDIM Novice Clicking I Long","scenarioList":[{"scenarioName":"Static Long","playCount":20}]}`,
		`{"playlistName":"VDIM Adept Tracking I Short","scenarioList":[{"scenarioName":"Smooth Short","playCount":1}]}`,
	} {
		if _, err = s.Import([]byte(data), Source{URL: "fixture:budget"}); err != nil {
			t.Fatal(err)
		}
	}
	prefs := defaults()
	prefs.Minutes = 5
	// A single long run exceeds the budget even after adaptive repetition.
	s.state.Catalog[0].Seconds = 600
	// The long template has less recent exposure and would previously win.
	runs := []models.RunRecord{record("short-history", "Smooth Short", 60, 1, time.Now().Add(-time.Hour))}
	p, err := s.Generate(prefs, runs)
	if err != nil || p.CurriculumID != s.state.Curricula[1].ID {
		t.Fatal("did not select the only fitting template", err)
	}
	prefs.Focus, prefs.Difficulty = "static", "novice"
	if _, err = s.Generate(prefs, nil); err == nil {
		t.Fatal("oversized template accepted")
	}
	prefs.Focus = "auto"
	prefs.CurriculumID = s.state.Curricula[0].ID
	prefs.Difficulty = "elite"
	prefs.Benchmark = "obsolete manual selection"
	p, err = s.Generate(prefs, nil)
	if err != nil || p.CurriculumID != s.state.Curricula[1].ID || p.Preferences.Difficulty != "any" {
		t.Fatal("stale manual selection overrode automatic inference", err)
	}
	for _, tier := range []string{"entry", "novice", "adept", "intermediate", "advanced", "elite"} {
		prefs.Difficulty = tier
		if err = validatePreferences(prefs); err != nil {
			t.Fatalf("native template tier %s rejected: %v", tier, err)
		}
	}
}

func TestManualFinishAndSkippedRowsDoNotUnlockExploration(t *testing.T) {
	s, template := templateFixture(t)
	p, err := s.Generate(defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("start", epoch, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("finish", epoch.Add(time.Minute), []models.RunRecord{record("one", "Static A", 60, 1, epoch.Add(time.Minute))}); err != nil {
		t.Fatal(err)
	}
	p = s.Snapshot(nil).Plan
	if p.Recorded != 60 || completedCurriculum(*p, template) {
		t.Fatal("early finish unlocked exploration")
	}
	for i := range p.Blocks {
		b := &p.Blocks[i]
		b.Runs, b.Recorded, b.Outcome = b.PlayCount, float64(b.PlayCount*60), "list_complete"
	}
	if !completedCurriculum(*p, template) {
		t.Fatal("fully recorded baseline rejected")
	}
	for _, outcome := range []string{"skipped", "missed", "session_limit"} {
		p.Blocks[1].Outcome = outcome
		if completedCurriculum(*p, template) {
			t.Fatalf("%s row unlocked exploration", outcome)
		}
	}
	p.Blocks[1].Outcome = "list_complete"
	template.Rows[1].Count++
	if completedCurriculum(*p, template) {
		t.Fatal("changed template reused stale baseline")
	}
}

func TestManagedPlaylistOwnerFailureRemovesUnownedFile(t *testing.T) {
	s, _ := templateFixture(t)
	if _, err := s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	// A nonempty directory at the marker path forces a portable write failure.
	other := t.TempDir()
	owner := filepath.Join(other, "AimMeow-Current.json.owner")
	if err := os.Mkdir(owner, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(owner, "block"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Install(other); err == nil {
		t.Fatal("owner write failure accepted")
	}
	if _, err := os.Stat(filepath.Join(other, "AimMeow-Current.json")); !os.IsNotExist(err) {
		t.Fatal("unowned new playlist left behind", err)
	}
}
