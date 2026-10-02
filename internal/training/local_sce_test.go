package training

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const localSCEFixture = `Name=Static A
GameVersion=fixture
Description=Static clicking scenario. Two close, small targets.
Timelimit=60
PlayerProfile=player.v1
AddedBots=target.bot
ScoreMultAccuracy=true
[Character Profile]
Name=player.v1
WeaponProfileNames=gun
[Weapon Profile]
Name=gun
MagazineMax=3
AmmoPerShot=1
AmmoReloadedOnKill=3
ReloadTimeFromEmpty=0.5
[Bot Profile]
Name=target
CharacterProfile=target
[Character Profile]
Name=target
MainBBRadius=10
MaxSpeed=0
AbilityProfileNames=approach.abilmov
[Movement Ability Profile]
Name=approach
Velocity=3500
[Bot Profile]
Name=unused
CharacterProfile=unused
[Character Profile]
Name=unused
MainBBRadius=9999
[Map Data]
Name=this is map text not a root config
`

func TestLocalSCEPreservesActiveFactsAndUnknownDifficulty(t *testing.T) {
	name, m, a, err := ParseLocalSCE([]byte(localSCEFixture))
	if err != nil {
		t.Fatal(err)
	}
	if name != "Static A" || m.DeclaredSeconds != 60 || m.AngularSize != nil || m.Role != "unknown" || a.Status != "file_parsed_model_unfitted" {
		t.Fatal("invalid assessment")
	}
	for _, tag := range []string{"short_transfer", "precision", "micro_adjustment", "reload_constraint", "accuracy_constraint"} {
		if !hasDemand(Scenario{Mechanics: m}, tag) {
			t.Fatal("missing tag", tag)
		}
	}
	velocity := false
	for _, f := range a.Fields {
		if f.Profile == "unused" {
			t.Fatal("unused profile imported")
		}
		if f.Key == "Velocity" && f.Raw == "3500" {
			velocity = true
		}
		if f.Line < 1 {
			t.Fatal("source line absent")
		}
	}
	if !velocity {
		t.Fatal("ability omitted despite zero MaxSpeed")
	}
	for _, issue := range a.Issues {
		if strings.HasPrefix(issue, "unresolved:") {
			t.Fatal("dotted player profile mishandled", issue)
		}
	}
	if _, _, _, err = ParseLocalSCE([]byte("Name=A\nName=B\n")); err == nil {
		t.Fatal("ambiguous identity")
	}
	if _, _, _, err = ParseLocalSCE([]byte{'N', 'a', 'm', 'e', '=', 0xff}); err == nil {
		t.Fatal("binary cache accepted")
	}
}

func TestPassiveLocalAssessmentWaitsForStableFileAndKeepsPlanFixed(t *testing.T) {
	s, _ := templateFixture(t)
	plan, err := s.Generate(defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := plan.Blocks[0].Scenario.Mechanics
	root := t.TempDir()
	path := filepath.Join(root, "download.sce")
	_ = os.WriteFile(path, []byte(localSCEFixture), 0600)
	_ = os.Chtimes(path, epoch.Add(-time.Minute), epoch.Add(-time.Minute))
	s.PollLocal([]string{root}, "", epoch)
	if s.state.Catalog[0].LocalAssessment != nil {
		t.Fatal("partial first observation assessed")
	}
	s.PollLocal([]string{root}, "", epoch.Add(15*time.Second))
	if s.state.Catalog[0].LocalAssessment == nil || !hasDemand(s.state.Catalog[0], "short_transfer") {
		t.Fatal("stable file not assessed")
	}
	if !reflect.DeepEqual(before, s.state.Plan.Blocks[0].Scenario.Mechanics) {
		t.Fatal("live plan rewritten")
	}
	first := s.state.Catalog[0].LocalAssessment.FileSHA256
	updated := strings.Replace(localSCEFixture, "MainBBRadius=10", "MainBBRadius=8", 1)
	_ = os.WriteFile(path, []byte(updated), 0600)
	_ = os.Chtimes(path, epoch.Add(time.Minute), epoch.Add(time.Minute))
	s.PollLocal([]string{root}, "", epoch.Add(2*time.Minute))
	s.PollLocal([]string{root}, "", epoch.Add(3*time.Minute))
	if first == s.state.Catalog[0].LocalAssessment.FileSHA256 {
		t.Fatal("content change not invalidated")
	}
	reopened, err := New(filepath.Dir(s.path))
	if err != nil || reopened.state.Catalog[0].LocalAssessment == nil {
		t.Fatal("assessment cache not persisted", err)
	}
}

// Run against the uploaded archive when available without adding raw user files
// to the repo. CI covers synthetic edge cases independently.
func TestUploadedSCEArchiveMatchesMechanismSnapshot(t *testing.T) {
	path := os.Getenv("REFLEKS_TEST_SCE_ARCHIVE")
	if path == "" {
		t.Skip("uploaded archive not available")
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	count := 0
	for _, f := range z.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".sce") {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		_ = r.Close()
		if e != nil {
			t.Fatal(e)
		}
		name, m, a, e := ParseLocalSCE(b)
		if e != nil {
			t.Fatal(f.Name, e)
		}
		expected := snapshotMechanics[strings.ToLower(name)]
		if expected == nil {
			t.Fatal("missing snapshot", name)
		}
		if m.FileSHA256 != expected.FileSHA256 || !reflect.DeepEqual(m.Tags, expected.Tags) || m.DeclaredSeconds != expected.DeclaredSeconds {
			t.Fatalf("snapshot mismatch %s: tags %v vs %v", name, m.Tags, expected.Tags)
		}
		if a.Status != "file_parsed_model_unfitted" {
			t.Fatal("invented fitted difficulty")
		}
		count++
	}
	if count != 58 {
		t.Fatalf("incomplete archive: %d", count)
	}
}

func TestSameNameDifferentLocalVersionsRemainUnknownAndRecover(t *testing.T) {
	s, _ := templateFixture(t)
	root := t.TempDir()
	a := filepath.Join(root, "A.sce")
	b := filepath.Join(root, "B.sce")
	_ = os.WriteFile(a, []byte(localSCEFixture), 0600)
	_ = os.WriteFile(b, []byte(strings.Replace(localSCEFixture, "MainBBRadius=10", "MainBBRadius=20", 1)), 0600)
	for _, p := range []string{a, b} {
		_ = os.Chtimes(p, epoch.Add(-time.Minute), epoch.Add(-time.Minute))
	}
	s.PollLocal([]string{root}, "", epoch)
	s.PollLocal([]string{root}, "", epoch.Add(15*time.Second))
	if s.state.Catalog[0].LocalAssessment.Status != "ambiguous_local_versions" || s.state.Catalog[0].Mechanics != nil {
		t.Fatal("conflicting revisions silently selected")
	}
	_ = os.Remove(b)
	s.PollLocal([]string{root}, "", epoch.Add(30*time.Second))
	if s.state.Catalog[0].LocalAssessment.Status != "file_parsed_model_unfitted" {
		t.Fatal("conflict did not recover")
	}
	_ = os.WriteFile(a, []byte("partial"), 0600)
	_ = os.Chtimes(a, epoch.Add(time.Minute), epoch.Add(time.Minute))
	s.PollLocal([]string{root}, "", epoch.Add(2*time.Minute))
	if s.state.Catalog[0].Mechanics != nil {
		t.Fatal("stale mechanism reused during download")
	}
	s.PollLocal([]string{root}, "", epoch.Add(3*time.Minute))
	if s.state.Catalog[0].LocalAssessment.Status != "invalid_local_file" {
		t.Fatal("bad content silently accepted")
	}
}
