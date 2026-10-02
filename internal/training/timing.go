package training

import (
	"math"
	"time"
)

// Duration is a planning estimate, not a live game-state signal. Ignore future,
// stale and invalid observations before selecting the newest comparable group.
func estimateTiming(s Scenario, rows []observation, now time.Time) TimingEstimate {
	t := TimingEstimate{Seconds: s.Seconds, Source: "catalog"}
	if t.Seconds <= 0 || t.Seconds > 3600 {
		t.Seconds, t.Source = 60, "default"
	}
	valid := []observation{}
	for _, r := range rows {
		if r.at.After(now) || r.at.Before(now.AddDate(0, 0, -90)) || r.duration <= 0 || r.duration > 3600 || math.IsNaN(r.duration) || math.IsInf(r.duration, 0) {
			continue
		}
		valid = append(valid, r)
		if !r.at.Before(now.Add(-24 * time.Hour)) {
			t.RecentSeconds += r.duration
		}
		if !r.at.Before(now.AddDate(0, 0, -7)) {
			t.WeeklySeconds += r.duration
		}
	}
	durations := []float64{}
	for _, r := range comparable(valid) {
		durations = append(durations, r.duration)
	}
	t.Samples = len(durations)
	if t.Samples >= 3 {
		t.Seconds = int(math.Ceil(median(durations)))
		t.Source = "history"
	}
	return t
}

// Short sets are a product heuristic, not a fatigue diagnosis. Keep whole runs,
// reduce repeats after recent exposure, and leave the remaining time for other
// scenarios instead of extending every scenario to an equal time share.
func plannedRepetitions(t TimingEstimate, role string) int {
	if role == "benchmark" || role == "explore" {
		return 1
	}
	if t.Seconds <= 0 {
		return 1
	}
	n := int(math.Round(180 / float64(t.Seconds)))
	if n < 1 {
		n = 1
	}
	if n > 3 {
		n = 3
	}
	if role == "warmup" && n > 2 {
		n = 2
	}
	if t.RecentSeconds >= 180 {
		n = 1
	} else if t.WeeklySeconds >= 600 && n > 1 {
		n--
	}
	return n
}
