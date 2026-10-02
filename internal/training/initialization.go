package training

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"refleks/internal/models"
	"strings"
	"time"
)

//go:embed data/vdim-s5-playlists.json
var bundledVDIM []byte

//go:embed data/benchmark-references.json
var bundledReferences []byte

// Called at desktop startup, before the training UI is available. Verified
// official-code responses make first-run initialization usable offline too.
func (s *Service) InitializeTraining() error {
	var data struct {
		Playlists []struct {
			SourceURL string          `json:"sourceURL"`
			Name      string          `json:"playlistName"`
			Code      string          `json:"playlistCode"`
			Rows      []CurriculumRow `json:"scenarioList"`
		} `json:"playlists"`
	}
	if err := json.Unmarshal(bundledVDIM, &data); err != nil {
		return err
	}
	items := []Scenario{}
	for _, p := range data.Playlists {
		b, _ := json.Marshal(struct {
			Name string          `json:"playlistName"`
			Rows []CurriculumRow `json:"scenarioList"`
		}{p.Name, p.Rows})
		rows, err := ParsePlaylist(b, Source{URL: p.SourceURL, Title: p.Name, Retrieved: "2026-10-02"})
		if err != nil {
			return fmt.Errorf("内置 VDIM 无效: %w", err)
		}
		if len(rows) == 0 || rows[0].ImportedCurriculum == nil {
			return fmt.Errorf("内置 VDIM 模板缺失")
		}
		rows[0].ImportedCurriculum.OfficialCode = p.Code
		items = append(items, rows...)
	}
	var references []Scenario
	if err := json.Unmarshal(bundledReferences, &references); err != nil {
		return err
	}
	// On an upgrade, current downloaded/live memberships take precedence over
	// the bundled cutoff snapshot. The bundle supplies missing offline anchors.
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range references {
		for _, c := range s.state.Catalog {
			if strings.EqualFold(c.Name, references[i].Name) {
				references[i].Benchmarks = mergeMemberships(references[i].Benchmarks, memberships(c))
			}
		}
	}
	s.state.Catalog = mergeCatalog(s.state.Catalog, append(items, references...))
	s.mergeCurricula(items)
	s.state.Preferences = automaticTrainingPreferences(s.state.Preferences)
	return s.save()
}

func automaticTrainingPreferences(p Preferences) Preferences {
	p.PlanningPolicy = "curriculum"
	p.ExecutionMode = "playlist"
	p.AutoAdvance = false
	p.CurriculumID = ""
	p.Difficulty = "any"
	p.Benchmark = ""
	p.Benchmarks = nil
	return p
}

// A session is an ordered, whole-row window of the original routine. Continue
// only after recorded completion, never after a manual finish or skipped row.
func curriculumWindow(t Curriculum, catalog []Scenario, runs []models.RunRecord, p Preferences, now time.Time, history []Plan) (Curriculum, int, int, error) {
	if len(t.Rows) == 0 {
		return t, 0, 0, fmt.Errorf("VDIM 模板没有场景")
	}
	start := 0
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		if h.CurriculumID != t.ID || h.CurriculumTotal != len(t.Rows) || h.CurriculumHash != "" && h.CurriculumHash != t.ContentSHA256 {
			continue
		}
		if h.CurriculumStart < 0 || h.CurriculumEnd > len(t.Rows) || h.CurriculumEnd <= h.CurriculumStart {
			continue
		}
		slice := t
		slice.Rows = t.Rows[h.CurriculumStart:h.CurriculumEnd]
		if completedCurriculum(h, slice) {
			start = h.CurriculumEnd % len(t.Rows)
		} else {
			start = h.CurriculumStart
		}
		break
	}
	obs := observed(runs)
	index := map[string]Scenario{}
	for _, c := range catalog {
		index[strings.ToLower(c.Name)] = c
	}
	end, used := start, 0
	totalUsable := p.Minutes * 60 * 9 / 10
	usable := totalUsable
	if curriculumBaseline(t, history) && p.Variety > 0 {
		usable -= int(float64(totalUsable) * p.Variety)
	}
	rows := []CurriculumRow{}
	for end < len(t.Rows) {
		row := t.Rows[end]
		c, ok := index[strings.ToLower(row.Name)]
		if !ok {
			return t, 0, 0, fmt.Errorf("模板场景 %s 不在关卡库", row.Name)
		}
		timing := estimateTiming(c, obs[strings.ToLower(c.Name)], now)
		row.SourceCount = row.Count
		row.Count = curriculumRepetitions(row, timing)
		// A required long single run takes precedence over the exploration reserve.
		if end == start && timing.Seconds > usable && timing.Seconds <= totalUsable {
			usable = totalUsable
		}
		remaining := (usable - used) / timing.Seconds
		if remaining < 1 {
			break
		}
		row.Count = min(row.Count, remaining)
		used += timing.Seconds * row.Count
		rows = append(rows, row)
		end++
	}
	if end == start {
		return t, 0, 0, fmt.Errorf("当前 VDIM 段落至少需 %d 分钟，请增加可用时间", (estimateTiming(index[strings.ToLower(t.Rows[start].Name)], obs[strings.ToLower(t.Rows[start].Name)], now).Seconds*10+539)/540)
	}
	window := t
	window.Rows = rows
	return window, start, end, nil
}

func curriculumBaseline(t Curriculum, history []Plan) bool {
	covered := make([]bool, len(t.Rows))
	for _, h := range history {
		if completedCurriculum(h, t) {
			return true
		}
		if h.CurriculumID != t.ID || h.CurriculumTotal != len(t.Rows) || h.CurriculumStart < 0 || h.CurriculumEnd > len(t.Rows) || h.CurriculumEnd <= h.CurriculumStart {
			continue
		}
		window := t
		window.Rows = t.Rows[h.CurriculumStart:h.CurriculumEnd]
		if completedCurriculum(h, window) {
			for i := h.CurriculumStart; i < h.CurriculumEnd; i++ {
				covered[i] = true
			}
		}
	}
	for _, v := range covered {
		if !v {
			return false
		}
	}
	return len(covered) > 0
}
