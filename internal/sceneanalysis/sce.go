package sceneanalysis

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type sceSection struct {
	kind   string
	fields []SCEField
}

func (s sceSection) value(key string) string {
	value := ""
	count := 0
	for _, f := range s.fields {
		if f.Key == key {
			value = f.Raw
			count++
		}
	}
	if count != 1 {
		return ""
	}
	return value
}
func ConfigNumber(raw string) (float64, bool) {
	v, e := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return v, e == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

// ParseLocalSCE reads configuration only. It never imports a .bin cache, invokes
// Steam/downloaders or claims a calibrated total-difficulty score.
func ParseLocalSCE(data []byte) (string, *Mechanics, *LocalAssessment, error) {
	rawData := data
	data, container, err := Decode(data)
	if err != nil {
		return "", nil, nil, err
	}
	sections := []sceSection{{kind: "Root"}}
	scan := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	scan.Buffer(make([]byte, 4096), 1<<20)
	line := 0
	opaqueConfiguration := false
	for scan.Scan() {
		line++
		rawLine := strings.TrimSuffix(scan.Text(), "\r")
		s := strings.TrimSpace(rawLine)
		if s == "" || strings.HasPrefix(s, ";") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
			continue
		}
		if s == "[Map Data]" {
			break
		}
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			if len(sections) >= 4096 {
				return "", nil, nil, fmt.Errorf("SCE profile 数量超过解析上限")
			}
			sections = append(sections, sceSection{kind: s[1 : len(s)-1]})
			continue
		}
		if key, value, ok := strings.Cut(rawLine, "="); ok {
			i := len(sections) - 1
			sections[i].fields = append(sections[i].fields, SCEField{Section: sections[i].kind, Key: strings.TrimSpace(key), Raw: value, Line: line})
		} else if s != "" && !strings.HasPrefix(s, ";") && !strings.HasPrefix(s, "#") && !strings.HasPrefix(s, "//") {
			opaqueConfiguration = true
		}
	}
	if err := scan.Err(); err != nil {
		return "", nil, nil, err
	}
	root := sections[0]
	name := strings.TrimSpace(root.value("Name"))
	if name == "" || len(name) > 300 || strings.ContainsAny(name, "\r\n\x00") {
		return "", nil, nil, fmt.Errorf("SCE 缺少唯一有效内部 Name")
	}
	digest := sha256.Sum256(rawData)
	hash := hex.EncodeToString(digest[:])
	a := &LocalAssessment{Status: "file_parsed_model_unfitted", FileSHA256: hash, GameVersion: root.value("GameVersion"), Fields: append([]SCEField{}, root.fields...), Issues: []string{"map_geometry_and_total_difficulty_not_fitted"}}
	a.Container = &container
	a.BodySHA256 = container.BodySHA256
	if opaqueConfiguration {
		a.Issues = append(a.Issues, "unsupported_configuration_line")
	}
	index := map[string]int{}
	for i := 1; i < len(sections); i++ {
		p := sections[i].value("Name")
		key := sections[i].kind + "\x00" + p
		if _, exists := index[key]; exists && p != "" {
			return name, nil, nil, fmt.Errorf("重复 profile Name")
		}
		index[key] = i
	}
	active := map[int]bool{}
	var resolve func(string, string)
	resolve = func(kind, token string) {
		if token == "" {
			return
		}
		i, ok := resolveIndex(sections, kind, token)
		if !ok {
			a.Issues = append(a.Issues, "unresolved:"+kind+":"+token)
			return
		}
		if active[i] {
			return
		}
		active[i] = true
		p := sections[i]
		expected := token
		if p.value("Name") != token {
			for _, suffix := range []string{".bot", ".rot", ".wpn", ".dodge", ".aim", ".char", ".abilmov", ".abilwep", ".abilmelee"} {
				if strings.HasSuffix(token, suffix) {
					expected = strings.TrimSuffix(token, suffix)
					break
				}
			}
		}
		if p.value("Name") != expected {
			a.Issues = append(a.Issues, "case_variant_reference:"+token)
		}
		for _, f := range p.fields {
			f.Profile = p.value("Name")
			a.Fields = append(a.Fields, f)
		}
		refs := map[string]string{}
		switch kind {
		case "Bot Profile":
			refs = map[string]string{"CharacterProfile": "Character Profile", "DodgeProfileNames": "Dodge Profile", "AimingProfileNames": "Aim Profile"}
			if p.value("NoDodging") == "true" {
				delete(refs, "DodgeProfileNames")
			}
			if p.value("NoAiming") == "true" {
				delete(refs, "AimingProfileNames")
			}
		case "Character Profile":
			refs = map[string]string{"WeaponProfileNames": "Weapon Profile", "AbilityProfileNames": "ability"}
		case "Bot Rotation Profile":
			refs = map[string]string{"ProfileNames": "Bot Profile"}
		case "Weapon Ability Profile":
			refs = map[string]string{"WeaponProfile": "Weapon Profile"}
		}
		keys := []string{}
		for key := range refs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, ref := range strings.Split(p.value(key), ";") {
				if ref == "" {
					continue
				}
				dest := refs[key]
				if dest == "ability" {
					switch {
					case strings.HasSuffix(ref, ".abilmov"):
						dest = "Movement Ability Profile"
					case strings.HasSuffix(ref, ".abilwep"):
						dest = "Weapon Ability Profile"
					case strings.HasSuffix(ref, ".abilmelee"):
						dest = "Melee Ability Profile"
					default:
						a.Issues = append(a.Issues, "unsupported_ability:"+ref)
						continue
					}
				}
				resolve(dest, ref)
			}
		}
	}
	resolve("Character Profile", root.value("PlayerProfile"))
	for _, token := range strings.Split(root.value("AddedBots"), ";") {
		if strings.HasSuffix(token, ".rot") {
			resolve("Bot Rotation Profile", token)
		} else {
			resolve("Bot Profile", token)
		}
	}
	m := &Mechanics{FileSHA256: hash, Status: "local_file_parsed_model_unfitted", Role: "unknown", GeometryStatus: "unknown", Tags: []string{}}
	desc := strings.ToLower(root.value("Description"))
	rules := map[string][]string{
		"short_transfer": {"two close, small targets"}, "wide_transfer": {"wide curved wall", "wideflicks"},
		"micro_adjustment": {"microadjustments", "wideflicks and micros", "two close, small targets"},
		"precision":        {"precision on tiny targets", "extra care", "two close, small targets"},
		"time_pressure":    {"before they reach you", "before they despawn or collide", "before it despawns", "falls to the ground"},
		"reflex_window":    {"short-lived target", "lifetime: 500 ms"}, "pacing": {"pacing to increase"},
		"phased_targets": {"number of targets increases every", "after killing 3 big bots"},
		"tracking":       {"hitscan tracking", "track vertically", "tracking while revolving"}, "projectile": {"fast projectile", "midair rockets"},
	}
	for tag, phrases := range rules {
		for _, p := range phrases {
			if strings.Contains(desc, p) {
				m.Tags = append(m.Tags, tag)
				break
			}
		}
	}
	if strings.Contains(desc, "static clicking scenario") {
		m.DeclaredSkill = "static"
	}
	if strings.EqualFold(root.value("ScoreMultAccuracy"), "true") {
		m.Tags = append(m.Tags, "accuracy_constraint")
	}
	if v, ok := ConfigNumber(root.value("Timelimit")); ok && v >= 10 && v <= 3600 {
		m.DeclaredSeconds = int(math.Ceil(v))
	}
	player, ok := resolveIndex(sections, "Character Profile", root.value("PlayerProfile"))
	if ok {
		for _, w := range strings.Split(sections[player].value("WeaponProfileNames"), ";") {
			w = strings.TrimSuffix(w, ".wpn")
			wi, exists := resolveIndex(sections, "Weapon Profile", w)
			if !exists {
				continue
			}
			constrained := true
			for _, k := range []string{"MagazineMax", "AmmoPerShot", "AmmoReloadedOnKill", "ReloadTimeFromEmpty"} {
				v, valid := ConfigNumber(sections[wi].value(k))
				constrained = constrained && valid && v > 0
			}
			if constrained {
				m.Tags = append(m.Tags, "reload_constraint")
				break
			}
		}
	}
	sort.Strings(m.Tags)
	sort.Strings(a.Issues)
	CalculateFileEvidence(a)
	a.Requirements = describe(data, sections, a)
	calculateComparison(data, sections, active, a)
	return name, m, a, nil
}

func resolveIndex(sections []sceSection, kind, token string) (int, bool) {
	names := []string{token}
	for _, suffix := range []string{".bot", ".rot", ".wpn", ".dodge", ".aim", ".char", ".abilmov", ".abilwep", ".abilmelee"} {
		if strings.HasSuffix(token, suffix) {
			names = append(names, strings.TrimSuffix(token, suffix))
			break
		}
	}
	for _, name := range names {
		for i, s := range sections {
			if s.kind == kind && s.value("Name") == name && name != "" {
				return i, true
			}
		}
	}
	for _, name := range names {
		match := -1
		for i, s := range sections {
			if s.kind == kind && strings.EqualFold(s.value("Name"), name) && name != "" {
				if match >= 0 {
					return 0, false
				}
				match = i
			}
		}
		if match >= 0 {
			return match, true
		}
	}
	return 0, false
}
