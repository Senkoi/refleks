package training

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

//go:embed data/sce-mechanics-2026-10-02.json
var mechanicsSnapshot []byte

var snapshotMechanics = func() map[string]*Mechanics {
	var data struct { Scenarios []Scenario `json:"scenarios"` }
	if err := json.Unmarshal(mechanicsSnapshot, &data); err != nil { panic(err) }
	out := map[string]*Mechanics{}
	for _, s := range data.Scenarios { out[strings.ToLower(s.Name)] = s.Mechanics }
	return out
}()

func enrichMechanics(s Scenario) Scenario {
	if s.Mechanics == nil {
		if m := snapshotMechanics[strings.ToLower(s.Name)]; m != nil {
			copy := *m
			copy.Tags = append([]string{}, m.Tags...)
			s.Mechanics = &copy
		}
	}
	if s.Mechanics != nil && s.Classification != "manual" {
		if s.Mechanics.DeclaredSeconds >= 10 && s.Mechanics.DeclaredSeconds <= 3600 {
			s.Seconds = s.Mechanics.DeclaredSeconds
		}
		if s.Skill == "unknown" && s.Classification == "inferred" && validSkill(s.Mechanics.DeclaredSkill) {
			s.Skill, s.Technique = s.Mechanics.DeclaredSkill, s.Mechanics.DeclaredSkill
			s.Classification, s.Enabled = "sce_description", true
		}
	}
	return s
}

func hasDemand(s Scenario, tag string) bool {
	if s.Mechanics != nil {
		for _, t := range s.Mechanics.Tags { if t == tag { return true } }
	}
	return false
}

func pressureDemand(s Scenario) bool {
	return hasDemand(s, "time_pressure") || hasDemand(s, "pacing")
}

// Scoring constraints stay distinct from skill coverage. Multiple demand tags
// divide a run's time, so a mixed scene cannot count as several full runs.
func demandTags(s Scenario) []string {
	out := []string{}
	if s.Mechanics == nil { return out }
	seen := map[string]bool{}
	for _, tag := range s.Mechanics.Tags {
		switch tag {
		case "short_transfer", "wide_transfer", "micro_adjustment", "precision", "time_pressure", "reflex_window", "pacing", "phased_targets", "tracking", "projectile":
			if !seen[tag] { out = append(out, tag); seen[tag] = true }
		}
	}
	return out
}

func addDemandTime(s Scenario, seconds float64, exposure map[string]float64) {
	tags := demandTags(s)
	if len(tags) == 0 { return }
	for _, tag := range tags { exposure[tag] += seconds / float64(len(tags)) }
}

func demandWeight(s Scenario, exposure map[string]float64) float64 {
	tags := demandTags(s)
	if len(tags) == 0 { return 1 }
	w := 0.0
	for _, tag := range tags { w += 1 / (1 + exposure[tag]/180) }
	// Bounded coverage preference; absence of tags is not proof of low load.
	return math.Max(.2, w/float64(len(tags)))
}

func demandSimilarity(a, b Scenario) float64 {
	at, bt := demandTags(a), demandTags(b)
	if len(at) == 0 || len(bt) == 0 { return 0 }
	union := map[string]bool{}
	for _, tag := range at { union[tag] = true }
	shared := 0
	for _, tag := range bt { if union[tag] { shared++ }; union[tag] = true }
	return float64(shared) / float64(len(union))
}

var demandLabels = map[string]string{
	"short_transfer": "短距离转移", "wide_transfer": "大角度转移",
	"micro_adjustment": "微调", "precision": "精度", "time_pressure": "时间压力",
	"reflex_window": "限时反应", "pacing": "节奏递增", "phased_targets": "分阶段目标",
	"tracking": "跟踪", "projectile": "弹道预判",
}

func mechanicsReason(s Scenario) string {
	if s.Mechanics == nil { return " 场景细分机制未知，未按名称补造维度。" }
	labels := []string{}
	for _, t := range demandTags(s) { labels = append(labels, demandLabels[t]) }
	if len(labels) == 0 { return " 已有 SCE 快照，细分技能证据不足。" }
	return fmt.Sprintf(" 按 %s 分配覆盖与负荷；依据 SCE 快照及作者说明，尚未验证游戏运行机制或标定跨场景难度。", strings.Join(labels, "、"))
}
