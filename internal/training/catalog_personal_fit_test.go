package training

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCatalogPersonalFitDoesNotRequireLocalFile(t *testing.T) {
	for _, status := range []string{"", "invalid_local_file", "ambiguous_local_versions", "local_file_unavailable"} {
		t.Run(status, func(t *testing.T) {
			scene := Scenario{Name: "fixture", Benchmarks: []BenchmarkMembership{{Name: "fixture", Thresholds: []float64{100, 200}}}}
			if status != "" {
				scene.LocalAssessment = &LocalAssessment{Status: status}
			}
			rows := []observation{}
			for i := 1; i <= 3; i++ {
				rows = append(rows, observation{score: 150, duration: 60, at: epoch.Add(-time.Duration(i) * time.Hour), signature: "same-settings"})
			}
			assessment := assessCatalog(scene, rows, epoch, Preferences{})
			if assessment.Fit != "suitable" || assessment.Samples != 3 {
				t.Fatalf("personal history was hidden by file state: %+v", assessment)
			}
			if assessment.HasDifficultyEvidence || assessment.HasPrecisionReference || assessment.FileStatus == "verified" {
				t.Fatalf("missing file acquired mechanism evidence: %+v", assessment)
			}
			scene.PersonalDifficulty = "hard"
			if got := assessCatalog(scene, nil, epoch, Preferences{}); got.Fit != "challenging" || got.Samples != 0 {
				t.Fatalf("manual feedback requires a file or fabricated samples: %+v", got)
			}
		})
	}
}

func TestDiscoveryPreferencePersistsWithoutChangingFixedPlan(t *testing.T) {
	dir := t.TempDir()
	service, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	service.state.Plan = &Plan{ID: "fixed", Status: "paused", Preferences: defaults(), Blocks: []Block{{Scenario: Scenario{Name: "fixed scene"}, PlayCount: 3, Budget: 180}}}
	before, _ := json.Marshal(service.state.Plan)
	if err := service.SetAutoDiscover(false); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(service.state.Plan)
	if string(before) != string(after) {
		t.Fatal("discovery preference rewrote the generated list")
	}
	reopened, err := New(dir)
	if err != nil || reopened.state.Preferences.AutoDiscover {
		t.Fatalf("preference did not persist: %v", err)
	}
	if reopened.state.Plan.ID != "fixed" || reopened.state.Plan.Blocks[0].PlayCount != 3 {
		t.Fatal("persisted plan changed")
	}
}
