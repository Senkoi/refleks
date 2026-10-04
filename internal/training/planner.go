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
	accuracy        float64
	accuracyKnown   bool
	hitsPerSecond   float64
	speedKnown      bool
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
		if s.Score < 0 || math.IsNaN(s.Score) || math.IsInf(s.Score, 0) || s.Duration <= 0 || s.Duration > 3600 || math.IsNaN(s.Duration) || math.IsInf(s.Duration, 0) {
			continue
		}
		sig := signature(s)
		key := strings.ToLower(s.Scenario)
		a := s.Accuracy
		if a > 1 {
			a /= 100
		}
		known := a > 0 && a <= 1 && !math.IsNaN(a) && !math.IsInf(a, 0)
		if s.HitCount >= 0 && s.MissCount >= 0 && int64(s.HitCount)+int64(s.MissCount) > 0 {
			a = float64(s.HitCount) / (float64(s.HitCount) + float64(s.MissCount))
			known = true
		}
		out[key] = append(out[key], observation{score: s.Score, duration: s.Duration, at: t, signature: sig, accuracy: a, accuracyKnown: known, hitsPerSecond: float64(s.HitCount) / s.Duration, speedKnown: s.HitCount > 0})
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
				m, samples := recentLevelScore(comparable(rows), now)
				if samples < 3 {
					continue
				}
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
		if s.Mechanics != nil && s.Mechanics.Role == "warmup" {
			continue
		}
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
	if p.PlanningPolicy != "" && p.PlanningPolicy != "curriculum" && p.PlanningPolicy != "legacy" {
		return fmt.Errorf("无效编排策略")
	}
	if p.ExecutionMode != "playlist" && p.ExecutionMode != "adaptive" {
		return fmt.Errorf("无效执行方式")
	}
	if p.Minutes < 5 || p.Minutes > 120 {
		return fmt.Errorf("训练预算须为 5–120 分钟")
	}
	if p.Focus != "auto" && !validSkill(p.Focus) {
		return fmt.Errorf("无效的训练重点")
	}
	if p.Difficulty != "any" && p.Difficulty != "entry" && p.Difficulty != "novice" && p.Difficulty != "adept" && p.Difficulty != "intermediate" && p.Difficulty != "advanced" && p.Difficulty != "elite" {
		return fmt.Errorf("无效的难度")
	}
	if math.IsNaN(p.Variety) || p.Variety < 0 || p.Variety > 0.5 {
		return fmt.Errorf("变化比例须在 0–50%% 之间")
	}
	if math.IsNaN(p.ThresholdRatio) || p.ThresholdRatio < 0.5 || p.ThresholdRatio > 1 {
		return fmt.Errorf("阈值比例须在 50–100%% 之间")
	}
	// Benchmark references are resolved automatically from the bounded catalog.
	// A former manual selection limit must not reject automatic initialization.
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
			pool = append(pool, enrichMechanics(s))
		}
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("没有可编排关卡。请先发现/导入列表，或调整分类和难度。")
	}
	plan := &Plan{ID: fmt.Sprintf("%d-%x", now.UnixMilli(), rng.Uint32()), Created: now.Format(time.RFC3339), Preferences: p, Status: "draft", Blocks: []Block{}, Warnings: []string{}, Seen: []string{}}
	if p.Focus == "auto" {
		plan.Theme = chooseTheme(pool, obs, now)
	}
	// Reserve 10%% for loading and breaks. A block is a time cap, not a promise of repetitions.
	seconds := p.Minutes * 60
	usable := int(float64(seconds) * 0.9)
	timings := map[string]TimingEstimate{}
	difficulties := map[string]DifficultyEvidence{}
	for _, scenario := range pool {
		timings[scenario.Name] = estimateTiming(scenario, obs[strings.ToLower(scenario.Name)], now)
		difficulties[scenario.Name] = assessDifficultyFor(scenario, obs[strings.ToLower(scenario.Name)], now, p)
	}
	remaining := usable
	warmupRemaining := usable / 2
	practiceRemaining := usable - warmupRemaining
	warming := true
	exposure := map[string]float64{}
	for _, s := range pool {
		addDemandTime(s, timings[s.Name].WeeklySeconds, exposure)
	}
	benchmarkPool := pool
	if plan.Theme != "" {
		focused := []Scenario{}
		for _, scenario := range pool {
			if scenarioTheme(scenario) == plan.Theme {
				focused = append(focused, scenario)
			}
		}
		if chooseBenchmark(focused, obs, p, now, practiceRemaining, rand.New(rand.NewSource(1))) != "" {
			benchmarkPool = focused
		}
	}
	selected := chooseBenchmark(benchmarkPool, obs, p, now, practiceRemaining, rng)
	reserved := 0
	for _, scenario := range pool {
		if _, ok := member(scenario, selected); ok && (scenario.Mechanics == nil || scenario.Mechanics.Role != "warmup") {
			d := timings[scenario.Name].Seconds
			if d <= practiceRemaining && (reserved == 0 || d < reserved) {
				reserved = d
			}
		}
	}
	if reserved == 0 {
		selected = ""
	}
	if selected == "" && (p.Benchmark != "" || len(p.Benchmarks) > 0) {
		plan.Warnings = append(plan.Warnings, "所选 benchmark 中没有符合当前时长和难度的测量关卡；本次不插入测量。")
	}
	chosen := map[string]bool{}
	families := map[string]bool{}
	for i := 0; i < len(pool)+2; i++ {
		role := "practice"
		if warming {
			role = "warmup"
		}
		if role == "practice" && rng.Float64() < p.Variety {
			role = "explore"
		}
		budget := practiceRemaining - reserved
		if warming {
			budget = warmupRemaining
		}
		// If an eligible lower-pressure scene fits, do not stack pressure
		// scenes or use them for preparation. Unknown is not "easy".
		avoidPressure := false
		if role != "benchmark" && (warming || (len(plan.Blocks) > 0 && pressureDemand(plan.Blocks[len(plan.Blocks)-1].Scenario))) {
			for _, candidate := range pool {
				_, measured := member(candidate, selected)
				if !chosen[candidate.Name] && !measured && !pressureDemand(candidate) && timings[candidate.Name].Seconds <= budget && (warming || candidate.Mechanics == nil || candidate.Mechanics.Role != "warmup") && len(demandTags(candidate)) > 0 {
					avoidPressure = true
					break
				}
			}
		}
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
			if !warming && s.Mechanics != nil && s.Mechanics.Role == "warmup" {
				continue
			}
			if avoidPressure && pressureDemand(s) {
				continue
			}
			dur := timings[s.Name].Seconds
			rows := comparable(obs[strings.ToLower(s.Name)])
			if dur > budget {
				continue
			}
			w := priority[s.Skill] / (1 + minutes[s.Skill]/30)
			w *= demandWeight(s, exposure)
			if len(plan.Blocks) > 0 && role != "benchmark" {
				w *= 1 - .7*demandSimilarity(s, plan.Blocks[len(plan.Blocks)-1].Scenario)
			}
			if plan.Theme != "" && role == "practice" {
				if scenarioTheme(s) == plan.Theme {
					w *= 7
				} else {
					w *= .3
				}
			}
			evidence := difficulties[s.Name]
			switch evidence.Fit {
			case "challenging":
				w *= .45
			case "suitable":
				w *= 1.4
			case "comfortable":
				w *= .8
			}
			// Progress from easier preparation to more demanding practice, but
			// never treat unknown tiers as a proven order.
			if role == "warmup" {
				if evidence.Level == "novice" {
					w *= 2
				}
				if evidence.Level == "advanced" {
					w *= .25
				}
			} else if role == "practice" && evidence.Level == "advanced" && len(plan.Blocks) < 3 {
				w *= .35
			}
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
			if warming {
				warming = false
				if warmupRemaining > 0 {
					plan.Warnings = append(plan.Warnings, fmt.Sprintf("热身半程有 %d 秒未填满：候选不足或完整单局放不下；保留完整局数。", warmupRemaining))
				}
				continue
			}
			if role == "benchmark" {
				plan.Warnings = append(plan.Warnings, "所选 benchmark 没有适合本次时长/难度且未重复的关卡，未强行插入。")
			}
			if selected != "" && role != "benchmark" {
				// Append the reserved fixed measurement after both phases.
				role = "benchmark"
				for j, candidate := range pool {
					_, measured := member(candidate, selected)
					if measured && !chosen[candidate.Name] && timings[candidate.Name].Seconds <= practiceRemaining && (candidate.Mechanics == nil || candidate.Mechanics.Role != "warmup") {
						best = j
						break
					}
				}
				if best < 0 {
					break
				}
				budget = practiceRemaining
			} else {
				break
			}
		}
		s := pool[best]
		chosen[s.Name] = true
		families[familyKey(s)] = true
		timing := timings[s.Name]
		dur := timing.Seconds
		count := plannedRepetitions(timing, role)
		if count > budget/dur {
			count = budget / dur
		}
		// Pressure is a planning precaution, not a fatigue diagnosis.
		if pressureDemand(s) {
			count = 1
		}
		blockBudget := count * dur
		remaining -= blockBudget
		if warming {
			warmupRemaining -= blockBudget
		} else {
			practiceRemaining -= blockBudget
		}
		addDemandTime(s, float64(blockBudget), exposure)
		b := Block{Scenario: s, Timing: timing, DifficultyEvidence: difficulties[s.Name], Role: role, Budget: blockBudget, PlayCount: count, Outcome: "pending", Reason: "匹配能力与难度，并降低近期重复；时长为上限。", Cue: "留意动作质量；本模块到时即可继续，不要求无限重开。"}
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
		if plan.Theme != "" && scenarioTheme(s) == plan.Theme && role == "practice" {
			b.Reason += " 本次自动轮换到该 VDIM 专项，优先安排同类练习。"
		}
		if b.DifficultyEvidence.Fit != "unknown" {
			b.Reason += fmt.Sprintf(" 个人适配评估为 %s；依据该关卡可比成绩或你的反馈，不等同于全体玩家难度。", b.DifficultyEvidence.Fit)
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
		b.Reason += mechanicsReason(s)
		plan.Blocks = append(plan.Blocks, b)
		if role == "benchmark" {
			break
		}
	}
	if len(plan.Blocks) == 0 {
		return nil, fmt.Errorf("可用关卡时长均超出模块预算，请增加训练时间")
	}
	if remaining > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("本次另有 %d 秒未分配：保持完整局数和少量重复，不为填满时间延长单关。", remaining))
	}
	if plan.Theme != "" {
		plan.Warnings = append(plan.Warnings, "按 VDIM 六类专项轮换并参考近七天完成记录；这是可变时长的改造编排，不是原作者的原样列表。")
	}
	plan.Warnings = append(plan.Warnings, "预留约 10% 时间用于休息与切换；未知关卡默认按 60 秒估计。", "有效练习量来自完成并写入记录的对局；未写出的中途重开不能精确统计。")
	plan.Warnings = append(plan.Warnings, "前半程预算用于热身，后半程用于专项、探索和固定测量；细分维度用于覆盖与负荷控制，尚不据此诊断细分弱项或计算统一难度。")
	if p.ExecutionMode == "playlist" {
		plan.Warnings = append(plan.Warnings, "游戏内列表使用固定次数；总预算到时，工作台停止计时但不会中断 KovaaK’s，请手动结束列表。")
	}
	if p.Benchmark == "" && len(p.Benchmarks) == 0 && p.Focus == "auto" {
		plan.Warnings = append(plan.Warnings, "尚未选择参考 benchmark；本次按周覆盖与历史重复均衡选图，不宣称已经诊断弱项。")
	}
	return plan, nil
}
