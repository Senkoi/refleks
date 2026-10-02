package training

import (
	_ "embed"
	"encoding/json"
	"math"
)

type FileMeasurement struct {
	Profile string  `json:"profile,omitempty"`
	Field   string  `json:"field"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Line    int     `json:"line"`
}
type PrecisionComparison struct {
	Reference      string  `json:"reference"`
	ReferenceHash  string  `json:"referenceHash"`
	Profile        string  `json:"profile"`
	RadiusRatio    float64 `json:"radiusRatio"`
	PrecisionDelta float64 `json:"precisionDelta"`
}

//go:embed data/precision-references.json
var precisionReferences []byte

func calculateFileEvidence(a *LocalAssessment) {
	if a == nil || a.Status != "file_parsed_model_unfitted" {
		return
	}
	a.Measurements = nil
	a.PrecisionComparisons = nil
	player := ""
	for _, f := range a.Fields {
		if f.Section == "Root" && f.Key == "PlayerProfile" {
			player = f.Raw
		}
	}
	for _, f := range a.Fields {
		unit := ""
		switch {
		case f.Section == "Root" && (f.Key == "MapScale" || f.Key == "Timescale"):
			unit = "configuration multiplier"
		case f.Section == "Character Profile" && f.Profile != player && (f.Key == "MainBBRadius" || f.Key == "MainBBHeight"):
			unit = "configuration length"
		case f.Section == "Character Profile" && f.Profile != player && f.Key == "MaxSpeed":
			unit = "configuration speed limit"
		case f.Section == "Dodge Profile" && (f.Key == "MinLRTimeChange" || f.Key == "MaxLRTimeChange"):
			unit = "configured seconds"
		}
		if unit == "" {
			continue
		}
		v, ok := configNumber(f.Raw)
		if !ok || v < 0 {
			continue
		}
		a.Measurements = append(a.Measurements, FileMeasurement{Profile: f.Profile, Field: f.Key, Value: v, Unit: unit, Line: f.Line})
	}
	var pairs []struct {
		A, B, HashA, HashB, Profile string
		Ratio                       float64
	}
	_ = json.Unmarshal(precisionReferences, &pairs)
	for _, p := range pairs {
		reference, hash, ratio := "", "", 0.0
		if a.FileSHA256 == p.HashA {
			reference, hash, ratio = p.B, p.HashB, 1/p.Ratio
		}
		if a.FileSHA256 == p.HashB {
			reference, hash, ratio = p.A, p.HashA, p.Ratio
		}
		if reference == "" || ratio <= 0 {
			continue
		}
		a.PrecisionComparisons = append(a.PrecisionComparisons, PrecisionComparison{Reference: reference, ReferenceHash: hash, Profile: p.Profile, RadiusRatio: ratio, PrecisionDelta: -math.Log2(ratio)})
	}
}

func scenarioFromLocal(name, path string, m *Mechanics, a *LocalAssessment) Scenario {
	skill := classify(name)
	if m != nil && validSkill(m.DeclaredSkill) {
		skill = m.DeclaredSkill
	}
	return enrichMechanics(Scenario{Name: name, Skill: skill, Technique: technique(name, skill), Family: family(name), Difficulty: "unknown", DifficultySource: "unknown", Seconds: 60, Enabled: validSkill(skill), Classification: "inferred", Mechanics: m, LocalAssessment: a, Sources: []Source{{URL: "local-sce:" + path, Title: "本地 SCE", Retrieved: a.ObservedAt}}})
}
