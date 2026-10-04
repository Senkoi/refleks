package training

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

const comparisonSchema = 1

type TargetSize struct {
	Profile string   `json:"profile"`
	Radius  float64  `json:"radius"`
	Height  *float64 `json:"height,omitempty"`
}

type PrecisionRelation struct {
	FamilyFingerprint string                `json:"familyFingerprint"`
	Profiles          []PrecisionComparison `json:"profiles"`
	Direction         string                `json:"direction"`
	MaxDelta          float64               `json:"maxDelta"`
	Uniform           bool                  `json:"uniform"`
	UniformDelta      float64               `json:"uniformDelta,omitempty"`
}

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
	for i, section := range sections {
		seen := map[string]bool{}
		profile := section.value("Name")
		if i == 0 {
			profile = ""
		}
		target := active[i] && section.kind == "Character Profile" && !strings.EqualFold(profile, player)
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
				v, ok := configNumber(f.Raw)
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
	b, _ := json.Marshal(canonical)
	h = sha256.Sum256(b)
	a.FamilyFingerprint = hex.EncodeToString(h[:])
}

// Rebuild references when local files change, not on clock/UI refresh. Copy
// assessments so already generated plans keep their original file evidence.
func refreshLocalRelations(catalog []Scenario) {
	groups := map[string][]int{}
	for i, c := range catalog {
		if a := c.LocalAssessment; a != nil && a.Status == "file_parsed_model_unfitted" {
			copy := *a
			calculateFileEvidence(&copy)
			catalog[i].LocalAssessment = &copy
			if a.FamilyFingerprint != "" {
				groups[a.FamilyFingerprint] = append(groups[a.FamilyFingerprint], i)
			}
		}
	}
	for _, indices := range groups {
		for _, i := range indices {
			a := catalog[i].LocalAssessment
			for _, j := range indices {
				if i == j {
					continue
				}
				r := precisionRelation(catalog[j], catalog[i])
				if r == nil {
					continue
				}
				for _, contrast := range r.Profiles {
					duplicate := false
					for _, old := range a.PrecisionComparisons {
						if old.ReferenceHash == contrast.ReferenceHash && old.Profile == contrast.Profile {
							duplicate = true
							break
						}
					}
					if !duplicate {
						a.PrecisionComparisons = append(a.PrecisionComparisons, contrast)
					}
				}
				if len(a.PrecisionComparisons) >= 12 {
					break
				}
			}
		}
	}
}

// File hashes are not compared to CSV Hash: their algorithms are unrelated.
func precisionRelation(anchor, candidate Scenario) *PrecisionRelation {
	a, b := anchor.LocalAssessment, candidate.LocalAssessment
	if a == nil || b == nil || a.Status != "file_parsed_model_unfitted" || b.Status != "file_parsed_model_unfitted" ||
		a.ComparisonSchema != comparisonSchema || b.ComparisonSchema != comparisonSchema ||
		a.FamilyFingerprint == "" || a.FamilyFingerprint != b.FamilyFingerprint ||
		a.FileSHA256 == b.FileSHA256 || anchor.Skill != candidate.Skill || scenarioTheme(anchor) != scenarioTheme(candidate) {
		return nil
	}
	if assessCatalog(anchor, nil, time.Time{}, Preferences{}).FileStatus != "verified" ||
		assessCatalog(candidate, nil, time.Time{}, Preferences{}).FileStatus != "verified" || len(a.TargetSizes) != len(b.TargetSizes) {
		return nil
	}
	r := &PrecisionRelation{FamilyFingerprint: a.FamilyFingerprint, Uniform: true, Direction: "unchanged"}
	direction := 0
	for i, source := range a.TargetSizes {
		target := b.TargetSizes[i]
		if source.Profile != target.Profile || source.Radius <= 0 || target.Radius <= 0 || (source.Height == nil) != (target.Height == nil) {
			return nil
		}
		ratio := target.Radius / source.Radius
		if source.Height != nil && math.Abs(*target.Height / *source.Height - ratio) > 1e-6 {
			return nil // Shape/aspect changes are not a radius-only precision axis.
		}
		delta := -math.Log2(ratio)
		if math.IsNaN(delta) || math.IsInf(delta, 0) {
			return nil
		}
		sign := 0
		if delta > 1e-6 {
			sign = 1
		}
		if delta < -1e-6 {
			sign = -1
		}
		if sign != 0 && direction != 0 && sign != direction {
			return nil // Mixed directions cannot define a single harder/easier variant.
		}
		if sign != 0 {
			direction = sign
		}
		r.MaxDelta = math.Max(r.MaxDelta, math.Abs(delta))
		if i == 0 {
			r.UniformDelta = delta
		}
		r.Uniform = r.Uniform && math.Abs(delta-r.UniformDelta) <= 1e-6
		r.Profiles = append(r.Profiles, PrecisionComparison{Reference: anchor.Name, ReferenceHash: a.FileSHA256, Profile: source.Profile, RadiusRatio: ratio, PrecisionDelta: delta})
	}
	if direction == 0 {
		return nil
	}
	if direction > 0 {
		r.Direction = "higher_precision"
	} else {
		r.Direction = "lower_precision"
	}
	return r
}
