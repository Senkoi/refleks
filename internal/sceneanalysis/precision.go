package sceneanalysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

const ComparisonSchema = 2
const comparisonSchema = ComparisonSchema

var mapDataHeader = regexp.MustCompile("(?m)^[ \\t]*\\[Map Data\\][ \\t]*\\r?$")

// Hash all defined configuration and the entire map payload, removing only
// root editorial labels and reachable non-player hitbox sizes. Unused profile
// changes, omitted defaults, phases, scoring and motion remain in the hash.
func calculateComparison(data []byte, sections []sceSection, active map[int]bool, a *LocalAssessment) {
	a.ComparisonSchema = comparisonSchema
	location := mapDataHeader.FindIndex(data)
	if location == nil {
		return
	}
	h := sha256.Sum256(data[location[1]:])
	a.MapDataSHA256 = hex.EncodeToString(h[:])
	for _, issue := range a.Issues {
		if strings.HasPrefix(issue, "unresolved:") || strings.HasPrefix(issue, "unsupported_ability:") || issue == "unsupported_configuration_line" {
			return
		}
	}
	player := strings.TrimSuffix(sections[0].value("PlayerProfile"), ".char")
	if player == "" || sections[0].value("AddedBots") == "" {
		return
	}
	canonical := []string{}
	scoring := map[string]bool{}
	if a.Requirements != nil {
		for _, t := range a.Requirements.Targets {
			scoring[t.Character] = true
		}
	}
	for i, section := range sections {
		seen := map[string]bool{}
		profile := section.value("Name")
		if i == 0 {
			profile = ""
		}
		target := active[i] && scoring[profile] && section.kind == "Character Profile" && !strings.EqualFold(profile, player)
		var size TargetSize
		size.Profile = profile
		for _, f := range section.fields {
			if seen[f.Key] { // Repeated keys have no assumed override semantics.
				a.Issues = append(a.Issues, "comparison_duplicate_field:"+section.kind+":"+f.Key)
				return
			}
			seen[f.Key] = true
			if i == 0 && (f.Key == "Name" || f.Key == "Description" || f.Key == "DifficultyTag" || f.Key == "AimSubTypeTag") {
				continue
			}
			if target && (f.Key == "MainBBRadius" || f.Key == "MainBBHeight") {
				v, ok := ConfigNumber(f.Raw)
				if !ok || v <= 0 {
					return
				}
				if f.Key == "MainBBRadius" {
					size.Radius = v
				} else {
					size.Height = &v
				}
				continue
			}
			b, _ := json.Marshal([]string{section.kind, profile, f.Key, f.Raw})
			canonical = append(canonical, string(b))
		}
		if target {
			if size.Radius <= 0 {
				return
			}
			a.TargetSizes = append(a.TargetSizes, size)
		}
	}
	if len(a.TargetSizes) == 0 {
		return
	}
	sort.Slice(a.TargetSizes, func(i, j int) bool { return a.TargetSizes[i].Profile < a.TargetSizes[j].Profile })
	sort.Strings(canonical)
	canonical = append(canonical, "map:"+a.MapDataSHA256)
	canonical = append(canonical, "container:"+containerSignature(a.Container))
	b, _ := json.Marshal(canonical)
	h = sha256.Sum256(b)
	a.FamilyFingerprint = hex.EncodeToString(h[:])
}
