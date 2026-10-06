package sceneanalysis

import "math"

type DodgeMotion struct {
	Profile          string          `json:"profile"`
	Axis             string          `json:"axis"`
	DwellMin         float64         `json:"dwellMin"`
	DwellMax         float64         `json:"dwellMax"`
	MidpointEnvelope *MotionEnvelope `json:"midpointEnvelope"`
	Sources          []SCEField      `json:"sources"`
}

func dodgeMotion(t Target) []DodgeMotion {
	out := []DodgeMotion{}
	v, a := factByKey(t.Facts, "MaxSpeed"), factByKey(t.Facts, "Acceleration")
	if t.DodgeGate != "candidate_on" || !nonnegative(v) || !nonnegative(a) {
		return out
	}
	for _, d := range t.DodgeEntries {
		s := sceSection{kind: d.Kind, fields: d.Fields}
		for _, axis := range []string{"LR", "FB"} {
			flag := "ToggleLeftRight"
			if axis == "FB" {
				flag = "ToggleForwardBack"
			}
			lo, hi := numberFact(s, "Min"+axis+"TimeChange", "configured seconds"), numberFact(s, "Max"+axis+"TimeChange", "configured seconds")
			if s.value(flag) != "true" || !positive(lo) || !positive(hi) || *hi.Value < *lo.Value {
				continue
			}
			envelope, ok := PeriodicMotion(*v.Value, *a.Value, (*lo.Value+*hi.Value)/2)
			if ok {
				out = append(out, DodgeMotion{d.Name, axis, *lo.Value, *hi.Value, envelope, append(append(v.Sources, a.Sources...), append(lo.Sources, hi.Sources...)...)})
			}
		}
	}
	return out
}

type HazardExposure struct {
	Bot          string   `json:"bot"`
	Character    string   `json:"character"`
	Helper       bool     `json:"helper"`
	MapSHA256    string   `json:"mapSHA256"`
	ObjectIndex  int      `json:"objectIndex"`
	DamageEvents int      `json:"damageEvents"`
	MinSeconds   float64  `json:"minSeconds"`
	MaxSeconds   float64  `json:"maxSeconds"`
	Status       string   `json:"status"`
	Conditions   []string `json:"conditions"`
}

func hazardExposures(d *Descriptor) []HazardExposure {
	out := []HazardExposure{}
	if d.Map == nil {
		return out
	}
	for _, t := range append(append([]Target{}, d.Targets...), d.Helpers...) {
		h, regen := factByKey(t.Facts, "MaxHealth"), factByKey(t.Facts, "HealthRegenPerSec")
		if !positive(h) || regen.Value == nil || *regen.Value != 0 {
			continue
		}
		for i, o := range d.Map.Objects {
			if o.Name != "Hurt" {
				continue
			}
			props := map[string]string{}
			duplicate := false
			for _, p := range o.Properties {
				if _, ok := props[p.Name]; ok {
					duplicate = true
				}
				props[p.Name] = p.Raw
			}
			if duplicate {
				continue
			}
			damage, ok := ConfigNumber(props["Damage"])
			cooldown, valid := ConfigNumber(props["Cooldown"])
			if !ok || !valid || damage <= 0 || cooldown <= 0 {
				continue
			}
			events := math.Ceil(*h.Value / damage)
			if events > 1e9 {
				continue
			}
			out = append(out, HazardExposure{Bot: t.Bot, Character: t.Character, Helper: t.Scoring == "disabled", MapSHA256: d.Map.SHA256, ObjectIndex: i, DamageEvents: int(events), MinSeconds: (events - 1) * cooldown, MaxSeconds: events * cooldown, Status: "calculated_under_explicit_conditions", Conditions: []string{"Continuous overlap with this one region; damage applies without invincibility, mitigation or external healing", "First trigger occurs within one cooldown; collision, event timeline and phase transition are not verified"}})
		}
	}
	return out
}
