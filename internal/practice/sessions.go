// Package practice owns observed training intervals, independent of plans.
package practice

import (
	"aimmeow/internal/constants"
	"aimmeow/internal/models"
	"fmt"
	"math"
	"sort"
	"time"
)

type Entry struct {
	RunID   string  `json:"runId"`
	EndedAt int64   `json:"endedAt"`
	Seconds float64 `json:"seconds"`
}

type SessionGroup struct {
	ID        string   `json:"id"`
	StartedAt int64    `json:"startedAt"`
	EndedAt   int64    `json:"endedAt"`
	RunIDs    []string `json:"runIds"`
	LegacyIDs []string `json:"legacyIds"`
}

type SessionRequest struct {
	Entries    []Entry `json:"entries"`
	GapMinutes int     `json:"gapMinutes"`
}

func RunID(r models.RunRecord) string {
	if r.FilePath != "" {
		return r.FilePath
	}
	return r.FileName + "|" + r.Stats.Summary.DatePlayed + "|" + r.Stats.Summary.Scenario
}

func Entries(runs []models.RunRecord) []Entry {
	out := make([]Entry, 0, len(runs))
	for _, r := range runs {
		end, err := time.Parse(time.RFC3339, r.Stats.Summary.DatePlayed)
		if err == nil {
			out = append(out, Entry{RunID(r), end.UnixMilli(), r.Stats.Summary.Duration})
		}
	}
	return out
}

// Group uses the rest from the previous end to the next start. An exact gap
// starts a new interval. Scores and plan ownership never determine boundaries.
func Group(entries []Entry, gap time.Duration, now time.Time) []SessionGroup {
	if gap <= 0 {
		gap = constants.DefaultSessionGapMinutes * time.Minute
	}
	ordered := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.RunID != "" && e.EndedAt > 0 && e.EndedAt <= now.Add(2*time.Second).UnixMilli() && !math.IsNaN(e.Seconds) && !math.IsInf(e.Seconds, 0) && e.Seconds >= 0 && e.Seconds <= 3600 {
			ordered = append(ordered, e)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].EndedAt == ordered[j].EndedAt {
			return ordered[i].RunID < ordered[j].RunID
		}
		return ordered[i].EndedAt < ordered[j].EndedAt
	})
	groups := []SessionGroup{}
	seen := map[string]bool{}
	for _, e := range ordered {
		if seen[e.RunID] {
			continue
		}
		seen[e.RunID] = true
		start := e.EndedAt - int64(e.Seconds*1000)
		if len(groups) == 0 || start-groups[len(groups)-1].EndedAt >= gap.Milliseconds() {
			groups = append(groups, SessionGroup{ID: "session-" + e.RunID, StartedAt: start, EndedAt: e.EndedAt, RunIDs: []string{}, LegacyIDs: []string{}})
		}
		g := &groups[len(groups)-1]
		g.StartedAt = min(g.StartedAt, start)
		g.EndedAt = max(g.EndedAt, e.EndedAt)
		g.RunIDs = append(g.RunIDs, e.RunID)
		// Old UI IDs used the first completion timestamp. Keep all possible
		// aliases so names/notes survive merged intervals and progressive loads.
		g.LegacyIDs = append(g.LegacyIDs, fmt.Sprintf("session-%d", e.EndedAt))
		g.LegacyIDs = append(g.LegacyIDs, "session-"+e.RunID)
	}
	return groups
}
