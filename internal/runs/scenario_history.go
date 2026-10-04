package runs

import (
	"math"
	"sort"
	"strings"
	"time"

	"refleks/internal/runs/kovaaks"
)

// ScenarioHistoryPoint contains chart data only, never trace or event payloads.
type ScenarioHistoryPoint struct {
	At          int64   `json:"at"`
	Score       float64 `json:"score"`
	Accuracy    float64 `json:"accuracy"`
	Version     string  `json:"version"`
	GameVersion string  `json:"gameVersion"`
	SensScale   string  `json:"sensScale"`
	Sens        float64 `json:"sens"`
	VertSens    float64 `json:"vertSens"`
	DPI         float64 `json:"dpi"`
	FOV         float64 `json:"fov"`
	FOVScale    string  `json:"fovScale"`
	Seconds     float64 `json:"seconds"`
	TargetScale float64 `json:"targetScale"`
	TimeScale   float64 `json:"timeScale"`
}

// LoadScenarioHistory is independent of the recent-runs display window and
// the 45-day ability window. Only the requested scenario's summaries are read.
func (s *Store) LoadScenarioHistory(name string) ([]ScenarioHistoryPoint, error) {
	dir, err := s.runsDir()
	if err != nil {
		return nil, err
	}
	return s.loadScenarioHistory(dir, name)
}

func (s *Store) loadScenarioHistory(dir, name string) ([]ScenarioHistoryPoint, error) {
	out := []ScenarioHistoryPoint{}
	name = strings.TrimSpace(name)
	if name == "" {
		return out, nil
	}
	if err := s.index.ensureScanned(dir); err != nil {
		return nil, err
	}
	selected := []recentFile{}
	for _, file := range s.index.recent(0, 0, 0) {
		info, err := kovaaks.ParseFilename(file.name)
		if err == nil && !strings.EqualFold(strings.TrimSpace(info.ScenarioName), name) {
			continue
		}
		selected = append(selected, file)
	}
	rows, err := s.loadRunSummaries(selected)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		summary := r.Stats.Summary
		if !strings.EqualFold(strings.TrimSpace(summary.Scenario), name) {
			continue
		}
		at, err := time.Parse(time.RFC3339, summary.DatePlayed)
		if err != nil {
			at = time.UnixMilli(runTimestampFromFileName(r.FileName, r.FilePath))
		}
		seconds := summary.Duration
		if r.Performances != nil && r.Performances.Header.ChallengeProfile.TimeLimit > 0 {
			seconds = float64(r.Performances.Header.ChallengeProfile.TimeLimit)
		}
		point := ScenarioHistoryPoint{At: at.UnixMilli(), Score: summary.Score, Accuracy: summary.Accuracy,
			Version: summary.Hash, GameVersion: summary.GameVersion, SensScale: summary.SensScale, Sens: summary.HorizSens, VertSens: summary.VertSens,
			DPI: summary.DPI, FOV: summary.FOV, FOVScale: summary.FOVScale, Seconds: seconds,
			TargetScale: summary.AvgTargetScale, TimeScale: summary.AvgTimeDilation}
		valid := point.At > 0
		for _, value := range []float64{point.Score, point.Accuracy, point.Sens, point.VertSens, point.DPI, point.FOV, point.Seconds, point.TargetScale, point.TimeScale} {
			valid = valid && !math.IsNaN(value) && !math.IsInf(value, 0)
		}
		if valid {
			out = append(out, point)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out, nil
}

// LoadPlayedScenarioNames uses the filename index so catalog rows don't each
// fetch a history. Older nonstandard filenames fall back to their summary.
func (s *Store) LoadPlayedScenarioNames() ([]string, error) {
	dir, err := s.runsDir()
	if err != nil {
		return nil, err
	}
	return s.loadPlayedScenarioNames(dir)
}

func (s *Store) loadPlayedScenarioNames(dir string) ([]string, error) {
	if err := s.index.ensureScanned(dir); err != nil {
		return nil, err
	}
	seen := map[string]string{}
	unknown := []recentFile{}
	for _, file := range s.index.recent(0, 0, 0) {
		info, err := kovaaks.ParseFilename(file.name)
		if err != nil {
			unknown = append(unknown, file)
			continue
		}
		name := strings.TrimSpace(info.ScenarioName)
		seen[strings.ToLower(name)] = name
	}
	rows, err := s.loadRunSummaries(unknown)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		name := strings.TrimSpace(r.Stats.Summary.Scenario)
		if name != "" {
			seen[strings.ToLower(name)] = name
		}
	}
	names := make([]string, 0, len(seen))
	for _, name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
