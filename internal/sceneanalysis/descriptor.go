package sceneanalysis

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
)

const DescriptorSchema = 2

// Fact distinguishes absent declarations from an explicit zero. A derived value
// is meaningful only under its listed conditions, never a gameplay measurement.
type Fact struct {
	Key        string     `json:"key"`
	Value      *float64   `json:"value"`
	Text       string     `json:"text,omitempty"`
	Unit       string     `json:"unit"`
	Status     string     `json:"status"`
	Sources    []SCEField `json:"sources,omitempty"`
	Conditions []string   `json:"conditions,omitempty"`
	Unknown    []string   `json:"unknown,omitempty"`
}
type RequirementAxis struct {
	Key   string `json:"key"`
	Facts []Fact `json:"facts"`
}
type Definition struct {
	Kind   string     `json:"kind"`
	Name   string     `json:"name"`
	Fields []SCEField `json:"fields"`
}
type Slot struct {
	Reference  string   `json:"reference"`
	Candidates []string `json:"candidates"`
	Complete   bool     `json:"complete"`
	ScoringMin *int     `json:"scoringMin"`
	ScoringMax *int     `json:"scoringMax"`
}
type Target struct {
	Bot          string        `json:"bot"`
	Character    string        `json:"character"`
	Scoring      string        `json:"scoring"`
	DodgeGate    string        `json:"dodgeGate"`
	Shape        string        `json:"shape"`
	Facts        []Fact        `json:"facts"`
	DodgeEntries []Definition  `json:"dodgeEntries,omitempty"`
	Abilities    []Definition  `json:"abilities,omitempty"`
	Windows      []Fact        `json:"windows,omitempty"`
	MotionModels []DodgeMotion `json:"motionModels,omitempty"`
}
type Descriptor struct {
	Schema                int               `json:"schema"`
	SemanticVersion       string            `json:"semanticVersion"`
	FileSHA256            string            `json:"fileSHA256"`
	BodySHA256            string            `json:"bodySHA256"`
	ContainerSignature    string            `json:"containerSignature"`
	MapSHA256             string            `json:"mapSHA256,omitempty"`
	GameVersion           string            `json:"gameVersion,omitempty"`
	DeclaredTask          string            `json:"declaredTask"`
	Task                  string            `json:"task"`
	Definitions           []Definition      `json:"definitions,omitempty"`
	Targets               []Target          `json:"targets"`
	Helpers               []Target          `json:"helpers"`
	Slots                 []Slot            `json:"slots"`
	ScoringMin            *int              `json:"scoringMin"`
	ScoringMax            *int              `json:"scoringMax"`
	Axes                  []RequirementAxis `json:"axes"`
	Features              []Fact            `json:"features"`
	Map                   *MapDescriptor    `json:"map,omitempty"`
	ControlledFingerprint string            `json:"controlledFingerprint,omitempty"`
	Unknown               []string          `json:"unknown"`
	Hazards               []HazardExposure  `json:"hazards,omitempty"`
	Influences            []InfluenceEdge   `json:"influences,omitempty"`
}

func num(v float64) *float64 { return &v }
func integer(v int) *int     { return &v }
func source(s sceSection, key string) []SCEField {
	out := []SCEField{}
	for _, f := range s.fields {
		if f.Key == key {
			f.Profile = s.value("Name")
			if s.kind == "Root" {
				f.Profile = ""
			}
			out = append(out, f)
		}
	}
	return out
}
func numberFact(s sceSection, key, unit string) Fact {
	f := Fact{Key: key, Unit: unit, Status: "unknown", Sources: source(s, key)}
	if len(f.Sources) != 1 {
		f.Unknown = []string{"missing_or_duplicate_declaration"}
		return f
	}
	if v, ok := ConfigNumber(f.Sources[0].Raw); ok {
		f.Value = num(v)
		f.Status = "declared"
	} else {
		f.Unknown = []string{"invalid_number"}
	}
	return f
}
func textFact(s sceSection, key string) Fact {
	f := Fact{Key: key, Unit: "declaration", Status: "unknown", Sources: source(s, key)}
	if len(f.Sources) == 1 {
		f.Text = f.Sources[0].Raw
		f.Status = "declared"
	} else {
		f.Unknown = []string{"missing_or_duplicate_declaration"}
	}
	return f
}
func derived(key, unit string, v *float64, sources []SCEField, conditions ...string) Fact {
	if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
		v = nil
	}
	f := Fact{Key: key, Unit: unit, Value: v, Status: "calculated_under_explicit_conditions", Sources: sources, Conditions: conditions}
	if v == nil {
		f.Status = "unknown"
		f.Unknown = []string{"incomplete_or_unsupported_basis"}
	}
	return f
}
func tokens(raw string) []string {
	var out []string
	for _, s := range strings.Split(raw, ";") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
func definition(s sceSection) Definition {
	f := append([]SCEField{}, s.fields...)
	for i := range f {
		f[i].Profile = s.value("Name")
	}
	return Definition{s.kind, s.value("Name"), f}
}
func factByKey(fs []Fact, key string) Fact {
	for _, f := range fs {
		if f.Key == key {
			return f
		}
	}
	return Fact{Key: key, Status: "unknown"}
}
func positive(f Fact) bool    { return f.Value != nil && *f.Value > 0 }
func nonnegative(f Fact) bool { return f.Value != nil && *f.Value >= 0 }
func task(tag string) string {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "clicking":
		return "clicking"
	case "tracking":
		return "tracking"
	case "switching", "target switching", "targetswitching":
		return "switching"
	}
	return "unknown"
}

func describe(body []byte, ss []sceSection, a *LocalAssessment) *Descriptor {
	root := ss[0]
	d := &Descriptor{Schema: DescriptorSchema, SemanticVersion: SemanticVersion, FileSHA256: a.FileSHA256, BodySHA256: a.BodySHA256, GameVersion: a.GameVersion, DeclaredTask: root.value("AimTypeTag"), Task: task(root.value("AimTypeTag")), Targets: []Target{}, Helpers: []Target{}, Slots: []Slot{}, Unknown: []string{"no_synchronous_target_and_aim_trajectory", "geometry_semantics_not_engine_verified"}}
	d.ContainerSignature = containerSignature(a.Container)
	for _, s := range ss {
		d.Definitions = append(d.Definitions, definition(s))
	}
	botIDs := map[int]bool{}
	remaining := 100000
	var expand func(string, map[int]bool) ([]string, bool)
	expand = func(ref string, path map[int]bool) ([]string, bool) {
		if remaining <= 0 || len(path) > 64 {
			d.Unknown = append(d.Unknown, "phase_expansion_limit")
			return nil, false
		}
		kind := "Bot Profile"
		if strings.HasSuffix(ref, ".rot") {
			kind = "Bot Rotation Profile"
		}
		i, ok := resolveIndex(ss, kind, ref)
		if !ok || path[i] {
			d.Unknown = append(d.Unknown, "unresolved_or_cyclic_slot:"+ref)
			return nil, false
		}
		if kind == "Bot Profile" {
			botIDs[i] = true
			remaining--
			return []string{ss[i].value("Name")}, true
		}
		next := map[int]bool{}
		for k, v := range path {
			next[k] = v
		}
		next[i] = true
		out := []string{}
		complete := true
		for _, name := range tokens(ss[i].value("ProfileNames")) {
			values, ok := expand(name, next)
			out = append(out, values...)
			complete = complete && ok
		}
		return out, complete && len(out) > 0
	}
	for _, ref := range tokens(root.value("AddedBots")) {
		names, complete := expand(ref, map[int]bool{})
		d.Slots = append(d.Slots, Slot{Reference: ref, Candidates: names, Complete: complete})
	}
	ids := []int{}
	for i := range botIDs {
		ids = append(ids, i)
	}
	sort.Ints(ids)
	eligibility := map[string]string{}
	scoredCharacters := map[int]bool{}
	for _, i := range ids {
		b := ss[i]
		ci, ok := resolveIndex(ss, "Character Profile", b.value("CharacterProfile"))
		c := sceSection{kind: "Character Profile"}
		if ok {
			c = ss[ci]
		}
		t := Target{Bot: b.value("Name"), Character: c.value("Name"), Scoring: "legacy_unknown", DodgeGate: "unknown", Shape: c.value("MainBBType")}
		switch b.value("DisableScoring") {
		case "true":
			t.Scoring = "disabled"
		case "false":
			t.Scoring = "explicit_enabled"
		}
		eligibility[t.Bot] = t.Scoring
		for _, key := range []string{"DisableScoring", "NoDodging", "NoAiming", "DodgeProfileNames", "DodgeProfileWeights"} {
			t.Facts = append(t.Facts, textFact(b, key))
		}
		for _, key := range []string{"MainBBRadius", "MainBBHeight", "MaxSpeed", "Acceleration", "MaxHealth", "HealthRegenPerSec", "HealthRegenDelay", "MinRespawnDelay", "MaxRespawnDelay", "Gravity", "SpawnOffsetMin", "SpawnOffsetMax"} {
			unit := "configuration units"
			if strings.Contains(key, "Delay") {
				unit = "configured seconds"
			}
			if strings.HasPrefix(key, "SpawnOffset") {
				t.Facts = append(t.Facts, textFact(c, key))
			} else {
				t.Facts = append(t.Facts, numberFact(c, key, unit))
			}
		}
		refs := tokens(b.value("DodgeProfileNames"))
		resolved := true
		for _, ref := range refs {
			di, found := resolveIndex(ss, "Dodge Profile", ref)
			if found {
				t.DodgeEntries = append(t.DodgeEntries, definition(ss[di]))
			} else {
				resolved = false
			}
		}
		for _, entry := range t.DodgeEntries {
			section := sceSection{kind: entry.Kind, fields: entry.Fields}
			for _, key := range []string{"MinLRTimeChange", "MaxLRTimeChange", "MinFBTimeChange", "MaxFBTimeChange"} {
				t.Facts = append(t.Facts, numberFact(section, key, "configured seconds"))
			}
			for _, key := range []string{"ToggleLeftRight", "ToggleForwardBack"} {
				t.Facts = append(t.Facts, textFact(section, key))
			}
		}
		if b.value("NoDodging") == "true" || b.value("NoDodging") == "false" && len(refs) == 0 && len(source(b, "DodgeProfileNames")) == 1 {
			t.DodgeGate = "off"
		} else if b.value("NoDodging") == "false" && len(refs) > 0 && resolved {
			t.DodgeGate = "candidate_on"
		}
		t.MotionModels = dodgeMotion(t)
		for _, ref := range tokens(c.value("AbilityProfileNames")) {
			kind := ""
			switch {
			case strings.HasSuffix(ref, ".abilmov"):
				kind = "Movement Ability Profile"
			case strings.HasSuffix(ref, ".abilwep"):
				kind = "Weapon Ability Profile"
			case strings.HasSuffix(ref, ".abilmelee"):
				kind = "Melee Ability Profile"
			}
			ai, found := resolveIndex(ss, kind, ref)
			if found {
				t.Abilities = append(t.Abilities, definition(ss[ai]))
			} else {
				d.Unknown = append(d.Unknown, "unresolved_ability:"+ref)
			}
		}
		h := factByKey(t.Facts, "MaxHealth")
		regen := factByKey(t.Facts, "HealthRegenPerSec")
		delay := factByKey(t.Facts, "HealthRegenDelay")
		var decay *float64
		if positive(h) && regen.Value != nil && *regen.Value < 0 && delay.Value != nil && *delay.Value == 0 {
			decay = num(*h.Value / -*regen.Value)
		}
		t.Windows = append(t.Windows, derived("self_decay_seconds", "configured seconds", decay, append(append(h.Sources, regen.Sources...), delay.Sources...), "No external damage, healing or invincibility; linear negative regeneration starts immediately; this is not an observed phase duration"))
		t.Windows = append(t.Windows, weaponWindows(root, c, ss)...)
		if t.Scoring == "disabled" {
			d.Helpers = append(d.Helpers, t)
		} else {
			d.Targets = append(d.Targets, t)
			if ok {
				scoredCharacters[ci] = true
			}
		}
	}
	low, high := 0, 0
	complete := len(d.Slots) > 0
	for i := range d.Slots {
		s := &d.Slots[i]
		lo, hi := 1, 0
		for _, name := range s.Candidates {
			e := eligibility[name]
			if e != "explicit_enabled" {
				lo = 0
			}
			if e != "disabled" {
				hi = 1
			}
			if e == "legacy_unknown" {
				d.Unknown = append(d.Unknown, "legacy_scoring_eligibility")
			}
		}
		if s.Complete {
			s.ScoringMin = integer(lo)
			s.ScoringMax = integer(hi)
			low += lo
			high += hi
		} else {
			complete = false
		}
	}
	if complete {
		d.ScoringMin = integer(low)
		d.ScoringMax = integer(high)
	}
	d.Map = ParseMap(body)
	if d.Map != nil {
		d.MapSHA256 = d.Map.SHA256
	}
	d.Hazards = hazardExposures(d)
	d.Influences = referenceGraph(ss, d)
	d.Features = features(d)
	d.Axes = []RequirementAxis{
		{Key: "precision", Facts: []Fact{}}, {Key: "motion_and_direction_changes", Facts: []Fact{}}, {Key: "spatial_selection_and_transfer", Facts: []Fact{}}, {Key: "kill_and_lifetime_windows", Facts: []Fact{}}, {Key: "error_cost_and_score_strategy", Facts: []Fact{}}, {Key: "phase_and_external_mechanisms", Facts: []Fact{}},
	}
	for _, t := range d.Targets {
		for _, f := range t.Facts {
			axis := 5
			switch f.Key {
			case "MainBBRadius", "MainBBHeight":
				axis = 0
			case "MaxSpeed", "Acceleration", "NoDodging", "Gravity", "DodgeProfileWeights", "MinLRTimeChange", "MaxLRTimeChange", "MinFBTimeChange", "MaxFBTimeChange", "ToggleLeftRight", "ToggleForwardBack":
				axis = 1
			case "MinRespawnDelay", "MaxRespawnDelay", "SpawnOffsetMin", "SpawnOffsetMax":
				axis = 2
			case "MaxHealth", "HealthRegenPerSec", "HealthRegenDelay":
				axis = 3
			}
			d.Axes[axis].Facts = append(d.Axes[axis].Facts, f)
		}
		d.Axes[3].Facts = append(d.Axes[3].Facts, t.Windows...)
	}
	for _, key := range []string{"Timescale", "MapScale", "TimeDilationBaseMultiplier", "TargetSizeBaseMultiplier"} {
		d.Axes[5].Facts = append(d.Axes[5].Facts, numberFact(root, key, "configuration multiplier"))
	}
	for _, key := range []string{"IsTimeDilationActive", "IsTargetSizeActive", "BotMaxLives", "BotTeams"} {
		d.Axes[5].Facts = append(d.Axes[5].Facts, textFact(root, key))
	}
	for _, f := range root.fields {
		if strings.HasPrefix(f.Key, "Score") || f.Key == "MultSqrtAcc" || f.Key == "MBSEnable" || f.Key == "EnableOverDamage" {
			d.Axes[4].Facts = append(d.Axes[4].Facts, textFact(root, f.Key))
		}
	}
	if pi, ok := resolveIndex(ss, "Character Profile", root.value("PlayerProfile")); ok {
		for _, ref := range tokens(ss[pi].value("WeaponProfileNames")) {
			if wi, ok := resolveIndex(ss, "Weapon Profile", ref); ok {
				for _, key := range []string{"MagazineMax", "AmmoPerShot", "AmmoReloadedOnKill", "ReloadTimeFromEmpty", "TimeBetweenShots", "DamagePerShot"} {
					d.Axes[4].Facts = append(d.Axes[4].Facts, numberFact(ss[wi], key, "declared weapon configuration"))
				}
			}
		}
	}
	d.Axes[2].Facts = append(d.Axes[2].Facts, derived("scoring_slots_min", "slots", intValue(d.ScoringMin), source(root, "AddedBots"), "Resolved phase candidates; unknown scoring eligibility excluded from the lower bound"), derived("scoring_slots_max", "slots", intValue(d.ScoringMax), source(root, "AddedBots"), "Candidate eligibility is a bound, not simultaneous observed occupancy"))
	d.Axes[2].Facts = append(d.Axes[2].Facts, Fact{Key: "angular_size_and_transfer", Unit: "degrees", Status: "unknown", Unknown: []string{"map_scale_camera_collision_and_spawn_sampling_not_engine_verified"}})
	d.ControlledFingerprint = controlledFingerprint(ss, scoredCharacters, d)
	sort.Strings(d.Unknown)
	d.Unknown = unique(d.Unknown)
	return d
}
func intValue(v *int) *float64 {
	if v == nil {
		return nil
	}
	return num(float64(*v))
}
func unique(v []string) []string {
	o := []string{}
	for _, s := range v {
		if len(o) == 0 || o[len(o)-1] != s {
			o = append(o, s)
		}
	}
	return o
}

// ControlledFingerprint is deliberately narrow: only hitbox and gated motion
// caps may vary. Changed intervals, helpers, map geometry, clocks or phases are
// retained. It does not license a total difficulty order.
func controlledFingerprint(ss []sceSection, scored map[int]bool, d *Descriptor) string {
	if d.MapSHA256 == "" || len(d.Targets) == 0 {
		return ""
	}
	rows := []string{}
	for i, s := range ss {
		seen := map[string]bool{}
		profile := s.value("Name")
		if i == 0 {
			profile = ""
		}
		for _, f := range s.fields {
			if seen[f.Key] {
				return ""
			}
			seen[f.Key] = true
			if i == 0 && (f.Key == "Name" || f.Key == "Description" || f.Key == "DifficultyTag" || f.Key == "AimSubTypeTag") {
				continue
			}
			if scored[i] && (f.Key == "MainBBRadius" || f.Key == "MainBBHeight" || f.Key == "MaxSpeed" || f.Key == "Acceleration") {
				continue
			}
			b, _ := json.Marshal([]string{s.kind, profile, f.Key, f.Raw})
			rows = append(rows, string(b))
		}
	}
	sort.Strings(rows)
	rows = append(rows, "map:"+d.MapSHA256)
	rows = append(rows, "container:"+d.ContainerSignature)
	b, _ := json.Marshal(rows)
	return digest(b)
}
func features(d *Descriptor) []Fact {
	keys := []string{"precision_radius", "speed_width_pressure", "acceleration_width_pressure"}
	out := []Fact{}
	for _, key := range keys {
		sum := 0.
		complete := len(d.Targets) > 0
		sources := []SCEField{}
		for _, t := range d.Targets {
			r := factByKey(t.Facts, "MainBBRadius")
			sources = append(sources, r.Sources...)
			if !positive(r) {
				complete = false
				continue
			}
			if key == "precision_radius" {
				sum -= math.Log(*r.Value)
				continue
			}
			capKey := "MaxSpeed"
			if key == "acceleration_width_pressure" {
				capKey = "Acceleration"
			}
			cap := factByKey(t.Facts, capKey)
			sources = append(sources, cap.Sources...)
			sources = append(sources, factByKey(t.Facts, "NoDodging").Sources...)
			sources = append(sources, factByKey(t.Facts, "DodgeProfileNames").Sources...)
			if t.DodgeGate == "off" {
				continue
			}
			if t.DodgeGate != "candidate_on" || !nonnegative(cap) {
				complete = false
				continue
			}
			sum += math.Log1p(*cap.Value / (2 * *r.Value))
		}
		var value *float64
		if complete {
			value = num(sum / float64(len(d.Targets)))
		}
		out = append(out, derived(key, "configuration log descriptor", value, sources, "Equal weight across distinct scoring candidates, not elapsed time occupancy", "Only gated Dodge caps; ability, gravity and actual trajectory are not represented", "One SCE length/time unit; base and clock declarations retained separately"))
	}
	var scarcity *float64
	if d.ScoringMin != nil && d.ScoringMax != nil && *d.ScoringMin > 0 && *d.ScoringMin == *d.ScoringMax {
		scarcity = num(-math.Log(float64(*d.ScoringMin)))
	}
	out = append(out, derived("constant_scoring_slot_scarcity", "negative log slots", scarcity, nil, "All phases have explicit and equal scoring eligibility; free choice geometry would already include this opportunity effect"))
	return out
}
