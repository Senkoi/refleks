package training

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func legacySlot(t *testing.T, dir, planID string) (string, []byte) {
	t.Helper()
	path := filepath.Join(dir, "Refleks-Adaptive-Current.json")
	b := []byte(`{"playlistName":"Refleks Adaptive Current","scenarioList":[]}`)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	marker, _ := json.Marshal(playlistOwner{Version: 1, PlanID: planID, Hash: playlistHash(b)})
	if err := os.WriteFile(path+".owner", marker, 0600); err != nil {
		t.Fatal(err)
	}
	return path, b
}

func TestBrandPlaylistMigratesOnlyVerifiedOwnedLegacySlot(t *testing.T) {
	for _, tampered := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned", true: "user-modified"}[tampered], func(t *testing.T) {
			s, _ := templateFixture(t)
			if _, err := s.Generate(defaults(), nil); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			old, original := legacySlot(t, dir, "previous-plan")
			if tampered {
				original = []byte(`{"playlistName":"my edited training"}`)
				if err := os.WriteFile(old, original, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, err := s.Install(dir)
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var payload struct {
				Name string `json:"playlistName"`
			}
			if json.Unmarshal(b, &payload) != nil || payload.Name != "AimMeow Current" {
				t.Fatalf("wrong brand payload: %s", b)
			}
			remaining, err := os.ReadFile(old)
			if tampered {
				if err != nil || !bytes.Equal(remaining, original) {
					t.Fatal("user-edited legacy slot was changed")
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("verified legacy slot was not removed")
			}
		})
	}
}

func TestBrandMigrationLeavesPausedLegacyPlanUnchanged(t *testing.T) {
	s, _ := templateFixture(t)
	if _, err := s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path, before := legacySlot(t, dir, s.state.Plan.ID)
	s.state.Plan.Status = "paused"
	got, err := s.Install(dir)
	if err != nil || got != path {
		t.Fatalf("legacy resume = %q, %v", got, err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("paused playlist was rewritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "AimMeow-Current.json")); !os.IsNotExist(err) {
		t.Fatal("paused plan created a second slot")
	}
}
