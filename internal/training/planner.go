package training

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"refleks/internal/models"
)

type observation struct {
	score, duration float64
	at              time.Time
	signature       string
}

func signature(s models.RunStatsSummary) string {
	return fmt.Sprintf("%s|%s|%.4f|%.4f|%.4f|%.4f", s.Hash, s.SensScale, s.HorizSens, s.VertSens, s.FOV, s.AvgTimeDilation)
}

func runKey(r models.RunRecord) string {
	if r.FilePath != "" {
		return r.FilePath
	}
	return r.FileName + "|" + r.Stats.Summary.DatePlayed + "|" + r.Stats.Summary.Scenario
}

func observed(runs []models.RunRecord) map[string][]observation {
	out := map[string][]observation{}
	seen := map[string]bool{}
	for _, r := range runs {
		if seen[runKey(r)] {
			continue
		}
		seen[runKey(r)] = true
		s := r.Stats.Summary
		t, err := time.Parse(time.RFC3339, s.DatePlayed)
		if err != nil {
			continue
		}
		if s.Score < 0 || math.IsNaN(s.Score) || math.IsInf(s.Score, 0) {
			continue
		}
		sig := signature(s)
		key := strings.ToLower(s.Scenario)
		out[key] = append(out[key], observation{s.Score, s.Duration, t, sig})
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool { return out[key][i].at.After(out[key][j].at) })
	}
	return out
}

// Only runs matching the latest scenario/settings signature inform a target.
func comparable(rows []observation) []observation {
	if len(rows) == 0 {
		return nil
	}
	out := []observation{}
	for _, r := range rows {
		if r.signature == rows[0].signature {
			out = append(out, r)
			if len(out) == 20 {
				break
			}
		}
	}
	return out
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func SkillProfile(catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time) []SkillStatus {
	obs := observed(runs)
	out := []SkillStatus{}
	for _, skill := range skills {
		st := SkillStatus{Skill: skill, Priority: 1, Evidence: "缺少同一 benchmark 难度内的可比成绩；保持均衡。"}
		ranks := []float64{}
		for _, s := range catalog {
			if s.Skill != skill {
				continue
			}
			rows := obs[strings.ToLower(s.Name)]
			for _, r := range rows {
				if !r.at.Before(now.AddDate(0, 0, -7)) && !r.at.After(now) {
					st.Minutes += math.Max(0, r.duration) / 60
					st.Samples++
				}
			}
			for _, membership := range memberships(s) {
				if !selectedBenchmark(p, membership.Name) || len(membership.Thresholds) < 2 {
					continue
				}
				cr := comparable(rows)
				scores := []float64{}
				for _, r := range cr {
					if !r.at.Before(now.AddDate(0, 0, -30)) {
						scores = append(scores, r.score)
					}
				}
				if len(scores) < 3 {
					continue
				}
				m := median(scores)
				ts := append([]float64{}, membership.Thresholds...)
				sort.Float64s(ts)
				rank := 0.0
				for i, t := range ts {
					if m >= t {
						rank = float64(i + 1)
					} else {
						lo := 0.0
						if i > 0 {
							lo = ts[i-1]
						}
						if t > lo {
							rank += math.Max(0, (m-lo)/(t-lo))
						}
						break
					}
				}
				ranks = append(ranks, math.Min(1, rank/float64(len(ts))))
			}
		}
		if len(ranks) > 0 {
			sum := 0.0
			for _, r := range ranks {
				sum += r
			}
			st.Priority = 1 + 2*(1-sum/float64(len(ranks)))
			st.Evidence = fmt.Sprintf("依据所选 benchmark 难度内 %d 张图的近期中位成绩；类别比较为初步估计。", len(ranks))
		}
		out = append(out, st)
	}
	return out
}

func selectedBenchmark(p Preferences, name string) bool {
	if name == "" {
		return false
	}
	if p.Benchmark == name {
		return true
	}
	for _, b := range p.Benchmarks {
		if b == name {
			return true
		}
	}
	return false
}

// Rotate among selected systems using measured runs; equal usage is settled by
// seeded randomness, so consecutive plans can include different benchmarks.
func chooseBenchmark(catalog []Scenario, obs map[string][]observation, p Preferences, now time.Time, budget int, rng *rand.Rand) string {
	counts := map[string]int{}
	for _, s := range catalog {
		if !s.Enabled || estimateTiming(s, obs[strings.ToLower(s.Name)], now).Seconds > budget || (p.Difficulty != "any" && s.Difficulty != "unknown" && s.Difficulty != p.Difficulty) {
			continue
		}
		for _, m := range memberships(s) {
			if !selectedBenchmark(p, m.Name) {
				continue
			}
			if _, ok := counts[m.Name]; !ok {
				counts[m.Name] = 0
			}
			for _, r := range obs[strings.ToLower(s.Name)] {
				if !r.at.Before(now.AddDate(0, 0, -7)) && !r.at.After(now) {
					counts[m.Name]++
				}
			}
		}
	}
	options := []string{}
	least := int(^uint(0) >> 1)
	for name, count := range counts {
		if count < least {
			least = count
			options = []string{name}
		} else if count == least {
			options = append(options, name)
		}
	}
	if len(options) == 0 {
		return ""
	}
	sort.Strings(options)
	return options[rng.Intn(len(options))]
}

func variantFor(s Scenario, catalog []Scenario, benchmark string) bool {
	if s.VariantOf == "" || benchmark == "" {
		return false
	}
	for _, original := range catalog {
		if strings.EqualFold(original.Name, s.VariantOf) {
			_, ok := member(original, benchmark)
			return ok
		}
	}
	return false
}

func familyKey(s Scenario) string {
	if s.VariantOf != "" {
		return strings.ToLower(s.VariantOf)
	}
	if s.Family != "" {
		return strings.ToLower(s.Family)
	}
	return strings.ToLower(s.Name)
}

func validatePreferences(p Preferences) error {
	if p.ExecutionMode != "playlist" && p.ExecutionMode != "adaptive" {
		return fmt.Errorf("无效执行方式")
	}
	if p.Minutes < 5 || p.Minutes > 120 {
		return fmt.Errorf("训练预算须为 5–120 分钟")
	}
	if p.Focus != "auto" && !validSkill(p.Focus) {
		return fmt.Errorf("无效的训练重点")
	}
	if p.Difficulty != "any" && p.Difficulty != "novice" && p.Difficulty != "intermediate" && p.Difficulty != "advanced" {
		return fmt.Errorf("无效的难度")
	}
	if math.IsNaN(p.Variety) || p.Variety < 0 || p.Variety > 0.5 {
		return fmt.Errorf("变化比例须在 0–50%% 之间")
	}
	if math.IsNaN(p.ThresholdRatio) || p.ThresholdRatio < 0.5 || p.ThresholdRatio > 1 {
		return fmt.Errorf("阈值比例须在 50–100%% 之间")
	}
	if len(p.Benchmarks) > 20 {
		return fmt.Errorf("最多选择 20 套 benchmark")
	}
	return nil
}

func Generate(catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time, rng *rand.Rand) (*Plan, error) {
	if p.ExecutionMode == "" {
		p.ExecutionMode = "playlist"
	}
	if err := validatePreferences(p); err != nil {
		return nil, err
	}
	obs := observed(runs)
	profile := SkillProfile(catalog, runs, p, now)
	priority := map[string]float64{}
	minutes := map[string]float64{}
	for _, s := range profile {
		priority[s.Skill] = s.Priority
		minutes[s.Skill] = s.Minutes
	}
	pool := []Scenario{}
	for _, s := range catalog {
		if s.Enabled && validSkill(s.Skill) && (p.Difficulty == "any" || s.Difficulty == "unknown" || s.Difficulty == p.Difficulty) {
			pool = append(pool, s)
		}
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("没有可编排关卡。请先发现/导入列表，或调整分类和难度。")
	}
	plan := &Plan{ID: fmt.Sprintf("%d-%x", now.UnixMilli(), rng.Uint32()), Created: now.Format(time.RFC3339), Preferences: p, Status: "draft", Blocks: []Block{}, Warnings: []string{}, Seen: []string{}}
	// Reserve 10%% for loading and breaks. A block is a time cap, not a promise of repetitions.
	seconds := p.Minutes * 60
	usable := int(float64(seconds) * 0.9)
	timings := map[string]TimingEstimate{}
	for _, scenario := range pool {
		timings[scenario.Name] = estimateTiming(scenario, obs[strings.ToLower(scenario.Name)], now)
	}
	remaining := usable
	selected := chooseBenchmark(pool, obs, p, now, usable, rng)
	reserved := 0
	for _, scenario := range pool {
		if _, ok := member(scenario, selected); ok {
			d := timings[scenario.Name].Seconds
			if d <= usable && (reserved == 0 || d < reserved) { reserved = d }
		}
	}
	if reserved == 0 { selected = "" }
	if selected == "" && (p.Benchmark != "" || len(p.Benchmarks) > 0) {
		plan.Warnings = append(plan.Warnings, "所选 benchmark 中没有符合当前时长和难度的测量关卡；本次不插入测量。")
	}
	chosen := map[string]bool{}
	families := map[string]bool{}
	for i := 0; i < len(pool); i++ {
		role := "practice"
		if i == 0 {
			role = "warmup"
		}
		if i == len(pool)-1 && selected != "" {
			role = "benchmark"
		}
		if role == "practice" && rng.Float64() < p.Variety {
			role = "explore"
		}
		budget := remaining - reserved
		if role == "benchmark" { budget = remaining }
		best := -1
		bestKey := math.Inf(1)
		for j, s := range pool {
			if chosen[s.Name] {
				continue
			}
			_, isBenchmark := member(s, selected)
			if role == "benchmark" && !isBenchmark {
				continue
			}
			if role != "benchmark" && isBenchmark {
				continue
			}
			dur := timings[s.Name].Seconds
			rows := comparable(obs[strings.ToLower(s.Name)])
			if dur > budget {
				continue
			}
			w := priority[s.Skill] / (1 + minutes[s.Skill]/30)
			if p.Focus != "auto" {
				if s.Skill == p.Focus {
					w *= 8
				} else {
					w *= 0.25
				}
			}
			recent := 0
			for _, r := range obs[strings.ToLower(s.Name)] {
				if !r.at.Before(now.AddDate(0, 0, -7)) {
					recent++
				}
			}
			w /= 1 + float64(recent)*p.Variety
			if families[familyKey(s)] {
				w *= 0.25
			}
			if role == "practice" && (related(s, selected) || variantFor(s, pool, selected)) {
				w *= 3
			}
			if role == "warmup" {
				if len(rows) > 0 {
					w *= 4
				}
				if s.Difficulty == "novice" {
					w *= 2
				}
			}
			if role == "explore" {
				if len(rows) == 0 {
					w *= 5
				} else {
					w *= 0.3
				}
			}
			if s.Classification == "manual" || s.Classification == "benchmark" {
				w *= 1.5
			}
			switch s.Preference {
			case "liked":
				w *= 1.5
			case "disliked":
				w *= 0.1
			}
			switch s.PersonalDifficulty {
			case "hard":
				if role == "warmup" {
					w *= 0.15
				} else {
					w *= 0.6
				}
			case "easy":
				if role == "warmup" {
					w *= 1.3
				} else {
					w *= 0.75
				}
			}
			key := -math.Log(math.Max(rng.Float64(), 1e-9)) / w
			if key < bestKey {
				bestKey = key
				best = j
			}
		}
		if best < 0 {
			if role == "benchmark" {
				plan.Warnings = append(plan.Warnings, "所选 benchmark 没有适合本次时长/难度且未重复的关卡，未强行插入。")
			}
			if selected != "" && role != "benchmark" {
				i = len(pool)-2 // Try the reserved measurement once, then finish.
				continue
			}
			break
		}
		s := pool[best]
		chosen[s.Name] = true
		families[familyKey(s)] = true
		timing := timings[s.Name]
		dur := timing.Seconds
		count := plannedRepetitions(timing, role)
		if count > budget/dur { count = budget/dur }
		blockBudget := count * dur
		remaining -= blockBudget
		b := Block{Scenario: s, Timing: timing, Role: role, Budget: blockBudget, PlayCount: count, Outcome: "pending", Reason: "匹配能力与难度，并降低近期重复；时长为上限。", Cue: "留意动作质量；本模块到时即可继续，不要求无限重开。"}
		if p.ExecutionMode == "playlist" {
			b.Reason = fmt.Sprintf("匹配能力与难度，并降低近期重复；游戏内列表安排 %d 局，次数根据单局长度与近 24 小时、近 7 天的已记录练习量估算。", count)
		}
		if related(s, selected) && role == "practice" {
			b.Reason += " 手动关联至本次参考 benchmark 的能力训练。"
		}
		if variantFor(s, pool, selected) && role == "practice" {
			b.Reason += " 这是本次参考测量关卡的已标注直接变体。"
		}
		if role == "benchmark" {
			b.Benchmark = selected
		}
		if s.Classification == "inferred" {
			b.Reason += " 分类由名称推断，可在关卡库修正。"
		}
		if s.Difficulty == "unknown" {
			b.Reason += " 难度尚未核验。"
		}
		rows := comparable(obs[strings.ToLower(s.Name)])
		if p.ExecutionMode == "adaptive" && role == "practice" && len(rows) >= 5 {
			bestScore := 0.0
			for _, r := range rows {
				bestScore = math.Max(bestScore, r.score)
			}
			b.Target = bestScore * p.ThresholdRatio
			b.Signature = rows[0].signature
			b.Reason += " 采用同版本/设置近期最佳成绩的个人阈值；这是限时改造版。"
		}
		if role == "benchmark" {
			b.Target = 0
			b.Reason = "固定完成一次并记录成绩；不采用达标后才记分的规则。"
		}
		if role == "explore" {
			b.Reason = "探索内容，不设成绩阈值；先建立个人基线。"
		}
		if s.Skill == "smooth" {
			b.Cue = "观察长横移是否连续匹配速度，而不是反复停顿追赶。参考 MattyOW Speed Matching；不据轨迹断言握力。"
		}
		if s.Skill == "reactive" {
			b.Cue = "观察换向后的恢复，并区分实际换向与提前反向。需要结合目标录像判断。"
		}
		if timing.Source == "history" {
			b.Reason += fmt.Sprintf(" 单局约 %d 秒，取最近 %d 条同版本/设置记录的中位数。", dur, timing.Samples)
		} else {
			b.Reason += fmt.Sprintf(" 可比历史不足 3 局，单局暂按关卡库或默认时长预估 %d 秒。", dur)
		}
		plan.Blocks = append(plan.Blocks, b)
	}
	if len(plan.Blocks) == 0 {
		return nil, fmt.Errorf("可用关卡时长均超出模块预算，请增加训练时间")
	}
	if remaining > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("本次另有 %d 秒未分配：保持完整局数和少量重复，不为填满时间延长单关。", remaining))
	}
	plan.Warnings = append(plan.Warnings, "预留约 10% 时间用于休息与切换；未知关卡默认按 60 秒估计。", "有效练习量来自完成并写入记录的对局；未写出的中途重开不能精确统计。")
	if p.ExecutionMode == "playlist" {
		plan.Warnings = append(plan.Warnings, "游戏内列表使用固定次数；总预算到时，工作台停止计时但不会中断 KovaaK’s，请手动结束列表。")
	}
	if p.Benchmark == "" && len(p.Benchmarks) == 0 && p.Focus == "auto" {
		plan.Warnings = append(plan.Warnings, "尚未选择参考 benchmark；本次按周覆盖与历史重复均衡选图，不宣称已经诊断弱项。")
	}
	return plan, nil
}
