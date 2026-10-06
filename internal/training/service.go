package training

import (
	"aimmeow/internal/models"
	"encoding/json"
	"path/filepath"
	"sync"
	"time"
)

type Service struct {
	guidance     guidanceCache
	mu           sync.Mutex
	dataRevision uint64
	assessmentCache
	checkpointAt    time.Time
	path            string
	state           State
	discovering     bool
	pendingReminder string
	scenes          localSceneCache
	repository      *stateRepository

	sessionGap time.Duration
}

func New(dir string) (*Service, error) {
	s := &Service{path: filepath.Join(dir, "adaptive-training.json"), state: State{Version: 1, Catalog: []Scenario{}, History: []Plan{}, Skills: []SkillStatus{}, Preferences: defaults(), Discovery: Discovery{Candidates: []Candidate{}, Warnings: []string{}}}}
	s.repository = newStateRepository(s.path)
	loaded, err := s.repository.Load(s.state)
	if err != nil {
		return nil, err
	}
	s.state = loaded
	normalizePlanEnd(s.state.Plan)
	for i := range s.state.History {
		normalizePlanEnd(&s.state.History[i])
	}
	// Keep version 1 readable: the new multi-system field extends the JSON
	// schema without dropping its original benchmark and score cutoffs.
	for i := range s.state.Catalog {
		item := &s.state.Catalog[i]
		item.Benchmarks = mergeMemberships(nil, memberships(*item))
		for j := range item.Benchmarks {
			item.Benchmarks[j] = normalizeMembership(item.Benchmarks[j])
		}
		calculateFileEvidence(item.LocalAssessment)
		*item = enrichMechanics(*item)
		if item.DifficultySource == "playlist" && item.Classification != "manual" {
			item.Difficulty, item.DifficultySource = "unknown", "unknown"
		}
	}
	if s.state.Preferences.ExecutionMode == "" {
		s.state.Preferences.ExecutionMode = "playlist"
	}
	s.dataRevision = s.state.Revision
	s.state.Initializing = false
	if s.state.Error == "应用重新启动，训练已暂停；请确认后继续。" {
		s.state.Error = ""
		s.state.Notice = "我回来啦喵，训练已暂停。准备好后继续，也可以结束这次练习。"
	}
	if p := s.state.Plan; p != nil && (p.Status == "running" || p.Status == "ready" || p.Status == "waiting") {
		p.Status = "paused"
		p.LastTick = 0
		s.state.Notice = "我回来啦喵，训练已暂停。准备好后继续，也可以结束这次练习。"
	}
	return s, nil
}

func (s *Service) save() error {
	s.dataRevision++
	s.evaluationKey = ""
	return s.persist()
}

func (s *Service) SetAutoDiscover(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.state.Preferences.AutoDiscover
	s.state.Preferences.AutoDiscover = enabled
	if err := s.save(); err != nil {
		s.state.Preferences.AutoDiscover = previous
		return err
	}
	return nil
}

func (s *Service) persist() error {
	s.syncProgressLocked()
	s.state.Revision = s.dataRevision
	if s.repository == nil {
		s.repository = newStateRepository(s.path)
	}
	return s.repository.Save(s.state)
}

func (s *Service) Snapshot(runs []models.RunRecord) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluateLocked(runs, time.Now())
	b, _ := json.Marshal(s.state)
	var copy State
	_ = json.Unmarshal(b, &copy)
	return copy
}

// Legacy "completed" meant ended for every reason. Do not invent full completion.
func normalizePlanEnd(p *Plan) {
	if p == nil || p.Status != "completed" || p.EndReason != "" {
		return
	}
	p.EndReason = "legacy_unknown"
	complete := len(p.Blocks) > 0
	for _, b := range p.Blocks {
		complete = complete && blockCompleted(b, p.Preferences.ExecutionMode)
		if b.Outcome == "session_limit" {
			p.EndReason = "time_budget"
		}
	}
	if complete {
		p.EndReason = "plan_complete"
	}
}
