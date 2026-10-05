package training

import (
	"aimmeow/internal/sceneanalysis"
	"fmt"
	"math"
	"time"
)

const comparisonSchema = sceneanalysis.ComparisonSchema

type ScenarioComparison struct {
	Anchor    string                   `json:"anchor"`
	Candidate string                   `json:"candidate"`
	Result    sceneanalysis.Comparison `json:"result"`
	Precision *PrecisionRelation       `json:"precision,omitempty"`
}

// A missing legacy task tag may use confirmed native/manual taxonomy. Name
// guesses never complete the fixed basis; explicit SCE tags keep priority.
func contextualRequirements(c Scenario) *sceneanalysis.Descriptor {
	d := c.LocalAssessment.Requirements
	if d == nil || d.Task != "unknown" {
		return d
	}
	if c.Classification != "manual" && c.Classification != "benchmark" {
		return d
	}
	copy := *d
	switch c.Skill {
	case "static", "dynamic":
		copy.Task = "clicking"
	case "smooth", "reactive":
		copy.Task = "tracking"
	case "switching":
		copy.Task = "switching"
	}
	return &copy
}

func CompareScenarios(anchor, candidate Scenario) ScenarioComparison {
	var a, b *sceneanalysis.Descriptor
	if anchor.LocalAssessment != nil && anchor.LocalAssessment.Status == "file_parsed_model_unfitted" {
		a = contextualRequirements(anchor)
	}
	if candidate.LocalAssessment != nil && candidate.LocalAssessment.Status == "file_parsed_model_unfitted" {
		b = contextualRequirements(candidate)
	}
	c := ScenarioComparison{Anchor: anchor.Name, Candidate: candidate.Name, Result: sceneanalysis.Compare(a, b, nil)}
	if anchor.Skill != candidate.Skill || scenarioTheme(anchor) != scenarioTheme(candidate) {
		c.Result.Kind, c.Result.PlannerUse = "incompatible", "inspect_only"
		c.Result.NeighborDistance = nil
		c.Result.Unknown = append(c.Result.Unknown, "training_goal_mismatch")
		return c
	}
	if r := precisionRelation(anchor, candidate); r != nil {
		c.Precision = r
		c.Result.Kind, c.Result.Direction, c.Result.PlannerUse = "strict_precision", r.Direction, "controlled_precision_trial"
	}
	return c
}

func (s *Service) CompareScenes(anchor, candidate string) (ScenarioComparison, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var a, b *Scenario
	for i := range s.state.Catalog {
		c := &s.state.Catalog[i]
		if c.Name == anchor {
			a = c
		}
		if c.Name == candidate {
			b = c
		}
	}
	if a == nil || b == nil {
		return ScenarioComparison{}, fmt.Errorf("对照场景不在关卡库中")
	}
	return CompareScenarios(*a, *b), nil
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
