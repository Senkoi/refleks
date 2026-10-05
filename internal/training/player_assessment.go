package training

import (
	"sort"
	"strings"
	"time"

	"aimmeow/internal/models"
	"aimmeow/internal/sceneanalysis"
)

// AssessmentInput contains evidence only. It cannot select or mutate a plan.
type AssessmentInput struct {
	Catalog     []Scenario
	Runs        []models.RunRecord
	Contexts    map[string]RunContext
	Preferences Preferences
	Now         time.Time
}
type PlayerAssessment struct {
	Anchors    []PersonalAnchor
	Levels     []PlayerLevel
	Skills     []SkillStatus
	Priorities map[string]ThemePriority
	Tiers      map[string]string
	Catalog    map[string]*CatalogAssessment
	Coverage   []DemandCoverage
}
type DemandCoverage struct {
	Theme      string   `json:"theme"`
	Key        string   `json:"key"`
	Min        float64  `json:"min"`
	Max        float64  `json:"max"`
	Scenes     []string `json:"scenes"`
	Status     string   `json:"status"`
	Conditions []string `json:"conditions"`
}

func AssessPlayer(in AssessmentInput) PlayerAssessment {
	out := PlayerAssessment{Catalog: map[string]*CatalogAssessment{}, Tiers: map[string]string{}, Anchors: []PersonalAnchor{}, Coverage: []DemandCoverage{}}
	anchors := personalAnchors(in.Catalog, in.Runs, in.Contexts, in.Now)
	references := automaticReferences(in.Catalog, in.Preferences)
	obs := levelObservations(in.Runs, in.Now)
	covered := map[string]*DemandCoverage{}
	for _, c := range in.Catalog {
		out.Catalog[c.Name] = assessCatalog(c, obs[strings.ToLower(c.Name)], in.Now, references)
		anchor, ok := anchors[strings.ToLower(c.Name)]
		if !ok {
			continue
		}
		out.Anchors = append(out.Anchors, anchor)
		a := c.LocalAssessment
		// Name-only history still supports task/native ability, but never relabels
		// that history with today's SCE requirements or an unrelated perf hash.
		if anchor.Status != "stable" || anchor.FileSHA256 == "" || a == nil || a.Status != "file_parsed_model_unfitted" || anchor.FileSHA256 != a.FileSHA256 || a.Requirements == nil || a.Requirements.Schema != sceneanalysis.DescriptorSchema || a.Requirements.SemanticVersion != sceneanalysis.SemanticVersion {
			continue
		}
		for _, f := range a.Requirements.Features {
			if f.Value == nil {
				continue
			}
			key := anchor.Theme + "\x00" + f.Key
			item := covered[key]
			if item == nil {
				item = &DemandCoverage{Theme: anchor.Theme, Key: f.Key, Min: *f.Value, Max: *f.Value, Status: "local_context_configuration_coverage", Conditions: []string{"Recent stable same-setting observations bound to local execution context; local file presence does not prove the game's loaded payload", "Range describes configurations with demonstrated task results; it is not a measured reaction time, motor limit or cross-scene score scale"}}
				covered[key] = item
			}
			item.Min = min(item.Min, *f.Value)
			item.Max = max(item.Max, *f.Value)
			item.Scenes = append(item.Scenes, c.Name)
		}
	}
	keys := []string{}
	for key := range covered {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.Coverage = append(out.Coverage, *covered[key])
	}
	out.Levels = PlayerLevels(in.Catalog, in.Runs, in.Now)
	out.Priorities = themePrioritiesWithLevels(in.Catalog, in.Runs, in.Now, out.Levels)
	for _, theme := range vdimThemes {
		out.Tiers[theme] = trainingTier(theme, out.Levels)
	}
	out.Skills = SkillProfile(in.Catalog, in.Runs, references, in.Now)
	return out
}
