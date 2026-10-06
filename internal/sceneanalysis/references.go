package sceneanalysis

import "strings"

type InfluenceEdge struct {
	FromKind   string `json:"fromKind"`
	From       string `json:"from"`
	ToKind     string `json:"toKind"`
	To         string `json:"to"`
	Field      string `json:"field"`
	Line       int    `json:"line"`
	Role       string `json:"role"`
	Activation string `json:"activation"`
	Resolved   bool   `json:"resolved"`
}

func referenceKind(ref string) string {
	suffixes := []struct{ suffix, kind string }{{".bot", "Bot Profile"}, {".rot", "Bot Rotation Profile"}, {".wpn", "Weapon Profile"}, {".dodge", "Dodge Profile"}, {".aim", "Aim Profile"}, {".char", "Character Profile"}, {".abilmov", "Movement Ability Profile"}, {".abilwep", "Weapon Ability Profile"}, {".abilmelee", "Melee Ability Profile"}}
	for _, v := range suffixes {
		if strings.HasSuffix(ref, v.suffix) {
			return v.kind
		}
	}
	return ""
}
func referenceGraph(ss []sceSection, d *Descriptor) []InfluenceEdge {
	out := []InfluenceEdge{}
	roles := map[string]string{}
	for _, t := range d.Helpers {
		roles["Bot Profile\x00"+t.Bot] = "helper"
		roles["Character Profile\x00"+t.Character] = "helper"
	}
	for _, t := range d.Targets {
		roles["Bot Profile\x00"+t.Bot] = "scoring_candidate"
		roles["Character Profile\x00"+t.Character] = "scoring_candidate"
	}
	defaults := map[string]string{"PlayerProfile": "Character Profile", "AddedBots": "Bot Profile", "CharacterProfile": "Character Profile", "WeaponProfileNames": "Weapon Profile", "WeaponsProfileNames": "Weapon Profile", "WeaponProfile": "Weapon Profile", "DodgeProfileNames": "Dodge Profile", "AimingProfileNames": "Aim Profile", "ProfileNames": "Bot Profile"}
	for _, s := range ss {
		for _, f := range s.fields {
			kind := defaults[f.Key]
			if kind == "" && !strings.Contains(f.Key, "Profile") {
				continue
			}
			for _, ref := range tokens(f.Raw) {
				dest := referenceKind(ref)
				if dest == "" {
					dest = kind
				}
				if dest == "" {
					continue
				}
				index, ok := resolveIndex(ss, dest, ref)
				to := ref
				if ok {
					to = ss[index].value("Name")
				}
				activation := "declared_reference_runtime_unknown"
				if f.Key == "DodgeProfileNames" && s.value("NoDodging") == "true" || f.Key == "AimingProfileNames" && s.value("NoAiming") == "true" {
					activation = "explicit_gate_off"
				}
				role := roles[s.kind+"\x00"+s.value("Name")]
				if role == "" {
					role = "definition"
				}
				out = append(out, InfluenceEdge{s.kind, s.value("Name"), dest, to, f.Key, f.Line, role, activation, ok})
			}
		}
	}
	return out
}
