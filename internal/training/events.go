package training

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

const GameEventsFile = "kovaaks-events.jsonl"

// EventReader consumes complete JSON lines. It starts at EOF, so an old game
// session cannot be replayed into a new training plan after app startup.
type EventReader struct {
	mu      sync.Mutex
	path    string
	offset  int64
	pending []byte
	started bool
}

func NewEventReader(path string) *EventReader { return &EventReader{path: path} }

func (r *EventReader) Read() ([]GameEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := os.Open(r.path)
	if os.IsNotExist(err) {
		r.offset, r.pending, r.started = 0, nil, true
		return nil, nil
	}
	if err != nil { return nil, err }
	defer f.Close()
	info, err := f.Stat()
	if err != nil { return nil, err }
	if !r.started || info.Size() < r.offset {
		r.offset, r.pending, r.started = info.Size(), nil, true
		return nil, nil
	}
	if info.Size() == r.offset { return nil, nil }
	// A bad or idle producer cannot make the app read an unbounded file.
	if info.Size()-r.offset > 1<<20 {
		r.offset, r.pending = info.Size(), nil
		return nil, fmt.Errorf("游戏事件积压超过 1 MB，已跳过")
	}
	buf := make([]byte, int(info.Size()-r.offset))
	if _, err := f.ReadAt(buf, r.offset); err != nil { return nil, err }
	r.offset = info.Size()
	buf = append(r.pending, buf...)
	rows := strings.Split(string(buf), "\n")
	r.pending = []byte(rows[len(rows)-1])
	if len(r.pending) > 4096 { r.pending = nil; return nil, fmt.Errorf("游戏事件行过长") }
	events := make([]GameEvent, 0, len(rows)-1)
	for _, row := range rows[:len(rows)-1] {
		if len(row) > 4096 { continue }
		var event GameEvent
		if json.Unmarshal([]byte(row), &event) == nil { events = append(events, event) }
	}
	return events, nil
}

// ApplyGameEvent observes game lifecycle without inferring a completed score.
// Events never directly advance the plan or launch another scenario.
func (s *Service) ApplyGameEvent(event GameEvent, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.state.Plan
	if p == nil || (p.Status != "running" && p.Status != "waiting") { return nil }
	if event.At < p.AcceptAfter-2000 || event.At > now.Add(10*time.Second).UnixMilli() || event.At < p.Game.LastEventAt { return nil }
	event.Scenario = strings.TrimSpace(event.Scenario)
	if event.Scenario == "" || len(event.Scenario) > 200 { return nil }
	switch event.Type {
	case "challenge_start", "challenge_restart", "challenge_complete", "challenge_canceled", "challenge_quit":
	default:
		return nil
	}
	g := &p.Game
	if event.At == g.LastEventAt && event.Type == g.LastEventType { return nil }
	if event.Type == "challenge_start" || event.Type == "challenge_restart" {
		if g.Phase == "playing" && strings.EqualFold(g.Scenario, event.Scenario) && event.At > g.StartedAt {
			g.AbortedSeconds += float64(event.At-g.StartedAt) / 1000
			g.Restarts++
		} else if event.Type == "challenge_restart" && strings.EqualFold(g.Scenario, event.Scenario) {
			g.Restarts++
		}
		g.StartedAt, g.Phase = event.At, "playing"
	} else {
		if g.Phase == "playing" && event.Type != "challenge_complete" && strings.EqualFold(g.Scenario, event.Scenario) && event.At > g.StartedAt {
			g.AbortedSeconds += float64(event.At-g.StartedAt) / 1000
		}
		g.StartedAt, g.Phase = 0, "between"
	}
	g.Scenario, g.LastEventAt, g.LastEventType = event.Scenario, event.At, event.Type
	if !g.RestartReminder && p.Index < len(p.Blocks) && strings.EqualFold(event.Scenario, p.Blocks[p.Index].Scenario.Name) && g.Restarts >= 3 && g.AbortedSeconds >= math.Min(90, float64(p.Blocks[p.Index].Budget)/2) {
		g.RestartReminder = true
		p.Reminder = "这一关已反复重开，累计未完成练习约 " + fmt.Sprintf("%.0f", g.AbortedSeconds) + " 秒。请在本局结束后换关或短暂休息。"
		s.pendingReminder = p.Reminder
	}
	return s.save()
}
