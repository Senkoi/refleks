package training

import (
	"aimmeow/internal/models"
	"time"
)

func legacyFixedSets(p Plan) bool {
	if p.PlannerVersion >= 6 {
		return false
	}
	for _, b := range p.Blocks {
		if b.SourcePlayCount == 0 && (b.PlayCount >= 5 || b.PlayCount == 0 && b.Budget == 300 && b.Scenario.Seconds <= 90) {
			return true
		}
	}
	return false
}

// BeginStartup hides persisted drafts until benchmark caches and history are ready.
func (s *Service) BeginStartup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = true
}

// PrepareStartup rebuilds inactive lists with current history and policy before
// exposing them. Current-policy sessions remain immutable; legacy fixed
// sets are archived once at restart.
func (s *Service) PrepareStartup(runs []models.RunRecord) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = true
	migrated := false
	if old := s.state.Plan; old != nil && old.Status == "paused" && legacyFixedSets(*old) {
		// Replace the legacy fixed five-minute startup policy.
		// Preserve every recorded score/progress as ended history; pending
		// rows cannot masquerade as a completed curriculum baseline.
		archived := *old
		archived.Blocks = append([]Block(nil), old.Blocks...)
		archived.Status, archived.EndedAt = "completed", time.Now().UnixMilli()
		for i := range archived.Blocks {
			if archived.Blocks[i].Outcome == "pending" || archived.Blocks[i].Outcome == "" {
				archived.Blocks[i].Outcome = "skipped"
			}
		}
		s.state.Plan = &archived
		migrated = true
	}
	if old := s.state.Plan; old != nil && (old.Status == "running" || old.Status == "ready" || old.Status == "paused" || old.Status == "waiting") {
		s.state.Initializing = false
		return false, nil
	}
	p := automaticTrainingPreferences(s.state.Preferences)
	if old := s.state.Plan; old != nil && old.PlannerVersion >= 7 && old.Status == "draft" {
		s.state.Initializing = false
		return false, nil
	}
	if err := validatePreferences(p); err != nil {
		p = defaults()
	}
	if _, err := s.generateLocked(p, runs); err != nil {
		s.state.Error = "启动训练列表生成失败：" + err.Error()
		s.state.Initializing = false
		return false, err
	}
	s.state.Notice = "本次列表安排好啦喵，我参考了你已有的测试成绩和训练记录。"
	if migrated {
		s.state.Notice = "旧版列表已归档，练过的记录都留好了喵。这次我按你的训练水平重新安排短组练习。"
	}
	return true, s.save()
}

// FinishStartup releases the UI only after the owned playlist installation.
func (s *Service) FinishStartup(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Initializing = false
	if err != nil {
		s.state.Error = err.Error()
	}
}

func (s *Service) StartupFailed(err error) { s.FinishStartup(err) }
