package sceneanalysis

import (
	"math"
	"sort"
)

type AxisContrast struct {
	Key        string   `json:"key"`
	A          *float64 `json:"a"`
	B          *float64 `json:"b"`
	Delta      *float64 `json:"delta"`
	Status     string   `json:"status"`
	Conditions []string `json:"conditions,omitempty"`
}
type NativeOrder struct {
	Benchmark string `json:"benchmark"`
	Category  string `json:"category"`
	Family    string `json:"family"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	TierA     string `json:"tierA"`
	TierB     string `json:"tierB"`
	IndexA    int    `json:"indexA"`
	IndexB    int    `json:"indexB"`
	HashA     string `json:"hashA"`
	HashB     string `json:"hashB"`
	Verified  bool   `json:"verified"`
}
type Comparison struct {
	Kind              string         `json:"kind"`
	Direction         string         `json:"direction"`
	Task              string         `json:"task"`
	HashA             string         `json:"hashA"`
	HashB             string         `json:"hashB"`
	Axes              []AxisContrast `json:"axes"`
	Unknown           []string       `json:"unknown"`
	ChangedMechanisms []string       `json:"changedMechanisms"`
	Basis             []string       `json:"basis,omitempty"`
	NeighborDistance  *float64       `json:"neighborDistance"`
	RankMargin        *float64       `json:"rankMargin"`
	Native            *NativeOrder   `json:"native,omitempty"`
	PlannerUse        string         `json:"plannerUse"`
}

// FixedBasis prevents pairwise missing-term deletion from producing a false
// total order. No learned coefficients or cross-benchmark ranks are deployed.
func FixedBasis(task string) []string {
	switch task {
	case "tracking":
		return []string{"precision_radius", "speed_width_pressure", "acceleration_width_pressure"}
	case "clicking", "switching":
		return []string{"precision_radius", "speed_width_pressure", "acceleration_width_pressure", "constant_scoring_slot_scarcity"}
	}
	return nil
}
func descriptorBlocks(d *Descriptor) []string {
	reasons := []string{}
	if d.Schema != DescriptorSchema || d.SemanticVersion != SemanticVersion {
		reasons = append(reasons, "unsupported_descriptor_version")
	}
	if d.Task == "unknown" {
		reasons = append(reasons, "unresolved_task")
	}
	if len(d.Helpers) > 0 {
		reasons = append(reasons, "external_helper_influence_not_modeled")
	}
	for _, s := range d.Slots {
		if !s.Complete {
			reasons = append(reasons, "unresolved_phase")
		}
	}
	if d.ScoringMin == nil || d.ScoringMax == nil || *d.ScoringMin != *d.ScoringMax {
		reasons = append(reasons, "unknown_or_phase_dependent_scoring_slots")
	}
	for _, t := range d.Targets {
		if t.Scoring != "explicit_enabled" {
			reasons = append(reasons, "unverified_scoring_eligibility")
		}
		if t.Shape != "Sphere" && t.Shape != "Spheroid" && t.Shape != "Cylinder" {
			reasons = append(reasons, "unsupported_hitbox_shape")
		}
		if t.DodgeGate == "unknown" {
			reasons = append(reasons, "unknown_dodge_gate")
		}
		if len(t.Abilities) > 0 {
			reasons = append(reasons, "ability_motion_not_modeled")
		}
		gravity := factByKey(t.Facts, "Gravity")
		if gravity.Value == nil || *gravity.Value != 0 {
			reasons = append(reasons, "gravity_not_modeled")
		}
	}
	for _, def := range d.Definitions {
		if def.Kind != "Root" {
			continue
		}
		for _, key := range []string{"IsTimeDilationActive", "IsTargetSizeActive"} {
			f := definitionValue(def, key)
			if f != "false" {
				reasons = append(reasons, "unknown_or_adaptive_modifiers")
			}
		}
	}
	sort.Strings(reasons)
	return unique(reasons)
}
func definitionValue(d Definition, key string) string {
	s := sceSection{kind: d.Kind, fields: d.Fields}
	return s.value(key)
}

// Compare reports configuration demands. estimated_direction is explicitly a
// concordant direction on a fixed basis, not an automatic difficulty upgrade.
func Compare(a, b *Descriptor, native *NativeOrder) Comparison {
	c := Comparison{Kind: "partial_axes", Direction: "uncertain", Task: "unknown", Axes: []AxisContrast{}, Unknown: []string{}, ChangedMechanisms: []string{}, PlannerUse: "inspect_only"}
	if a == nil || b == nil {
		c.Unknown = append(c.Unknown, "missing_descriptor")
		return c
	}
	c.HashA, c.HashB = a.FileSHA256, b.FileSHA256
	if a.Task != b.Task || a.Task == "unknown" {
		c.Kind = "incompatible"
		c.Unknown = append(c.Unknown, "task_mismatch_or_unknown")
	} else {
		c.Task = a.Task
	}
	keys := []string{"precision_radius", "speed_width_pressure", "acceleration_width_pressure", "constant_scoring_slot_scarcity"}
	for _, key := range keys {
		af, bf := factByKey(a.Features, key), factByKey(b.Features, key)
		v := AxisContrast{Key: key, A: af.Value, B: bf.Value, Status: "unknown", Conditions: af.Conditions}
		if af.Value != nil && bf.Value != nil {
			v.Delta = num(*bf.Value - *af.Value)
			v.Status = "calculated_under_explicit_conditions"
		}
		c.Axes = append(c.Axes, v)
	}
	if native != nil && native.Verified && native.Benchmark != "" && native.Category != "" && native.Family != "" && native.Version != "" && native.Source != "" && native.HashA == a.FileSHA256 && native.HashB == b.FileSHA256 && native.IndexA != native.IndexB && c.Kind != "incompatible" {
		copy := *native
		c.Native = &copy
		c.Kind = "native_family_order"
		c.PlannerUse = "native_scope_only"
		c.Direction = "higher_native_tier"
		if native.IndexB < native.IndexA {
			c.Direction = "lower_native_tier"
		}
		return c
	}
	if a.MapSHA256 != b.MapSHA256 {
		c.ChangedMechanisms = append(c.ChangedMechanisms, "map_geometry")
	}
	if a.ControlledFingerprint == "" || a.ControlledFingerprint != b.ControlledFingerprint {
		c.ChangedMechanisms = append(c.ChangedMechanisms, "configuration_outside_supported_caps")
	}
	c.Unknown = append(c.Unknown, descriptorBlocks(a)...)
	c.Unknown = append(c.Unknown, descriptorBlocks(b)...)
	for i, t := range a.Targets {
		if i >= len(b.Targets) || t.Bot != b.Targets[i].Bot || t.Character != b.Targets[i].Character {
			c.Unknown = append(c.Unknown, "target_profile_mismatch")
			break
		}
		ah, bh := factByKey(t.Facts, "MainBBHeight"), factByKey(b.Targets[i].Facts, "MainBBHeight")
		if (ah.Value == nil) != (bh.Value == nil) || ah.Value != nil && *ah.Value != *bh.Value {
			c.Unknown = append(c.Unknown, "height_change_outside_radius_basis")
		}
		ar, br := factByKey(t.Facts, "MainBBRadius"), factByKey(b.Targets[i].Facts, "MainBBRadius")
		if t.Shape != "Sphere" && ar.Value != nil && br.Value != nil && *ar.Value != *br.Value {
			c.Unknown = append(c.Unknown, "unverified_shape_projection_for_radius_change")
		}
	}
	c.Basis = FixedBasis(c.Task)
	distance := 0.
	pos, neg := false, false
	complete := len(c.Basis) > 0
	for _, key := range c.Basis {
		af, bf := factByKey(a.Features, key), factByKey(b.Features, key)
		if af.Value == nil || bf.Value == nil {
			complete = false
			c.Unknown = append(c.Unknown, "incomplete_fixed_basis:"+key)
			continue
		}
		delta := *bf.Value - *af.Value
		distance += delta * delta
		pos = pos || delta > 1e-8
		neg = neg || delta < -1e-8
	}
	sort.Strings(c.Unknown)
	c.Unknown = unique(c.Unknown)
	if c.Kind != "incompatible" && complete && len(c.Unknown) == 0 && len(c.ChangedMechanisms) == 0 {
		c.NeighborDistance = num(math.Sqrt(distance / float64(len(c.Basis))))
		c.PlannerUse = "neighbor_tiebreak_only"
		if !(pos && neg) {
			c.Kind = "estimated_direction"
			c.Direction = "unchanged"
			if pos {
				c.Direction = "higher_declared_demand"
			}
			if neg {
				c.Direction = "lower_declared_demand"
			}
		}
	}
	return c
}
