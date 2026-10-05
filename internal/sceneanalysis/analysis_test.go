package sceneanalysis

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

const fixture = `Name=A
AimTypeTag=Clicking
Timelimit=60
GameVersion=fixture
PlayerProfile=player
AddedBots=target.bot;target.bot
InvincibleBots=false
IsTimeDilationActive=false
IsTargetSizeActive=false
Timescale=1
MapScale=1
TimeDilationBaseMultiplier=1
TargetSizeBaseMultiplier=1
[Character Profile]
Name=player
WeaponProfileNames=gun.wpn
[Weapon Profile]
Name=gun
Type=Hitscan
ShotsPerClick=1
IsBurstWeapon=false
IsChargeWeapon=false
AlsoShoot=
DamagePerShot=100
TimeBetweenShots=0.1
MagazineMax=3
AmmoPerShot=1
AmmoReloadedOnKill=4
[Bot Profile]
Name=target
DisableScoring=false
NoDodging=false
DodgeProfileNames=dodge.dodge;dodge.dodge
CharacterProfile=target
[Character Profile]
Name=target
MainBBType=Sphere
MainBBRadius=10
MainBBHeight=10
MaxSpeed=100
Acceleration=500
Gravity=0
MaxHealth=100
HealthRegenPerSec=0
HealthRegenDelay=0
InvincibleBots=false
HeadshotOnly=false
MeshHitDetection=false
AbilityProfileNames=
[Dodge Profile]
Name=dodge
ToggleLeftRight=true
MinLRTimeChange=0.1
MaxLRTimeChange=0.2
ToggleForwardBack=false
[Map Data]
{"objects":[{"type":"gameObject","name":"SpawnPoint","location":"1, 2, 3","properties":[{"name":"TeamMask","value":2},{"name":"PermittedCharacterProfiles","value":"target; other,target"}]}]}
`

func parse(t *testing.T, s string) *LocalAssessment {
	t.Helper()
	_, _, a, err := ParseLocalSCE([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestBoundedBinaryContainerAndRawIdentity(t *testing.T) {
	body := []byte(fixture)
	raw := make([]byte, 12)
	binary.LittleEndian.PutUint32(raw, 0xf55bace6)
	binary.LittleEndian.PutUint32(raw[4:], 1)
	binary.LittleEndian.PutUint32(raw[8:], uint32(len(body)+1))
	raw = append(append(raw, body...), 0)
	raw = append(raw, 255, 254, 0)
	name, _, a, err := ParseLocalSCE(raw)
	if err != nil || name != "A" {
		t.Fatal(name, err)
	}
	plain := parse(t, fixture)
	if a.FileSHA256 == plain.FileSHA256 || a.BodySHA256 != plain.BodySHA256 || a.MapDataSHA256 != plain.MapDataSHA256 || a.Container.TrailerBytes != 3 {
		t.Fatal("raw/body/map identities conflated")
	}
	for _, index := range []int{4, 8} {
		bad := append([]byte(nil), raw...)
		binary.LittleEndian.PutUint32(bad[index:], 0xffffffff)
		if _, _, _, err := ParseLocalSCE(bad); err == nil {
			t.Fatal("unsupported container accepted")
		}
	}
	if _, _, _, err := ParseLocalSCE(raw[:11]); err == nil {
		t.Fatal("truncated header accepted")
	}
}
func TestExactReferencesHelpersAndPhaseBounds(t *testing.T) {
	s := strings.ReplaceAll(fixture, "CharacterProfile=target", "CharacterProfile= target")
	s = strings.ReplaceAll(s, "Name=target\nMainBBType", "Name= target\nMainBBType")
	s = strings.Replace(s, "AddedBots=target.bot;target.bot", "AddedBots=rotation.rot;target.bot", 1)
	s = strings.Replace(s, "[Map Data]", `[Bot Profile]
Name=helper
DisableScoring=true
NoDodging=true
CharacterProfile=helper
[Character Profile]
Name=helper
MainBBRadius=0.1
MaxSpeed=3000
[Bot Rotation Profile]
Name=rotation
ProfileNames=target.bot;helper.bot;target.bot
[Map Data]`, 1)
	d := parse(t, s).Requirements
	if len(d.Targets) != 1 || d.Targets[0].Character != " target" || len(d.Helpers) != 1 || *d.ScoringMin != 1 || *d.ScoringMax != 2 || len(d.Slots[0].Candidates) != 3 || len(d.Targets[0].DodgeEntries) != 2 {
		t.Fatal("roles, spacing or multiplicity lost", d)
	}
	precision := factByKey(d.Features, "precision_radius")
	if precision.Value == nil || math.Abs(*precision.Value+math.Log(10)) > 1e-9 {
		t.Fatal("helper included in precision")
	}
	if factByKey(d.Features, "constant_scoring_slot_scarcity").Value != nil {
		t.Fatal("phase bounds collapsed to constant count")
	}
}
func TestMissingGateDiffersFromExplicitClosedGate(t *testing.T) {
	missing := parse(t, strings.Replace(fixture, "NoDodging=false\n", "", 1)).Requirements
	closed := parse(t, strings.Replace(fixture, "NoDodging=false", "NoDodging=true", 1)).Requirements
	if factByKey(missing.Features, "speed_width_pressure").Value != nil {
		t.Fatal("missing gate defaulted")
	}
	v := factByKey(closed.Features, "speed_width_pressure").Value
	if v == nil || *v != 0 {
		t.Fatal("closed Dodge gate ignored")
	}
	if factByKey(closed.Targets[0].Facts, "MaxSpeed").Value == nil {
		t.Fatal("declared cap discarded")
	}
}
func TestConditionalWindowsRemainWithoutMap(t *testing.T) {
	s := strings.Split(fixture, "[Map Data]")[0]
	d := parse(t, s).Requirements
	if d.Map != nil || factByKey(d.Targets[0].Windows, "ideal_body_hits_to_kill").Value == nil {
		t.Fatal("weapon windows depend on geometry")
	}
	rate := factByKey(d.Targets[0].Windows, "full_refund_reload_at_q50")
	if rate.Value == nil || math.Abs(*rate.Value-71.42857142857143) > 1e-7 {
		t.Fatal("reload curve", rate)
	}
	bad := parse(t, strings.Replace(s, "HeadshotOnly=false", "HeadshotOnly=true", 1)).Requirements
	if factByKey(bad.Targets[0].Windows, "ideal_body_hits_to_kill").Value != nil {
		t.Fatal("head-only treated as body hit")
	}
	decay := parse(t, strings.Replace(strings.Replace(s, "MaxHealth=100", "MaxHealth=1900", 1), "HealthRegenPerSec=0", "HealthRegenPerSec=-100", 1)).Requirements
	f := factByKey(decay.Targets[0].Windows, "self_decay_seconds")
	if f.Value == nil || *f.Value != 19 || f.Status != "calculated_under_explicit_conditions" {
		t.Fatal("self decay", f)
	}
}

func TestLegacyWeaponGatesAreConditionalAndExplicitEnabledGateBlocksWindow(t *testing.T) {
	legacy := strings.Replace(strings.Replace(fixture, "IsChargeWeapon=false\n", "", 1), "MeshHitDetection=false\n", "", 1)
	d := parse(t, legacy).Requirements
	f := factByKey(d.Targets[0].Windows, "full_refund_reload_at_q50")
	if f.Value == nil || len(f.Unknown) == 0 || f.Status != "calculated_under_explicit_conditions" {
		t.Fatal("legacy gates silently defaulted", f)
	}
	charged := parse(t, strings.Replace(fixture, "IsChargeWeapon=false", "IsChargeWeapon=true", 1)).Requirements
	if factByKey(charged.Targets[0].Windows, "ideal_body_hits_to_kill").Value != nil {
		t.Fatal("known active charge mechanism ignored")
	}
}
func TestFixedBasisComparisonAbstainsAndIsSymmetric(t *testing.T) {
	a := parse(t, fixture).Requirements
	b := parse(t, strings.ReplaceAll(strings.Replace(fixture, "Name=A", "Name=B", 1), "MaxSpeed=100", "MaxSpeed=150")).Requirements
	c := parse(t, strings.ReplaceAll(strings.Replace(fixture, "Name=A", "Name=C", 1), "MaxSpeed=100", "MaxSpeed=200")).Requirements
	ab, ba, bc := Compare(a, b, nil), Compare(b, a, nil), Compare(b, c, nil)
	if ab.Kind != "estimated_direction" || ab.Direction != "higher_declared_demand" || ba.Direction != "lower_declared_demand" || bc.Direction != ab.Direction || *ab.NeighborDistance != *ba.NeighborDistance || ab.RankMargin != nil {
		t.Fatal("comparison", ab, ba, bc)
	}
	for i := range ab.Axes {
		if ab.Axes[i].Delta != nil && math.Abs(*ab.Axes[i].Delta+*ba.Axes[i].Delta) > 1e-10 {
			t.Fatal("non-antisymmetric contrast")
		}
	}
	partial := parse(t, strings.Replace(fixture, "Acceleration=500\n", "", 1)).Requirements
	if Compare(a, partial, nil).NeighborDistance != nil {
		t.Fatal("partial basis used in selection")
	}
	interval := parse(t, strings.Replace(fixture, "MinLRTimeChange=0.1", "MinLRTimeChange=0.05", 1)).Requirements
	if Compare(a, interval, nil).Kind != "partial_axes" {
		t.Fatal("shorter interval automatically harder")
	}
	score := parse(t, strings.Replace(fixture, "Timelimit=60", "Timelimit=60\nScorePerKill=1000", 1)).Requirements
	for i, f := range a.Features {
		other := score.Features[i]
		if f.Value != nil && other.Value != nil && *f.Value != *other.Value {
			t.Fatal("score multiplier altered demands")
		}
	}
}
func TestNativeOrderRequiresVerifiedVersionBinding(t *testing.T) {
	a := parse(t, fixture).Requirements
	b := parse(t, strings.Replace(fixture, "Name=A", "Name=B", 1)).Requirements
	n := &NativeOrder{Benchmark: "one", Category: "clicking", Family: "same", Version: "v1", Source: "fixture", IndexA: 0, IndexB: 1, HashA: a.FileSHA256, HashB: b.FileSHA256}
	if Compare(a, b, n).Kind == "native_family_order" {
		t.Fatal("name candidate promoted")
	}
	n.Verified = true
	if Compare(a, b, n).Kind != "native_family_order" {
		t.Fatal("verified native scope lost")
	}
	n.HashB = "stale"
	if Compare(a, b, n).Kind == "native_family_order" {
		t.Fatal("stale native binding")
	}
}
func TestFullReflexEntitiesAndVolumeFilters(t *testing.T) {
	raw := []byte("[Map Data]\nreflex map version 8\n entity\n  type PlayerSpawn\n  Vector3 position 1 2 3\n  Bool teamA\n entity\n  type PlayerSpawn\n  Vector3 position 1 2 3\n  Bool teamB\n")
	m := ParseMap(raw)
	if m.Format != "reflex_v8" || len(m.Objects) != 2 {
		t.Fatal("indent entity parser", m)
	}
	if len(SpawnCandidates(m, "target", 2, false).Objects) != 1 {
		t.Fatal("Reflex raw pool")
	}
	d := parse(t, strings.ReplaceAll(fixture, "SpawnPoint", "SpawnVolume")).Requirements
	p := SpawnCandidates(d.Map, "target", 2, false)
	if len(p.Objects) != 1 || p.Status != "calculated_under_explicit_conditions" || d.Map.Counts["SpawnVolume"] != 1 {
		t.Fatal("comma and semicolon filters")
	}
}
func TestCoupledMotionShortDwellCanReduceExcursion(t *testing.T) {
	a, _ := PeriodicMotion(100, 100, .1)
	b, _ := PeriodicMotion(100, 100, 1)
	if !(a.CenterExcursion < b.CenterExcursion && a.RMSSpeed < b.RMSSpeed) {
		t.Fatal("standalone direction frequency model")
	}
	for _, q := range []float64{0, .5, .75, .9, 1} {
		rate, ok := FullRefundReloadRate(3, 1, q)
		if !ok || rate < 0 || rate > 1 {
			t.Fatal("ammo rate")
		}
	}
	if _, ok := FullRefundReloadRate(math.NaN(), 1, .5); ok {
		t.Fatal("NaN accepted")
	}
}
func TestUploadedWorkshopFixture(t *testing.T) {
	path := os.Getenv("AIMMEOW_TEST_BINARY_SCE")
	if path == "" {
		t.Skip("optional user fixture")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	name, _, a, err := ParseLocalSCE(raw)
	if err != nil || name != "1w3ts Pasu Perfected Micro Goated" || a.Container.Format != "workshop_binary_v1" || a.Container.TrailerBytes != 76 {
		t.Fatal(name, err)
	}
}

func TestShadowModelUsesFixedCompleteBasisAndDoesNotEnterPlanner(t *testing.T) {
	a := parse(t, fixture).Requirements
	b := parse(t, strings.Replace(fixture, "MaxSpeed=100", "MaxSpeed=150", 1)).Requirements
	c := parse(t, strings.Replace(fixture, "MaxSpeed=100", "MaxSpeed=200", 1)).Requirements
	m := RankModel{Version: "synthetic-test", Status: "shadow_only", Task: "clicking", Basis: FixedBasis("clicking"), Weights: []float64{1, 1, 1, 1}, TrainPairRMS: []float64{1, 1, 1, 1}, TrainingEvidence: "synthetic; no production calibration"}
	ab, e := ShadowMargin(a, b, m)
	if e != nil || ab.Margin == nil {
		t.Fatal(ab, e)
	}
	ba, _ := ShadowMargin(b, a, m)
	bc, _ := ShadowMargin(b, c, m)
	ac, _ := ShadowMargin(a, c, m)
	if math.Abs(*ab.Margin+*ba.Margin) > 1e-10 || math.Abs(*ab.Margin+*bc.Margin-*ac.Margin) > 1e-10 || Compare(a, b, nil).RankMargin != nil {
		t.Fatal("partial/cyclic ranking or shadow entered runtime")
	}
	missing := parse(t, strings.Replace(fixture, "Acceleration=500\n", "", 1)).Requirements
	if estimate, _ := ShadowMargin(a, missing, m); estimate.Margin != nil {
		t.Fatal("missing term silently dropped")
	}
	m.Basis = m.Basis[:2]
	if _, e = ShadowMargin(a, b, m); e == nil {
		t.Fatal("variable feature basis accepted")
	}
}
