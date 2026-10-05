package sceneanalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type MapProperty struct {
	Name string `json:"name"`
	Raw  string `json:"raw"`
}
type MapObject struct {
	Type       string        `json:"type"`
	Name       string        `json:"name"`
	Location   *[3]float64   `json:"location"`
	Rotation   *[3]float64   `json:"rotation"`
	Scale      *[3]float64   `json:"scale"`
	Properties []MapProperty `json:"properties,omitempty"`
}
type MapDescriptor struct {
	SHA256  string         `json:"sha256"`
	Format  string         `json:"format"`
	Status  string         `json:"status"`
	Counts  map[string]int `json:"counts"`
	Objects []MapObject    `json:"objects,omitempty"`
	Issues  []string       `json:"issues,omitempty"`
}

func vector(raw string) *[3]float64 {
	tokens := strings.FieldsFunc(raw, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' })
	if len(tokens) != 3 {
		return nil
	}
	v := [3]float64{}
	for i, s := range tokens {
		n, ok := ConfigNumber(s)
		if !ok {
			return nil
		}
		v[i] = n
	}
	return &v
}

// ParseMap retains every spawn point (including duplicate records), volume and
// event. Declared vectors and filters are not engine-verified geometry.
func ParseMap(body []byte) *MapDescriptor {
	loc := mapDataHeader.FindIndex(body)
	if loc == nil {
		return nil
	}
	raw := body[loc[1]:]
	m := &MapDescriptor{SHA256: digest(raw), Format: "unknown", Status: "unknown", Counts: map[string]int{}, Objects: []MapObject{}}
	b := bytes.TrimSpace(raw)
	if bytes.HasPrefix(b, []byte("{")) {
		var doc struct {
			Objects []struct {
				Type       string `json:"type"`
				Name       string `json:"name"`
				Location   string `json:"location"`
				Rotation   string `json:"rotation"`
				Scale      string `json:"scale"`
				Properties []struct {
					Name  string          `json:"name"`
					Value json.RawMessage `json:"value"`
				} `json:"properties"`
			} `json:"objects"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			m.Issues = append(m.Issues, "invalid_kmc_json")
			return m
		}
		m.Format = "kmc_json"
		m.Status = "declared"
		for _, o := range doc.Objects {
			obj := MapObject{Type: o.Type, Name: o.Name, Location: vector(o.Location), Rotation: vector(o.Rotation), Scale: vector(o.Scale)}
			seen := map[string]bool{}
			for _, p := range o.Properties {
				value := string(p.Value)
				var s string
				if json.Unmarshal(p.Value, &s) == nil {
					value = s
				}
				obj.Properties = append(obj.Properties, MapProperty{p.Name, value})
				if seen[p.Name] {
					m.Issues = append(m.Issues, "duplicate_map_property:"+p.Name)
				}
				seen[p.Name] = true
			}
			m.add(obj)
		}
	} else if bytes.HasPrefix(b, []byte("reflex map version 8")) {
		m.Format = "reflex_v8"
		m.Status = "declared"
		lines := strings.Split(string(b), "\n")
		for i := 0; i < len(lines); i++ {
			line := strings.TrimSuffix(lines[i], "\r")
			if strings.TrimSpace(line) != "entity" {
				continue
			}
			indent := indentation(line)
			obj := MapObject{Type: "entity"}
			seen := map[string]bool{}
			for j := i + 1; j < len(lines); j++ {
				text := strings.TrimSpace(lines[j])
				if text == "" {
					continue
				}
				if indentation(lines[j]) <= indent {
					break
				}
				parts := strings.SplitN(text, " ", 3)
				if len(parts) < 2 {
					m.Issues = append(m.Issues, "unsupported_reflex_entity_field")
					continue
				}
				if parts[0] == "type" {
					obj.Name = strings.TrimSpace(strings.TrimPrefix(text, "type"))
					continue
				}
				if len(parts) == 2 && parts[0] == "Bool" {
					parts = append(parts, "") // Preserve Reflex boolean markers without inventing a value.
				}
				if len(parts) != 3 {
					m.Issues = append(m.Issues, "unsupported_reflex_entity_field")
					continue
				}
				key, value := parts[1], parts[2]
				if seen[key] {
					m.Issues = append(m.Issues, "duplicate_map_property:"+key)
				}
				seen[key] = true
				obj.Properties = append(obj.Properties, MapProperty{key, value})
				switch key {
				case "position":
					obj.Location = vector(value)
				case "angles":
					obj.Rotation = vector(value)
				case "scale":
					obj.Scale = vector(value)
				}
			}
			m.add(obj)
		}
	} else {
		m.Issues = append(m.Issues, "unsupported_map_format")
	}
	return m
}
func indentation(s string) int { return len(s) - len(strings.TrimLeft(s, " \t")) }
func (m *MapDescriptor) add(o MapObject) {
	key := o.Name
	if key == "" {
		key = o.Type
	}
	m.Counts[key]++
	if o.Name == "PlayerSpawn" || o.Name == "SpawnPoint" || o.Name == "SpawnVolume" || o.Name == "Hurt" || o.Name == "Teleporter" || o.Name == "Waypoint" {
		m.Objects = append(m.Objects, o)
	}
}

type SpawnPool struct {
	Objects    []MapObject `json:"objects"`
	Status     string      `json:"status"`
	Conditions []string    `json:"conditions"`
	Issues     []string    `json:"issues,omitempty"`
}

// SpawnCandidates exposes a hypothesis explicitly. Team bit numbering, volume
// scale, axes, absent flags and collision are not assumed verified by KovaaK's.
func SpawnCandidates(m *MapDescriptor, character string, team int, player bool) SpawnPool {
	p := SpawnPool{Objects: []MapObject{}, Status: "calculated_under_explicit_conditions", Conditions: []string{"KMC bit(team-1), exact character filters with unique case fallback; Reflex teamB=player/teamA=bot hypothesis", "Raw point locations and volume declarations; no sampling, collision or camera reconstruction"}}
	if m == nil || m.Status != "declared" {
		p.Status = "unknown"
		p.Issues = []string{"missing_or_unsupported_map"}
		return p
	}
	for _, o := range m.Objects {
		if o.Name != "PlayerSpawn" && o.Name != "SpawnPoint" && o.Name != "SpawnVolume" {
			continue
		}
		props := map[string]string{}
		duplicate := false
		for _, v := range o.Properties {
			if _, ok := props[v.Name]; ok {
				duplicate = true
			}
			props[v.Name] = v.Raw
		}
		if duplicate {
			p.Issues = append(p.Issues, "duplicate_spawn_properties")
			continue
		}
		if m.Format == "kmc_json" {
			if raw := props["PermittedCharacterProfiles"]; raw != "" {
				names := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' })
				exact := false
				fallback := 0
				for _, n := range names {
					if n == character {
						exact = true
					}
					if strings.EqualFold(n, character) {
						fallback++
					}
				}
				if !exact && fallback != 1 {
					continue
				}
				if !exact {
					p.Issues = append(p.Issues, "case_filter_hypothesis:"+character)
				}
			}
			mask, ok := ConfigNumber(props["TeamMask"])
			if team > 0 && team <= 31 && ok {
				if mask != float64(int64(mask)) || mask < 0 {
					p.Issues = append(p.Issues, "invalid_team_mask")
					continue
				}
				if int64(mask)&(1<<uint(team-1)) == 0 {
					continue
				}
			} else if team != 0 {
				p.Issues = append(p.Issues, "unknown_team_filter")
			}
		} else {
			flag := "teamA"
			if player {
				flag = "teamB"
			}
			if team != 0 {
				if _, ok := props[flag]; !ok {
					p.Issues = append(p.Issues, "missing_reflex_team_marker")
					continue
				}
			}
		}
		if o.Location == nil {
			p.Issues = append(p.Issues, fmt.Sprintf("missing_location:%s", o.Name))
		}
		p.Objects = append(p.Objects, o)
	}
	return p
}
