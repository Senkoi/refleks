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

// Every fixed measurement has one familiarization run followed by two scored
// runs. Baseline and retest use the same protocol; none is a PB measurement.
type MeasurementSpec struct {
	StudyID string `json:"studyId"`
	Phase   string `json:"phase"`
}

type MeasurementResult struct {
	Score         float64  `json:"score"`
	Accuracy      *float64 `json:"accuracy,omitempty"`
	HitsPerSecond *float64 `json:"hitsPerSecond,omitempty"`
	Signature     string   `json:"signature"`
	Settings      string   `json:"settings"`
	FileSHA256    string   `json:"fileSHA256"`
	At            int64    `json:"at"`
	Samples       int      `json:"samples"`
}

type TrainingStudy struct {
	TransferContaminated bool               `json:"transferContaminated,omitempty"`
	ID                   string             `json:"id"`
	PlanID               string             `json:"planId"`
	Theme                string             `json:"theme"`
	AnchorScenario       string             `json:"anchorScenario"`
	TrainingScenario     string             `json:"trainingScenario"`
	TransferScenario     string             `json:"transferScenario,omitempty"`
	AnchorHash           string             `json:"anchorHash"`
	TrainingHash         string             `json:"trainingHash"`
	TransferHash         string             `json:"transferHash,omitempty"`
	Relation             PrecisionRelation  `json:"relation"`
	Status               string             `json:"status"`
	Feedback             string             `json:"feedback,omitempty"`
	CreatedAt            int64              `json:"createdAt"`
	TrainedAt            int64              `json:"trainedAt,omitempty"`
	DueAt                int64              `json:"dueAt,omitempty"`
	ExpiresAt            int64              `json:"expiresAt,omitempty"`
	Baseline             *MeasurementResult `json:"baseline,omitempty"`
	Trial                *MeasurementResult `json:"trial,omitempty"`
	Retest               *MeasurementResult `json:"retest,omitempty"`
	TransferBaseline     *MeasurementResult `json:"transferBaseline,omitempty"`
	TransferRetest       *MeasurementResult `json:"transferRetest,omitempty"`
	RetentionChange      *float64           `json:"retentionChange,omitempty"`
	TransferChange       *float64           `json:"transferChange,omitempty"`
}

type ResponsePrediction struct {
	Status        string  `json:"status"`
	ExpectedScore float64 `json:"expectedScore,omitempty"`
	Samples       int     `json:"samples"`
	Days          int     `json:"days"`
	ValidationMAE float64 `json:"validationMAE,omitempty"`
	BaselineMAE   float64 `json:"baselineMAE,omitempty"`
}

func measured(samples []PracticeSample, warmup, count int, hash string) *MeasurementResult {
	if len(samples) < warmup+count {
		return nil
	}
	data := samples[warmup : warmup+count]
	scores, accuracy, speed := []float64{}, []float64{}, []float64{}
	seen := map[string]bool{}
	var previous int64
	for _, p := range samples[:warmup+count] {
		if seen[p.RunID] || p.Signature != data[0].Signature || p.FileSHA256 != hash || hash == "" {
			return nil
		}
		if p.At <= previous {
			return nil
		}
		previous = p.At
		seen[p.RunID] = true
	}
	for _, p := range data {
		scores = append(scores, p.Score)
		if p.Accuracy != nil {
			accuracy = append(accuracy, *p.Accuracy)
		}
		if p.HitsPerSecond != nil {
			speed = append(speed, *p.HitsPerSecond)
		}
	}
	r := &MeasurementResult{Score: median(scores), Signature: data[0].Signature, Settings: data[0].Settings, FileSHA256: hash, At: data[len(data)-1].At, Samples: count}
	if len(accuracy) == count {
		v := median(accuracy)
		r.Accuracy = &v
	}
	if len(speed) == count {
		v := median(speed)
		r.HitsPerSecond = &v
	}
	return r
}

func responsePrediction(studies []TrainingStudy, anchor PersonalAnchor, relation *PrecisionRelation) *ResponsePrediction {
	result := &ResponsePrediction{Status: "insufficient"}
	if relation == nil || !relation.Uniform || anchor.MedianScore <= 0 || anchor.FileSHA256 == "" {
		return result
	}
	type point struct {
		x, y float64
		at   int64
	}
	points := []point{}
	days, contrasts, seen := map[string]bool{}, map[string]bool{}, map[string]bool{}
	nonPositive := false
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, st := range studies {
		if seen[st.ID] || st.Relation.FamilyFingerprint != relation.FamilyFingerprint || !st.Relation.Uniform || st.Baseline == nil || st.Trial == nil ||
			st.Baseline.Settings != st.Trial.Settings ||
			st.Baseline.Signature != anchor.Signature || st.AnchorScenario != anchor.Scenario || st.AnchorHash != anchor.FileSHA256 || st.AnchorHash != st.Baseline.FileSHA256 || st.TrainingHash != st.Trial.FileSHA256 {
			continue
		}
		if anchor.LastPlayed > 0 && (st.Trial.At < anchor.LastPlayed-int64(45*24*time.Hour/time.Millisecond) || st.Trial.At > anchor.LastPlayed+int64(24*time.Hour/time.Millisecond)) {
			continue
		}
		seen[st.ID] = true
		if st.Baseline.Score <= 0 || st.Trial.Score <= 0 {
			nonPositive = true // Keep failed trials; do not fit only surviving positive scores.
			continue
		}
		x := st.Relation.UniformDelta
		y := math.Log(st.Trial.Score / st.Baseline.Score)
		if !finite(x) || !finite(y) || math.Abs(x) < 1e-6 {
			continue
		}
		points = append(points, point{x, y, st.Trial.At})
		days[time.UnixMilli(st.Trial.At).Local().Format("2006-01-02")] = true
		contrasts[fmt.Sprintf("%.6f", x)] = true
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	result.Samples, result.Days = len(points), len(days)
	if nonPositive {
		result.Status = "non_positive"
		return result
	}
	if len(points) < 6 || len(days) < 3 || len(contrasts) < 2 {
		return result
	}
	if relation.UniformDelta < lo-1e-6 || relation.UniformDelta > hi+1e-6 {
		result.Status = "out_of_range"
		return result
	}
	sort.Slice(points, func(i, j int) bool { return points[i].at < points[j].at })
	// Hold out entire days, not neighboring retries. Fixed origin: a zero
	// feature difference predicts the anchor's score, not a free intercept.
	// Include earlier whole days until at least two held-out episodes exist.
	// Normal execution permits one trial per day while awaiting its retest.
	cut := len(points)
	for cut > 0 && len(points)-cut < 2 {
		lastDay := time.UnixMilli(points[cut-1].at).Local().Format("2006-01-02")
		for cut > 0 && time.UnixMilli(points[cut-1].at).Local().Format("2006-01-02") == lastDay {
			cut--
		}
	}
	if cut < 4 || len(points)-cut < 2 {
		result.Status = "unvalidated"
		return result
	}
	fit := func(data []point) float64 {
		xy, xx := 0.0, .1 // Small shrinkage; a product default, not a universal coefficient.
		for _, p := range data {
			xy += p.x * p.y
			xx += p.x * p.x
		}
		return xy / xx
	}
	beta := fit(points[:cut])
	for _, p := range points[cut:] {
		result.ValidationMAE += math.Abs(p.y - beta*p.x)
		result.BaselineMAE += math.Abs(p.y)
	}
	n := float64(len(points) - cut)
	result.ValidationMAE /= n
	result.BaselineMAE /= n
	if result.BaselineMAE <= .001 || result.ValidationMAE > result.BaselineMAE*.9 {
		result.Status = "baseline_better"
		return result
	}
	result.Status = "local_backtest"
	result.ExpectedScore = anchor.MedianScore * math.Exp(fit(points)*relation.UniformDelta)
	// This is a conditional personal response estimate, not a causal effect
	// of hitbox size, a global difficulty score or a calibrated interval.
	return result
}

type personalizationBundle struct {
	before, after []Block
	study         *TrainingStudy
}

func (b personalizationBundle) seconds() int {
	n := 0
	for _, block := range b.before {
		n += block.Budget
	}
	for _, block := range b.after {
		n += block.Budget
	}
	return n
}

func measurementBlock(st TrainingStudy, scene Scenario, phase string, obs map[string][]observation, now time.Time) Block {
	timing := estimateTiming(scene, obs[strings.ToLower(scene.Name)], now)
	return Block{Scenario: scene, Role: "assessment", Timing: timing, PlayCount: 3, Budget: 3 * timing.Seconds, Outcome: "pending",
		Measurement: &MeasurementSpec{st.ID, phase}, AnchorScenario: st.AnchorScenario,
		Reason: "固定前测/复测：第一局熟悉，后两局取中位成绩；版本、设置或时长变化时不合并比较。",
		Cue:    "按平常方式完成三局；第一局不计入测量，结果只影响后续计划。"}
}

func studyOpen(st TrainingStudy) bool {
	return st.Status == "planned" || st.Status == "waiting" || st.Status == "due" || st.Status == "transfer_due"
}

func reviewTheme(studies []TrainingStudy, now time.Time) string {
	theme := ""
	var earliest int64
	for _, st := range studies {
		if (st.Status == "due" || st.Status == "transfer_due") && st.DueAt <= now.UnixMilli() && st.ExpiresAt >= now.UnixMilli() && (earliest == 0 || st.DueAt < earliest) {
			theme, earliest = st.Theme, st.DueAt
		}
	}
	return theme
}

func preparePersonalization(t Curriculum, catalog []Scenario, runs []models.RunRecord, contexts map[string]RunContext, studies []TrainingStudy, now time.Time, p Preferences, explored bool, firstSeconds int, rng *rand.Rand) personalizationBundle {
	bundle := personalizationBundle{}
	obs := observed(runs)
	byName := map[string]Scenario{}
	for _, s := range catalog {
		byName[strings.ToLower(s.Name)] = enrichMechanics(s)
	}
	available := p.Minutes*60*9/10 - firstSeconds
	// Pending delayed tests have priority over new trials within the chosen
	// category, and never change the category of a 24-hour continuation.
	for _, st := range studies {
		if st.Theme != t.Theme || st.DueAt == 0 || st.DueAt > now.UnixMilli() || st.ExpiresAt < now.UnixMilli() || !studyOpen(st) {
			continue
		}
		for _, item := range []struct {
			name, hash, phase string
			done, baseline    *MeasurementResult
		}{
			{st.AnchorScenario, st.AnchorHash, "retest", st.Retest, st.Baseline},
			{st.TransferScenario, st.TransferHash, "transfer_retest", st.TransferRetest, st.TransferBaseline},
		} {
			if item.done != nil || item.baseline == nil {
				continue
			}
			scene, ok := byName[strings.ToLower(item.name)]
			if !ok || !scene.Enabled || scene.LocalAssessment == nil || scene.LocalAssessment.Status != "file_parsed_model_unfitted" || scene.LocalAssessment.FileSHA256 != item.hash {
				continue
			}
			block := measurementBlock(st, scene, item.phase, obs, now)
			if block.Budget <= available-bundle.seconds() {
				bundle.before = append(bundle.before, block)
			}
		}
		return bundle // At most one study per plan; postpone if a whole test cannot fit.
	}
	if !explored || p.Variety <= 0 || available <= 0 {
		return bundle
	}
	for _, st := range studies {
		if st.Theme == t.Theme && studyOpen(st) && st.Status != "planned" {
			return bundle
		}
	}
	anchors := personalAnchors(catalog, runs, contexts, now)
	baseNames := map[string]bool{}
	for _, row := range t.Rows {
		baseNames[strings.ToLower(row.Name)] = true
	}
	families := map[string][]Scenario{}
	for _, c := range catalog {
		if c.Enabled && c.Preference != "disliked" && c.PersonalDifficulty != "hard" && c.LocalAssessment != nil && c.LocalAssessment.FamilyFingerprint != "" {
			families[c.LocalAssessment.FamilyFingerprint] = append(families[c.LocalAssessment.FamilyFingerprint], enrichMechanics(c))
		}
	}
	type candidate struct {
		scene, anchor Scenario
		personal      PersonalAnchor
		relation      *PrecisionRelation
		weight        float64
		prediction    *ResponsePrediction
	}
	options := []candidate{}
	for _, row := range t.Rows {
		a, ok := byName[strings.ToLower(row.Name)]
		personal := anchors[strings.ToLower(row.Name)]
		if !ok || row.Role == "warmup" || personal.Status != "stable" || a.LocalAssessment == nil {
			continue
		}
		for _, c := range families[a.LocalAssessment.FamilyFingerprint] {
			if baseNames[strings.ToLower(c.Name)] {
				continue
			}
			relation := precisionRelation(a, c)
			direction := "higher_precision"
			if a.PersonalDifficulty == "hard" {
				direction = "lower_precision"
			}
			if relation == nil || relation.Direction != direction {
				continue
			}
			blocked := false
			for _, st := range studies {
				if st.TrainingHash == c.LocalAssessment.FileSHA256 && st.Feedback == "hard" && now.UnixMilli()-st.TrainedAt < int64(7*24*time.Hour/time.Millisecond) {
					blocked = true
				}
			}
			if blocked {
				continue
			}
			// The closest available controlled change receives the first probe.
			weight := 1 / (.05 + relation.MaxDelta)
			mastered := false
			for _, st := range studies {
				if st.AnchorHash == a.LocalAssessment.FileSHA256 && st.TrainingHash == c.LocalAssessment.FileSHA256 && st.Trial != nil {
					known := anchors[strings.ToLower(c.Name)]
					if st.Feedback == "easy" || known.Status == "stable" && known.MedianScore > st.Trial.Score+max(known.ScoreMAD, st.Trial.Score*.02) {
						mastered = true
					}
				}
			}
			if mastered {
				continue
			}
			if known := anchors[strings.ToLower(c.Name)]; known.Status == "stable" {
				weight += 5
				if known.Days > 0 && known.MedianScore <= 0 {
					continue
				}
			}
			prediction := responsePrediction(studies, personal, relation)
			if prediction.Status == "local_backtest" {
				weight *= max(.25, min(1.25, prediction.ExpectedScore/personal.MedianScore))
			}
			options = append(options, candidate{c, a, personal, relation, weight, prediction})
		}
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].weight != options[j].weight {
			return options[i].weight > options[j].weight
		}
		return options[i].scene.Name < options[j].scene.Name
	})
	for _, c := range options {
		timing := estimateTiming(c.scene, obs[strings.ToLower(c.scene.Name)], now)
		trialSeconds := 2 * timing.Seconds
		if trialSeconds > int(float64(p.Minutes*60*9/10)*min(p.Variety, .1)) {
			continue
		}
		st := TrainingStudy{ID: fmt.Sprintf("study-%d-%x", now.UnixMilli(), rng.Uint32()), Theme: t.Theme, AnchorScenario: c.anchor.Name,
			TrainingScenario: c.scene.Name, AnchorHash: c.anchor.LocalAssessment.FileSHA256, TrainingHash: c.scene.LocalAssessment.FileSHA256,
			Relation: *c.relation, Status: "planned", CreatedAt: now.UnixMilli()}
		before := measurementBlock(st, c.anchor, "baseline", obs, now)
		if before.Budget+trialSeconds > available {
			continue
		}
		bundle.before = []Block{before}
		// A familiar, same-goal scene outside this plan can measure transfer.
		// It remains a separate metric, never a cross-scene score comparison.
		for _, transfer := range catalog {
			if baseNames[strings.ToLower(transfer.Name)] || transfer.Name == c.scene.Name || !goalCompatible(c.anchor, enrichMechanics(transfer)) ||
				!transfer.Enabled || transfer.Preference == "disliked" || transfer.LocalAssessment == nil ||
				transfer.LocalAssessment.Status != "file_parsed_model_unfitted" || transfer.LocalAssessment.FamilyFingerprint == c.relation.FamilyFingerprint ||
				anchors[strings.ToLower(transfer.Name)].Status != "stable" {
				continue
			}
			test := measurementBlock(st, transfer, "transfer_baseline", obs, now)
			if before.Budget+test.Budget+trialSeconds > available || before.Budget+test.Budget > p.Minutes*60*9/10/4 {
				continue
			}
			st.TransferScenario, st.TransferHash = transfer.Name, transfer.LocalAssessment.FileSHA256
			bundle.before = append(bundle.before, test)
			break
		}
		source := "same_family_probe"
		if anchors[strings.ToLower(c.scene.Name)].Status == "stable" {
			source = "direct_history"
		}
		if c.prediction.Status == "local_backtest" {
			source = "personal_response"
		}
		decision := &SceneDecision{Source: source, Anchor: c.personal, Relation: c.relation, Prediction: c.prediction}
		bundle.after = []Block{{Scenario: c.scene, Role: "explore", PlayCount: 2, Budget: trialSeconds, Timing: timing, Personalization: decision,
			Measurement: &MeasurementSpec{st.ID, "trial"}, AnchorScenario: c.anchor.Name, Outcome: "pending",
			Reason: "匹配本地同族配置与布局；仅目标尺寸改变。先完成两局试练，结果用于下一次选择，不换算总体难度。",
			Cue:    "同族精度试练：保持平常节奏；两局完成后停止追加，观察速度与命中率的组合。"}}
		bundle.study = &st
		return bundle
	}
	return bundle
}

func (s *Service) updateStudiesLocked(now time.Time) {
	index := map[string]Scenario{}
	for _, c := range s.state.Catalog {
		index[strings.ToLower(c.Name)] = c
	}
	for i := range s.state.TrainingStudies {
		st := &s.state.TrainingStudies[i]
		if studyOpen(*st) {
			for name, hash := range map[string]string{st.AnchorScenario: st.AnchorHash, st.TrainingScenario: st.TrainingHash, st.TransferScenario: st.TransferHash} {
				if name == "" {
					continue
				}
				c, ok := index[strings.ToLower(name)]
				if !ok || c.LocalAssessment == nil || c.LocalAssessment.Status != "file_parsed_model_unfitted" || c.LocalAssessment.FileSHA256 != hash {
					st.Status = "invalidated"
					break
				}
			}
		}
		if !studyOpen(*st) {
			continue
		}
		plans := []*Plan{}
		for j := range s.state.History {
			plans = append(plans, &s.state.History[j])
		}
		if s.state.Plan != nil {
			plans = append(plans, s.state.Plan)
		}
		for _, plan := range plans {
			for _, block := range plan.Blocks {
				m := block.Measurement
				if m == nil || m.StudyID != st.ID {
					continue
				}
				switch m.Phase {
				case "baseline":
					if st.Baseline == nil {
						st.Baseline = measured(block.Observations, 1, 2, st.AnchorHash)
					}
				case "transfer_baseline":
					if st.TransferBaseline == nil {
						st.TransferBaseline = measured(block.Observations, 1, 2, st.TransferHash)
					}
				case "trial":
					if st.Trial == nil {
						st.Trial = measured(block.Observations, 0, 2, st.TrainingHash)
					}
				case "retest", "transfer_retest":
					hash, baseline := st.AnchorHash, st.Baseline
					if m.Phase == "transfer_retest" {
						hash, baseline = st.TransferHash, st.TransferBaseline
					}
					result := measured(block.Observations, 1, 2, hash)
					if result != nil && baseline != nil && result.Signature != baseline.Signature {
						st.Status = "settings_changed"
						continue
					}
					if result == nil || baseline == nil || result.Signature != baseline.Signature ||
						len(block.Observations) == 0 || block.Observations[0].StartedAt < st.DueAt || result.At > st.ExpiresAt {
						continue
					}
					if m.Phase == "retest" {
						st.Retest = result
						if baseline.Score > 0 {
							change := result.Score/baseline.Score - 1
							st.RetentionChange = &change
						}
					} else {
						st.TransferRetest = result
						if baseline.Score > 0 && !st.TransferContaminated {
							change := result.Score/baseline.Score - 1
							st.TransferChange = &change
						}
					}
				}
			}
		}
		if st.Status == "settings_changed" {
			continue
		}
		if st.Trial != nil {
			st.TrainedAt = st.Trial.At
			st.DueAt = st.TrainedAt + int64(24*time.Hour/time.Millisecond)
			st.ExpiresAt = st.TrainedAt + int64(72*time.Hour/time.Millisecond)
			st.Status = "waiting"
			if st.Baseline == nil || st.Baseline.At >= st.Trial.At || st.Baseline.Settings != st.Trial.Settings {
				st.Status = "baseline_missing"
			} else if st.Retest != nil && (st.TransferBaseline == nil || st.TransferRetest != nil) {
				st.Status = "measured"
			} else if now.UnixMilli() > st.ExpiresAt {
				st.Status = "expired"
			} else if now.UnixMilli() >= st.DueAt {
				st.Status = "due"
				if st.Retest != nil {
					st.Status = "transfer_due"
				}
			}
		} else if now.UnixMilli()-st.CreatedAt > int64(72*time.Hour/time.Millisecond) {
			st.Status = "incomplete"
		}
	}
	// Keep completed study results after the normal plan history is trimmed.
	if len(s.state.TrainingStudies) > 128 {
		s.state.TrainingStudies = s.state.TrainingStudies[len(s.state.TrainingStudies)-128:]
	}
	for key, c := range s.state.RunContexts {
		if c.At < now.AddDate(0, 0, -45).UnixMilli() {
			delete(s.state.RunContexts, key)
		}
	}
	if len(s.state.RunContexts) > 5000 {
		keys := make([]string, 0, len(s.state.RunContexts))
		for key := range s.state.RunContexts {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return s.state.RunContexts[keys[i]].At < s.state.RunContexts[keys[j]].At })
		for _, key := range keys[:len(keys)-5000] {
			delete(s.state.RunContexts, key)
		}
	}
}

// Practice on a retained transfer scene between measurements invalidates a
// transfer interpretation, including practice outside Refleks-owned plans.
func (s *Service) auditTransferExposureLocked(runs []models.RunRecord, now time.Time) {
	for i := range s.state.TrainingStudies {
		st := &s.state.TrainingStudies[i]
		if st.TransferBaseline == nil || st.TransferContaminated {
			continue
		}
		until := now.UnixMilli()
		if st.TransferRetest != nil {
			until = st.TransferRetest.At
		}
		for _, r := range runs {
			if !strings.EqualFold(r.Stats.Summary.Scenario, st.TransferScenario) || !validPractice(r, now) {
				continue
			}
			at, _ := time.Parse(time.RFC3339, r.Stats.Summary.DatePlayed)
			if at.UnixMilli() <= st.TransferBaseline.At || at.UnixMilli() > until {
				continue
			}
			c := s.state.RunContexts[runKey(r)]
			if c.StudyID == st.ID && c.Phase == "transfer_retest" {
				continue
			}
			st.TransferContaminated = true
			st.TransferChange = nil
			break
		}
	}
}

func (s *Service) TrialFeedback(id, feedback string) error {
	if feedback != "easy" && feedback != "suitable" && feedback != "hard" {
		return fmt.Errorf("无效试练反馈")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.TrainingStudies {
		st := &s.state.TrainingStudies[i]
		if st.ID == id {
			if st.Trial == nil {
				return fmt.Errorf("试练尚未形成两局有效记录")
			}
			st.Feedback = feedback
			return s.save()
		}
	}
	return fmt.Errorf("试练记录不存在")
}
