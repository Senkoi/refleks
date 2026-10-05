package sceneanalysis

import (
	"fmt"
	"math"
)

// RankModel is an explicit offline artifact. There are no default fitted
// coefficients, no runtime model setting, and no conversion to success odds.
type RankModel struct {
	Version          string    `json:"version"`
	Status           string    `json:"status"`
	Task             string    `json:"task"`
	Basis            []string  `json:"basis"`
	Weights          []float64 `json:"weights"`
	TrainPairRMS     []float64 `json:"trainPairRMS"`
	TrainingEvidence string    `json:"trainingEvidence"`
}
type ShadowEstimate struct {
	Version string   `json:"version"`
	Status  string   `json:"status"`
	Margin  *float64 `json:"margin"`
	Reasons []string `json:"reasons,omitempty"`
}

func (m RankModel) Validate() error {
	basis := FixedBasis(m.Task)
	if m.Version == "" || m.Status != "shadow_only" || m.TrainingEvidence == "" || len(basis) == 0 || len(m.Basis) != len(basis) || len(m.Weights) != len(basis) || len(m.TrainPairRMS) != len(basis) {
		return fmt.Errorf("shadow model requires version, training evidence and a complete fixed task basis")
	}
	for i, key := range basis {
		if m.Basis[i] != key || m.Weights[i] < 0 || m.TrainPairRMS[i] <= 0 || math.IsNaN(m.Weights[i]+m.TrainPairRMS[i]) || math.IsInf(m.Weights[i]+m.TrainPairRMS[i], 0) {
			return fmt.Errorf("invalid shadow basis/scale at %s", key)
		}
	}
	return nil
}

// ShadowMargin is antisymmetric within a compatible fixed basis. It does not
// alter Compare.Kind, scene grades, recommended tiers or already fixed plans.
func ShadowMargin(a, b *Descriptor, m RankModel) (ShadowEstimate, error) {
	out := ShadowEstimate{Version: m.Version, Status: "abstained"}
	if err := m.Validate(); err != nil {
		return out, err
	}
	c := Compare(a, b, nil)
	if a == nil || b == nil || a.Task != m.Task || b.Task != m.Task || c.NeighborDistance == nil {
		out.Reasons = append(append([]string{}, c.Unknown...), c.ChangedMechanisms...)
		return out, nil
	}
	margin := 0.
	for i, key := range m.Basis {
		af, bf := factByKey(a.Features, key), factByKey(b.Features, key)
		if af.Value == nil || bf.Value == nil {
			return out, nil
		}
		margin += m.Weights[i] * (*bf.Value - *af.Value) / m.TrainPairRMS[i]
	}
	if math.IsNaN(margin) || math.IsInf(margin, 0) {
		out.Reasons = []string{"nonfinite_margin"}
		return out, nil
	}
	out.Status = "shadow_only"
	out.Margin = num(margin)
	return out, nil
}
