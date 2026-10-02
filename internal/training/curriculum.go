package training

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"refleks/internal/models"
)

type CurriculumRow struct {
	Name        string `json:"scenarioName"`
	Count       int    `json:"playCount"`
	SourceCount int    `json:"sourcePlayCount,omitempty"`
	Role        string `json:"role,omitempty"`
}

type Curriculum struct {
	OfficialCode  string          `json:"officialCode,omitempty"`
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Theme         string          `json:"theme"`
	Tier          string          `json:"tier,omitempty"`
	Source        Source          `json:"source"`
	ContentSHA256 string          `json:"contentSHA256"`
	Rows          []CurriculumRow `json:"rows"`
}

var vdimCanonicalTitle = regexp.MustCompile(`(?i)^(entry|novice|adept|intermediate|advanced|elite)\s+s5\s*[-–]\s*(clicking|tracking|switching)\s+(i|ii)$`)

// Only actual structured playlists are templates. Sharecodes and alternate
// engine ports cannot manufacture the author's scene order or repetitions.
func parseCurriculum(data []byte, src Source) (*Curriculum, error) {
	var p struct {
		Name string          `json:"playlistName"`
		Rows []CurriculumRow `json:"scenarioList"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if !strings.Contains(strings.ToLower(p.Name+" "+src.Title+" "+src.URL), "vdim") && !vdimCanonicalTitle.MatchString(strings.TrimSpace(p.Name)) {
		return nil, nil
	}
	if len(p.Rows) == 0 || len(p.Rows) > 2000 {
		return nil, fmt.Errorf("VDIM 模板关卡数量无效")
	}
	for i := range p.Rows {
		r := &p.Rows[i]
		r.SourceCount = 0 // Source JSON cannot manufacture an adaptive completion proof.
		if strings.TrimSpace(r.Name) == "" || len(r.Name) > 300 || strings.ContainsAny(r.Name, "\r\n\x00") {
			return nil, fmt.Errorf("VDIM 模板含无效场景名")
		}
		if r.Count == 0 {
			r.Count = 1
		}
		if r.Count < 1 || r.Count > 100 {
			return nil, fmt.Errorf("VDIM 模板重复次数无效")
		}
		if r.Role != "" && r.Role != "warmup" && r.Role != "practice" && r.Role != "benchmark" {
			return nil, fmt.Errorf("VDIM 模板角色无效")
		}
	}
	identity := sha256.Sum256([]byte(src.URL + "\x00" + p.Name))
	content := sha256.Sum256(data)
	name := strings.ToLower(p.Name)
	theme := classify(p.Name)
	switch {
	case strings.Contains(name, "clicking ii"):
		theme = "dynamic"
	case strings.Contains(name, "clicking i"):
		theme = "static"
	case strings.Contains(name, "tracking ii"):
		theme = "reactive"
	case strings.Contains(name, "tracking i"):
		theme = "smooth"
	case strings.Contains(name, "switching ii"):
		theme = "switching_evasive"
	case strings.Contains(name, "switching i"):
		theme = "switching_speed"
	}
	tier := ""
	for _, t := range []string{"entry", "novice", "adept", "intermediate", "advanced", "elite"} {
		if strings.Contains(name, t) {
			tier = t
			break
		}
	}
	return &Curriculum{ID: hex.EncodeToString(identity[:12]), Name: p.Name, Theme: theme, Tier: tier, Source: src, ContentSHA256: hex.EncodeToString(content[:]), Rows: p.Rows}, nil
}

func (s *Service) mergeCurricula(items []Scenario) {
	for _, item := range items {
		if item.ImportedCurriculum == nil {
			continue
		}
		t := *item.ImportedCurriculum
		found := false
		for i := range s.state.Curricula {
			if s.state.Curricula[i].ID == t.ID {
				s.state.Curricula[i] = t
				found = true
				break
			}
		}
		if !found && len(s.state.Curricula) < 100 {
			s.state.Curricula = append(s.state.Curricula, t)
		}
	}
}

func selectCurriculum(ts []Curriculum, catalog []Scenario, p Preferences, runs []models.RunRecord, now time.Time) (*Curriculum, error) {
	return selectCurriculumWithHistory(ts, catalog, p, runs, now, nil)
}

func selectCurriculumWithHistory(ts []Curriculum, catalog []Scenario, p Preferences, runs []models.RunRecord, now time.Time, history []Plan) (*Curriculum, error) {
	// Explicit template selection takes precedence over automatic theme/tier
	// filters. GenerateCurriculum still validates its contents and full budget.
	if p.CurriculumID != "" {
		for i := range ts {
			if ts[i].ID == p.CurriculumID {
				return &ts[i], nil
			}
		}
		return nil, fmt.Errorf("所选 VDIM 模板不存在")
	}
	eligible := []Curriculum{}
	matched := false
	for _, t := range ts {
		if p.Focus != "auto" && t.Theme != p.Focus && !(p.Focus == "switching" && strings.HasPrefix(t.Theme, "switching_")) {
			continue
		}
		if p.Difficulty != "any" && t.Tier != "" && t.Tier != p.Difficulty {
			continue
		}
		matched = true
		// Reuse generation's timing and validation rather than picking an
		// oversized/unavailable template and failing while another one fits.
		window, _, _, windowErr := curriculumWindow(t, catalog, runs, p, now, history)
		if windowErr != nil {
			continue
		}
		if _, err := GenerateCurriculum(window, catalog, runs, p, now, rand.New(rand.NewSource(0)), false); err != nil {
			continue
		}
		eligible = append(eligible, t)
	}
	if len(eligible) == 0 {
		if matched {
			return nil, fmt.Errorf("符合重点与档位的 VDIM 模板均无法在当前预算内完整运行，或含不可用场景；请增加可用时间或检查禁用的场景")
		}
		return nil, fmt.Errorf("尚无符合条件的真实 VDIM 模板；自动初始化未完成，请检查初始化错误")
	}
	levels := PlayerLevels(catalog, runs, now)
	bestDistance := map[string]int{}
	distance := func(t Curriculum) int {
		d := tierIndex(t.Tier) - tierIndex(inferredTier(t.Theme, levels))
		if d < 0 {
			return -d
		}
		return d * 10
	}
	for _, t := range eligible {
		d := distance(t)
		old, ok := bestDistance[t.Theme]
		if !ok || d < old {
			bestDistance[t.Theme] = d
		}
	}
	fitted := []Curriculum{}
	for _, t := range eligible {
		if distance(t) == bestDistance[t.Theme] {
			fitted = append(fitted, t)
		}
	}
	eligible = fitted
	obs := observed(runs)
	score := func(t Curriculum) float64 {
		v := 0.0
		for _, scene := range catalog {
			if scenarioTheme(scene) != t.Theme {
				continue
			}
			for _, r := range obs[strings.ToLower(scene.Name)] {
				if !r.at.After(now) && !r.at.Before(now.AddDate(0, 0, -7)) {
					v += r.duration
				}
			}
		}
		return v
	}
	day := (int(now.Weekday()) + 6) % 7
	if day >= len(vdimThemes) {
		day = 0
	}
	preferred := vdimThemes[day]
	sort.SliceStable(eligible, func(i, j int) bool {
		a, b := score(eligible[i]), score(eligible[j])
		if a != b {
			return a < b
		}
		if (eligible[i].Theme == preferred) != (eligible[j].Theme == preferred) {
			return eligible[i].Theme == preferred
		}
		if (eligible[i].OfficialCode != "") != (eligible[j].OfficialCode != "") {
			return eligible[i].OfficialCode != ""
		}
		return eligible[i].Name < eligible[j].Name
	})
	return &eligible[0], nil
}

// Completed means every original row was actually recorded. Manual finish,
// skipped rows and time caps do not establish a full foundation session.
// Compare rows/counts too, so an edited template requires its own baseline.
func completedCurriculum(p Plan, t Curriculum) bool {
	if p.CurriculumHash != "" && t.ContentSHA256 != "" && p.CurriculumHash != t.ContentSHA256 {
		return false
	}
	if p.CurriculumID != t.ID || p.Status != "completed" || len(p.Blocks) < len(t.Rows) || len(t.Rows) == 0 {
		return false
	}
	for i, row := range t.Rows {
		b := p.Blocks[i]
		role := row.Role
		if role == "" {
			role = "practice"
		}
		sourceMatches := b.PlayCount == row.Count
		if b.SourcePlayCount > 0 {
			sourceMatches = b.SourcePlayCount == originalRowCount(row) && b.PlayCount > 0 && b.PlayCount <= b.SourcePlayCount
		}
		if !strings.EqualFold(b.Scenario.Name, row.Name) || b.Role != role || !sourceMatches || b.Runs < b.PlayCount || b.Recorded <= 0 || b.Outcome != "list_complete" {
			return false
		}
	}
	return true
}

func goalCompatible(anchor, candidate Scenario) bool {
	if !candidate.Enabled || !validSkill(anchor.Skill) || anchor.Skill != candidate.Skill {
		return false
	}
	if scenarioTheme(anchor) != scenarioTheme(candidate) {
		return false
	}
	if candidate.Mechanics != nil && candidate.Mechanics.Role == "warmup" {
		return false
	}
	// Extra dimensions are allowed, but known original demands and constraints
	// cannot disappear. Unknown mechanics do not prove goal compatibility.
	if anchor.Mechanics != nil && len(anchor.Mechanics.Tags) > 0 {
		for _, tag := range anchor.Mechanics.Tags {
			if !hasDemand(candidate, tag) {
				return false
			}
		}
		return true
	}
	return strings.EqualFold(candidate.VariantOf, anchor.Name)
}

func GenerateCurriculum(t Curriculum, catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time, rng *rand.Rand, explored bool) (*Plan, error) {
	if p.ExecutionMode == "" {
		p.ExecutionMode = "playlist"
	}
	if err := validatePreferences(p); err != nil {
		return nil, err
	}
	obs := observed(runs)
	index := map[string]Scenario{}
	for _, s := range catalog {
		index[strings.ToLower(s.Name)] = enrichMechanics(s)
	}
	plan := &Plan{PlayerTier: inferredTier(t.Theme, PlayerLevels(catalog, runs, now)), TemplateTier: t.Tier, PlannerVersion: currentPlannerVersion, ID: fmt.Sprintf("%d-%x", now.UnixMilli(), rng.Uint32()), Created: now.Format(time.RFC3339), Preferences: p, Status: "draft", CurriculumID: t.ID, CurriculumName: t.Name, CurriculumHash: t.ContentSHA256, Theme: t.Theme, Blocks: []Block{}, Warnings: []string{}, Seen: []string{}}
	used := 0
	baseNames := map[string]bool{}
	for _, row := range t.Rows {
		s, ok := index[strings.ToLower(row.Name)]
		if !ok {
			return nil, fmt.Errorf("模板场景 %s 不在关卡库", row.Name)
		}
		if !s.Enabled && s.Classification == "manual" {
			return nil, fmt.Errorf("模板场景 %s 已被你禁用；未强行加入或悄悄删改模板", row.Name)
		}
		timing := estimateTiming(s, obs[strings.ToLower(s.Name)], now)
		role := row.Role
		if role == "" {
			role = "practice"
		}
		b := Block{Scenario: s, Timing: timing, DifficultyEvidence: assessDifficultyFor(s, obs[strings.ToLower(s.Name)], now, p), Role: role, SourcePlayCount: originalRowCount(row), PlayCount: row.Count, Budget: timing.Seconds * row.Count, Outcome: "pending", Reason: "保留 VDIM 场景顺序和训练目标；按单局时长、近期重复量与预算分配短组。", Cue: "按原训练目标完成；下载后评估只影响下一次生成。"}
		if role == "benchmark" {
			b.Target = 0
		}
		plan.Blocks = append(plan.Blocks, b)
		used += b.Budget
		baseNames[strings.ToLower(s.Name)] = true
	}
	usable := p.Minutes * 60 * 9 / 10
	if used > usable {
		return nil, fmt.Errorf("保留模板顺序与次数需约 %d 分钟（含切换）；当前预算 %d 分钟不足，未截断或压缩原列表", (used*10+539)/540, p.Minutes)
	}
	// First generation remains exactly the real template. Exploration is an
	// add-on after at least one completed, recorded template session, never a
	// replacement of a foundation or measurement slot.
	if explored && p.Variety > 0 {
		levels := PlayerLevels(catalog, runs, now)
		p = automaticReferences(catalog, p)
		limit := int(float64(usable) * p.Variety)
		if limit > usable-used {
			limit = usable - used
		}
		exposure := map[string]float64{}
		for _, s := range catalog {
			addDemandTime(s, estimateTiming(s, obs[strings.ToLower(s.Name)], now).WeeklySeconds, exposure)
		}
		for _, base := range plan.Blocks {
			if base.Role == "benchmark" || base.Role == "warmup" {
				continue
			}
			best := -1
			bestWeight := -1.0
			for i, c := range catalog {
				c = enrichMechanics(c)
				if baseNames[strings.ToLower(c.Name)] || !goalCompatible(base.Scenario, c) {
					continue
				}
				_, aligned := explorationReference(c, levels, p)
				if len(memberships(c)) > 0 && !aligned {
					continue
				}
				// VDIM tier describes the template's audience, not the candidate's
				// mechanism difficulty. Personal feedback remains authoritative.
				if c.PersonalDifficulty == "hard" || c.Preference == "disliked" {
					continue
				}
				d := estimateTiming(c, obs[strings.ToLower(c.Name)], now).Seconds
				if d > limit {
					continue
				}
				w := demandWeight(c, exposure)/(1+float64(len(obs[strings.ToLower(c.Name)]))) + .001*rng.Float64()
				if aligned {
					w += 2
				}
				if w > bestWeight {
					best, bestWeight = i, w
				}
			}
			if best < 0 {
				continue
			}
			c := enrichMechanics(catalog[best])
			timing := estimateTiming(c, obs[strings.ToLower(c.Name)], now)
			plan.Blocks = append(plan.Blocks, Block{Scenario: c, Timing: timing, DifficultyEvidence: assessDifficultyFor(c, obs[strings.ToLower(c.Name)], now, p), AnchorScenario: base.Scenario.Name, Role: "explore", Budget: timing.Seconds, PlayCount: 1, Outcome: "pending", Reason: "保留原模板后额外探索；匹配原场景训练目标，允许新增细分维度；未评估难度保持未知。", Cue: "单次探索，不设分数阈值；结果只影响下一份列表。"})
			if reference, ok := explorationReference(c, levels, p); ok {
				b := &plan.Blocks[len(plan.Blocks)-1]
				b.Benchmark = reference.Name
				b.Reason = "保留原训练目标，自动插入最新可用 benchmark 的同分类、对应原生难度场景作为探索变体；不是额外的独立测量任务。"
			}
			baseNames[strings.ToLower(c.Name)] = true
			limit -= timing.Seconds
			addDemandTime(c, float64(timing.Seconds), exposure)
		}
	}
	plan.Warnings = append(plan.Warnings, "保留模板场景顺序；原次数作为上限，单局时长与近期训练量决定本次短组次数。", "首次模板运行不插入探索；后续只在剩余预算内追加同目标探索，当前列表不会实时改写。", "SCE 仅从游戏已保存的本地文件评估；没有文件时保持待评估，不额外下载场景。")
	return plan, nil
}
