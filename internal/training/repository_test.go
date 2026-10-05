package training

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repositoryFixture(t *testing.T) (*stateRepository, State) {
	t.Helper()
	r := newStateRepository(filepath.Join(t.TempDir(), "adaptive-training.json"))
	_, m, a, err := ParseLocalSCE([]byte(localSCEFixture))
	if err != nil {
		t.Fatal(err)
	}
	s := State{Version: 1, Preferences: defaults(), Catalog: []Scenario{{Name: "Static A", Skill: "static", LocalAssessment: a, Mechanics: m}}, Plan: &Plan{ID: "p", Status: "running", Blocks: []Block{{Scenario: Scenario{Name: "Static A", Mechanics: m, LocalAssessment: a}, Reason: "selected once", Target: 100, PlayCount: 2, Budget: 120, Outcome: "pending"}}}, History: []Plan{{ID: "old", Status: "completed"}}}
	return r, s
}
func TestLegacyMigrationPreservesExactBackupAndAllDomains(t *testing.T) {
	r, st := repositoryFixture(t)
	raw, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(r.path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := r.Load(State{})
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Save(loaded); err != nil {
		t.Fatal(err)
	}
	backup, _ := os.ReadFile(r.path + ".legacy-v1.json")
	if !bytes.Equal(raw, backup) {
		t.Fatal("legacy source not backed up exactly")
	}
	restored, err := newStateRepository(r.path).Load(State{})
	if err != nil || restored.Plan.ID != "p" || len(restored.History) != 1 || restored.Catalog[0].LocalAssessment.Requirements == nil {
		t.Fatal("domain roundtrip", err)
	}
	if len(r.parts) != 8 {
		t.Fatal("domain partitions")
	}
}
func TestClockAndAnalysisChangesDoNotRewriteFixedPlan(t *testing.T) {
	r, st := repositoryFixture(t)
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	before := map[string]statePart{}
	for k, v := range r.parts {
		before[k] = v
	}
	st.Plan.Elapsed = 17
	if err := r.SaveExecution(st); err != nil {
		t.Fatal(err)
	}
	for k, v := range r.parts {
		if k != "execution" && v != before[k] {
			t.Fatal("clock rewrote", k)
		}
	}
	st.Plan.Blocks[0].Target = 0
	st.Plan.Blocks[0].Reason = "version mismatch at execution"
	st.Catalog[0].LocalAssessment = &LocalAssessment{Status: "invalid_local_file"}
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	if r.parts["plan"] != before["plan"] || r.parts["analysis"] == before["analysis"] {
		t.Fatal("definition and parser cache not independent")
	}
	loaded, err := newStateRepository(r.path).Load(State{})
	if err != nil || loaded.Plan.Elapsed != 17 || loaded.Plan.Blocks[0].Target != 0 || loaded.Plan.Blocks[0].Scenario.LocalAssessment.Requirements == nil || loaded.Catalog[0].LocalAssessment.Status != "invalid_local_file" {
		t.Fatal("fixed scenario/progress overlay", err)
	}
}
func TestFailedManifestCommitKeepsPreviousGenerationReadable(t *testing.T) {
	r, st := repositoryFixture(t)
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(r.path)
	if err := os.Mkdir(r.path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	st.Plan.Elapsed = 42
	if err := r.SaveExecution(st); err == nil {
		t.Fatal("failed manifest commit hidden")
	}
	after, _ := os.ReadFile(r.path)
	if !bytes.Equal(old, after) {
		t.Fatal("manifest was partially overwritten")
	}
	loaded, err := newStateRepository(r.path).Load(State{})
	if err != nil || loaded.Plan.Elapsed != 0 {
		t.Fatal("previous generation lost", err)
	}
}
func TestCorruptionAndUntrustedPartPathsAreRejected(t *testing.T) {
	r, st := repositoryFixture(t)
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	ref := r.parts["analysis"]
	path := filepath.Join(r.dir(), ref.File)
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := newStateRepository(r.path).Load(State{}); err == nil || !strings.Contains(err.Error(), "校验失败") {
		t.Fatal("corrupt cache accepted", err)
	}
	m := stateManifest{Version: 2, StorageVersion: 2, Parts: r.parts}
	m.Parts["analysis"] = statePart{File: "../../private.json", SHA256: ref.SHA256}
	raw, _ := json.Marshal(m)
	_ = os.WriteFile(r.path, raw, 0600)
	if _, err := newStateRepository(r.path).Load(State{}); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatal("untrusted path accepted", err)
	}
}
func TestCapturedDefinitionOwnsNestedSnapshots(t *testing.T) {
	r, st := repositoryFixture(t)
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	old := r.parts["plan"]
	st.Plan.Blocks[0].Scenario.Mechanics.Tags[0] = "accidental mutation"
	if err := r.Save(st); err != nil {
		t.Fatal(err)
	}
	if r.parts["plan"] != old {
		t.Fatal("fixed definition shares live nested pointers")
	}
}
