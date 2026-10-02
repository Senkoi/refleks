package training

import (
	"math"
	"strings"
	"time"
)

// File verification and difficulty evidence are separate. A parsed SCE, a
// name guess or a manual difficulty label cannot masquerade as a fitted model.
type CatalogAssessment struct {
	FileStatus            string `json:"fileStatus"`
	DifficultyStatus      string `json:"difficultyStatus"`
	HasDifficultyEvidence bool   `json:"hasDifficultyEvidence"`
	HasPrecisionReference bool   `json:"hasPrecisionReference"`
	HasBenchmarkReference bool   `json:"hasBenchmarkReference"`
	Fit                   string `json:"fit"`
	Samples               int    `json:"samples"`
}

func assessCatalog(s Scenario, rows []observation, now time.Time, p Preferences) *CatalogAssessment {
	e := &CatalogAssessment{FileStatus: "missing", DifficultyStatus: "unfitted", Fit: "unknown"}
	a := s.LocalAssessment
	if a == nil {
		return e
	}
	switch a.Status {
	case "ambiguous_local_versions":
		e.FileStatus = "conflict"
	case "invalid_local_file":
		e.FileStatus = "invalid"
	case "waiting_for_stable_local_file":
		e.FileStatus = "pending"
	case "local_file_unavailable":
		e.FileStatus = "missing"
	case "file_parsed_model_unfitted":
		e.FileStatus = "parsed"
		player := ""
		durationOK := false
		playerFields, durationFields := 0, 0
		for _, f := range a.Fields {
			if f.Section != "Root" {
				continue
			}
			if f.Key == "PlayerProfile" {
				playerFields++
				player = strings.TrimSpace(f.Raw)
			}
			if f.Key == "Timelimit" {
				durationFields++
				v, ok := configNumber(f.Raw)
				durationOK = ok && v > 0 && v <= 3600
			}
		}
		referencesOK := true
		for _, issue := range a.Issues {
			if strings.HasPrefix(issue, "unresolved:") || strings.HasPrefix(issue, "unsupported_ability:") {
				referencesOK = false
			}
		}
		if a.FileSHA256 != "" && a.FilePath != "" && player != "" && durationOK && referencesOK && playerFields == 1 && durationFields == 1 {
			e.FileStatus = "verified"
		}
	default:
		e.FileStatus = "pending"
	}
	if e.FileStatus != "verified" {
		return e
	}
	for _, r := range a.PrecisionComparisons {
		if r.ReferenceHash != "" && r.RadiusRatio > 0 && !math.IsNaN(r.RadiusRatio) && !math.IsInf(r.RadiusRatio, 0) {
			e.HasPrecisionReference = true
			break
		}
	}
	for _, m := range memberships(s) {
		if !selectedBenchmark(p, m.Name) || m.NativeDifficulty == "" {
			continue
		}
		for _, v := range m.Thresholds {
			if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
				e.HasBenchmarkReference = true
				break
			}
		}
	}
	fit := assessDifficultyFor(s, rows, now, p)
	e.Fit, e.Samples = fit.Fit, fit.Samples
	e.HasDifficultyEvidence = e.HasPrecisionReference || e.HasBenchmarkReference
	switch {
	case e.HasPrecisionReference:
		e.DifficultyStatus = "precision_reference"
	case e.HasBenchmarkReference:
		e.DifficultyStatus = "benchmark_reference"
	}
	return e
}
