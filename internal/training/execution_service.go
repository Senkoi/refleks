package training

import (
	"aimmeow/internal/models"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func (s *Service) TakeReminder() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	message := s.pendingReminder
	s.pendingReminder = ""
	return message
}

// Action returns a scenario to launch. The caller owns Steam integration.
func (s *Service) Action(action string, now time.Time, runs []models.RunRecord) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.state.Plan
	if p == nil {
		return "", fmt.Errorf("先让我安排一份训练列表喵。")
	}
	if action == "pause" || action == "finish" {
		// The watcher can write a finished run before the next two-second poll.
		// Ingest it before changing state, otherwise a manual pause/finish loses it.
		s.tickLocked(now, runs)
	}
	launch := ""
	switch action {
	case "start":
		s.clock(now)
		s.expire()
		if p.Status != "draft" && p.Status != "ready" && p.Status != "paused" {
			return "", fmt.Errorf("当前状态不能开始")
		}
		if p.Elapsed >= float64(p.Preferences.Minutes*60) || p.Index >= len(p.Blocks) {
			return "", fmt.Errorf("计划已结束")
		}
		if p.Status == "draft" {
			p.Seen = []string{}
			for _, r := range runs {
				p.Seen = append(p.Seen, runKey(r))
			}
		}
		// Ignore any records that finished while paused or before this block began.
		p.AcceptAfter = now.UnixMilli()
		p.LastTick = now.UnixMilli()
		p.Status = "running"
		s.state.Error = ""
		if p.Preferences.ExecutionMode != "playlist" {
			launch = p.Blocks[p.Index].Scenario.Name
		}
	case "pause":
		if p.Status == "running" || p.Status == "ready" || p.Status == "waiting" {
			s.clock(now)
			s.expire()
			if p.Status != "completed" {
				p.Status = "paused"
			}
			p.LastTick = 0
		}
	case "next":
		if p.Status != "running" && p.Status != "paused" && p.Status != "waiting" {
			return "", fmt.Errorf("当前状态不能跳过")
		}
		s.clock(now)
		s.expire()
		paused := p.Status == "paused"
		if p.Status != "completed" {
			s.advance("skipped", now)
			if p.Status == "ready" && p.Preferences.ExecutionMode == "playlist" {
				p.Status = "running"
				if paused {
					p.Status = "paused"
				}
			}
		}
	case "finish":
		s.clock(now)
		if p.Status != "completed" {
			p.Status = "completed"
			p.EndReason = "manual"
			p.EndedAt = now.UnixMilli()
			p.LastTick = 0
		}
	default:
		return "", fmt.Errorf("未知操作")
	}
	return launch, s.save()
}

func (s *Service) clock(now time.Time) {
	p := s.state.Plan
	if p == nil || (p.Status != "running" && p.Status != "ready" && p.Status != "waiting") {
		return
	}
	if p.LastTick > 0 {
		dt := math.Max(0, float64(now.UnixMilli()-p.LastTick)/1000)
		p.Elapsed += dt
		if p.Status == "running" || p.Status == "waiting" {
			p.BlockElapsed += dt
		}
	}
	p.LastTick = now.UnixMilli()
}

func (s *Service) expire() {
	p := s.state.Plan
	if p != nil && p.Status != "completed" && p.Elapsed >= float64(p.Preferences.Minutes*60) {
		if !p.RemindedEnd {
			p.RemindedEnd = true
			p.Reminder = "今天安排的时间到啦喵。练完这一局就结束列表，松松爪子休息一下。"
			s.pendingReminder = p.Reminder
		}
		if p.EndedAt == 0 {
			p.EndedAt = p.LastTick - int64((p.Elapsed-float64(p.Preferences.Minutes*60))*1000)
		}
		p.Status = "completed"
		p.EndReason = "time_budget"
		p.LastTick = 0
		if p.Index < len(p.Blocks) && p.Blocks[p.Index].Outcome == "pending" {
			p.Blocks[p.Index].Outcome = "session_limit"
		}
	}
}

func (s *Service) advance(outcome string, now time.Time) {
	p := s.state.Plan
	p.Blocks[p.Index].Outcome = outcome
	p.Reminder = ""
	p.Index++
	p.BlockElapsed = 0
	p.AcceptAfter = now.UnixMilli()
	p.Status = "ready"
	if p.Index >= len(p.Blocks) {
		p.Status = "completed"
		p.EndedAt = now.UnixMilli()
		p.EndReason = "plan_complete"
		for _, b := range p.Blocks {
			if !blockCompleted(b, p.Preferences.ExecutionMode) {
				p.EndReason = "items_processed"
				break
			}
		}
		p.LastTick = 0
	}
}

// Tick consumes each completed run once, in chronological order, even when the
// Training page is closed. It never kills a live run at a time limit.
func (s *Service) Tick(now time.Time, runs []models.RunRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tickLocked(now, runs)
}

func (s *Service) tickLocked(now time.Time, runs []models.RunRecord) string {
	p := s.state.Plan
	// File ingestion may lag the cap by more than one polling interval. Accept
	// only runs that actually ended before the cap, for a bounded grace period.
	if p != nil && p.Status == "completed" && p.EndedAt > 0 && now.UnixMilli() <= p.EndedAt+120000 && p.Index < len(p.Blocks) {
		previousRecorded := p.Recorded
		seen := map[string]bool{}
		for _, k := range p.Seen {
			seen[k] = true
		}
		ordered := append([]models.RunRecord{}, runs...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Stats.Summary.DatePlayed < ordered[j].Stats.Summary.DatePlayed })
		for _, r := range ordered {
			if seen[runKey(r)] {
				continue
			}
			seen[runKey(r)] = true
			p.Seen = append(p.Seen, runKey(r))
			v := r.Stats.Summary
			end, err := time.Parse(time.RFC3339, v.DatePlayed)
			if err != nil || v.TimeRemaining > 1 || v.Duration <= 0 || math.IsNaN(v.Duration) || math.IsInf(v.Duration, 0) || v.Score < 0 || math.IsNaN(v.Score) || math.IsInf(v.Score, 0) {
				continue
			}
			start := end.Add(-time.Duration(v.Duration * float64(time.Second)))
			b := &p.Blocks[p.Index]
			if !strings.EqualFold(v.Scenario, b.Scenario.Name) || start.UnixMilli() < p.AcceptAfter-2000 || end.UnixMilli() > p.EndedAt+1000 || end.After(now.Add(2*time.Second)) {
				continue
			}
			b.Recorded += v.Duration
			p.Recorded += v.Duration
			b.Runs++
			b.LastCompletedAt = end.UnixMilli()
			s.capturePractice(b, r, now)
			b.Best = math.Max(b.Best, v.Score)
		}
		if p.Recorded != previousRecorded {
			s.syncSessionContextsLocked(runs, now)
			s.updateAnchorEvaluationsLocked(runs, now)
		}
		if err := s.checkpoint(now, p.Recorded != previousRecorded); err != nil {
			s.state.Error = "保存训练进度失败：" + err.Error()
		}
		return ""
	}
	if p == nil || (p.Status != "running" && p.Status != "ready" && p.Status != "waiting") {
		return ""
	}
	previousRecorded, previousStatus, previousIndex := p.Recorded, p.Status, p.Index
	s.clock(now)
	launch := ""
	if p.Status == "running" || p.Status == "waiting" {
		seen := map[string]bool{}
		for _, id := range p.Seen {
			seen[id] = true
		}
		ordered := append([]models.RunRecord{}, runs...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Stats.Summary.DatePlayed < ordered[j].Stats.Summary.DatePlayed })
		for _, r := range ordered {
			if p.Status != "running" && p.Status != "waiting" {
				break
			}
			if seen[runKey(r)] {
				continue
			}
			seen[runKey(r)] = true
			p.Seen = append(p.Seen, runKey(r))
			sum := r.Stats.Summary
			end, err := time.Parse(time.RFC3339, sum.DatePlayed)
			if err != nil {
				continue
			}
			start := end.Add(-time.Duration(sum.Duration * float64(time.Second)))
			if start.UnixMilli() < p.AcceptAfter-2000 || end.After(now.Add(2*time.Second)) {
				continue
			}
			if sum.TimeRemaining > 1 || sum.Duration <= 0 || math.IsNaN(sum.Duration) || math.IsInf(sum.Duration, 0) || sum.Score < 0 || math.IsNaN(sum.Score) || math.IsInf(sum.Score, 0) {
				b := &p.Blocks[p.Index]
				if p.PlannerVersion >= 8 && strings.EqualFold(sum.Scenario, b.Scenario.Name) && b.Runs < 4 {
					b.ObservationInterrupted = true
				}
				continue
			}
			if p.Preferences.ExecutionMode == "playlist" && p.Index+1 < len(p.Blocks) {
				current, next := p.Blocks[p.Index], p.Blocks[p.Index+1]
				// Same-name adjacent rows provide no name-change signal. Use the
				// exported repetition boundary; different-name rows still wait for
				// a completed run in the next scenario, allowing extra practice.
				if current.Runs >= max(1, current.PlayCount) && strings.EqualFold(current.Scenario.Name, next.Scenario.Name) && strings.EqualFold(sum.Scenario, next.Scenario.Name) {
					s.advance("list_complete", start)
					p.BlockElapsed = math.Min(p.Elapsed, math.Max(0, now.Sub(start).Seconds()))
					p.Status = "running"
				}
			}
			if p.Preferences.ExecutionMode == "playlist" && !strings.EqualFold(p.Blocks[p.Index].Scenario.Name, sum.Scenario) {
				// A finished run in a later playlist row is stronger evidence than
				// the generated play counts. Keep the local plan aligned if the
				// player skipped a row or manually advanced in-game.
				future := -1
				for i := p.Index + 1; i < len(p.Blocks); i++ {
					if strings.EqualFold(p.Blocks[i].Scenario.Name, sum.Scenario) {
						future = i
						break
					}
				}
				if future >= 0 {
					for p.Index < future {
						outcome := "missed"
						if p.Blocks[p.Index].Outcome == "list_complete" {
							outcome = "list_complete"
						}
						s.advance(outcome, start)
					}
					// Backdate to this completed run's start, not its file-arrival
					// time. The first run of the next scenario already used time.
					p.BlockElapsed = math.Min(p.Elapsed, math.Max(0, now.Sub(start).Seconds()))
					p.Status = "running"
				}
			}
			b := &p.Blocks[p.Index]
			if !strings.EqualFold(b.Scenario.Name, sum.Scenario) {
				continue
			}
			b.Recorded += sum.Duration
			p.Recorded += sum.Duration
			b.Runs++
			b.LastCompletedAt = end.UnixMilli()
			s.capturePractice(b, r, now)
			b.Best = math.Max(b.Best, sum.Score)
			if b.Target > 0 && b.Signature != "" && signature(sum) != b.Signature && legacySignature(sum) != b.Signature {
				b.Target = 0
				b.Reason += " 本模块检测到版本或设置变化，已取消原阈值。"
			}
			if b.Target > 0 && b.Personalization != nil && practiceSample(r).Signature != b.Personalization.Anchor.Signature {
				b.Target = 0
				b.Reason += " 个人锚点的版本、设置或时长不再可比，取消原目标。"
			}
			outcome := ""
			if p.Preferences.ExecutionMode == "playlist" {
				count := b.PlayCount
				if count < 1 {
					count = 1
				}
				if b.Runs >= count {
					b.Outcome = "list_complete"
					// Repetitions are a target, not evidence that the player has
					// changed scenarios. Keep attributing extra runs to this block.
					if p.Index == len(p.Blocks)-1 {
						outcome = "list_complete"
						p.Reminder = "这一份练完啦喵！辛苦了，松松爪子休息一下。"
						s.pendingReminder = p.Reminder
					}
				}
			} else if b.Role == "benchmark" {
				outcome = "measured"
			} else if b.Target > 0 && sum.Score >= b.Target {
				outcome = "threshold"
			} else if p.BlockElapsed >= float64(b.Budget) || b.Recorded >= float64(b.Budget) {
				outcome = "time_limit"
			}
			if outcome != "" {
				boundary := now
				if p.Preferences.ExecutionMode == "playlist" {
					boundary = end
				}
				s.advance(outcome, boundary)
				if p.Status == "ready" && p.Preferences.ExecutionMode == "playlist" {
					p.Status = "running"
				} else if p.Status == "ready" && p.Preferences.AutoAdvance {
					next := p.Blocks[p.Index]
					if p.Elapsed+float64(next.Scenario.Seconds) <= float64(p.Preferences.Minutes*60) {
						p.Status = "running"
						launch = next.Scenario.Name
					}
				}
			}
		}
		// No run was completed at the cap: do not launch another scenario over a live run.
		if p.Status == "running" && p.Preferences.ExecutionMode != "playlist" && p.BlockElapsed >= float64(p.Blocks[p.Index].Budget) {
			p.Status = "waiting"
		}
	}
	if p.Status == "running" && p.Preferences.ExecutionMode == "playlist" && p.Index < len(p.Blocks) && p.BlockElapsed >= float64(p.Blocks[p.Index].Budget) && p.RemindedBlock != p.Index+1 {
		p.RemindedBlock = p.Index + 1
		p.Reminder = "这张图练得差不多啦喵。完成这一局，就换张图或休息一下。"
		s.pendingReminder = p.Reminder
	}
	s.expire()
	if p.Status == "completed" {
		launch = ""
	}
	if p.Recorded != previousRecorded || p.Status != previousStatus || p.Index != previousIndex {
		s.syncSessionContextsLocked(runs, now)
		s.updateAnchorEvaluationsLocked(runs, now)
	}
	if err := s.checkpoint(now, p.Recorded != previousRecorded || p.Status != previousStatus || p.Index != previousIndex); err != nil {
		s.state.Error = "保存训练进度失败：" + err.Error()
	}
	return launch
}

func (s *Service) LaunchFailed(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Error = "无法启动关卡：" + err.Error()
	if p := s.state.Plan; p != nil {
		p.Status = "paused"
		p.LastTick = 0
	}
	_ = s.save()
}
