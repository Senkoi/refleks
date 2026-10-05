package training

import (
	"aimmeow/internal/sceneanalysis"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Scene file observation owns its cache and lock, separate from execution.
type localSceneCache struct {
	mu       sync.Mutex
	files    map[string]string
	attempts map[string]time.Time
	parsed   map[string]localParsedSCE
	dirty    bool
}

type localParsedSCE struct {
	name       string
	mechanics  *Mechanics
	assessment *LocalAssessment
}

// LocalRoots follows the configured Steam library as well as the main Steam
// installation. No assumption that the game lives on C: is needed.
func LocalRoots(game, steam string) []string {
	out := []string{}
	if game != "" {
		out = append(out, filepath.Join(game, "FPSAimTrainer", "Scenarios"), filepath.Join(game, "FPSAimTrainer", "Saved", "SaveGames", "Scenarios"), filepath.Join(filepath.Dir(filepath.Dir(game)), "workshop", "content", "824270"))
	}
	if steam != "" {
		out = append(out, filepath.Join(steam, "steamapps", "workshop", "content", "824270"))
	}
	return out
}

// PollLocal scans bounded roots and waits for the same file metadata on two
// polls, plus a two-second quiet period. Partial Steam writes are not assessed.
// Every downloaded scene is added and evaluated; the active plan's copied blocks stay fixed.
func (s *Service) PollLocal(roots []string, playlists string, now time.Time) {
	s.scenes.mu.Lock()
	defer s.scenes.mu.Unlock()
	if s.scenes.files == nil {
		s.scenes.files = map[string]string{}
		s.scenes.attempts = map[string]time.Time{}
		s.scenes.parsed = map[string]localParsedSCE{}
	}
	if playlists != "" {
		s.pollCurricula(playlists)
	}
	visited := map[string]bool{}
	changed := s.scenes.dirty
	invalidate := func(path, status string) {
		delete(s.scenes.parsed, path)
		s.mu.Lock()
		defer s.mu.Unlock()
		for i := range s.state.Catalog {
			c := &s.state.Catalog[i]
			matches := c.LocalAssessment != nil && c.LocalAssessment.FilePath == path
			if c.LocalAssessment != nil {
				for _, p := range c.LocalAssessment.FilePaths {
					matches = matches || p == path
				}
			}
			if matches && c.LocalAssessment.Status != status {
				c.Mechanics = nil
				if c.Classification == "sce_description" {
					c.Skill = "unknown"
					c.Technique = "unknown"
					c.Classification = "inferred"
					c.Enabled = false
				}
				c.LocalAssessment = &LocalAssessment{Status: status, FilePath: path}
				changed = true
			}
		}
	}
	remaining := 20000
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if remaining <= 0 {
				return fs.SkipAll
			}
			remaining--
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if visited[path] || !strings.EqualFold(filepath.Ext(path), ".sce") {
				return nil
			}
			visited[path] = true
			info, e := d.Info()
			if e != nil || info.Size() > 16<<20 || info.Size() == 0 {
				return nil
			}
			sig := fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
			old := s.scenes.files[path]
			if old == "done:"+sig {
				return nil
			}
			if old != sig {
				invalidate(path, "waiting_for_stable_local_file")
				s.scenes.files[path] = sig
				return nil
			}
			if now.Sub(info.ModTime()) < 2*time.Second {
				return nil
			}
			if attempt, ok := s.scenes.attempts[path]; ok && now.Sub(attempt) < time.Minute {
				return nil
			}
			s.scenes.attempts[path] = now
			b, e := os.ReadFile(path)
			if e != nil {
				return nil
			}
			after, e := os.Stat(path)
			if e != nil || after.Size() != info.Size() || after.ModTime() != info.ModTime() {
				return nil
			}
			name, m, a, e := ParseLocalSCE(b)
			if e != nil {
				invalidate(path, "invalid_local_file")
				return nil
			}
			a.FilePath = path
			a.ObservedAt = now.UTC().Format(time.RFC3339)
			s.scenes.parsed[path] = localParsedSCE{name, m, a}
			s.scenes.files[path] = "done:" + sig
			return nil
		})
	}
	// A same-name local SCE and Workshop SCE may be different revisions. Keep
	// that conflict unknown instead of choosing whichever directory ran last.
	byName := map[string][]localParsedSCE{}
	paths := []string{}
	for path := range s.scenes.parsed {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if !visited[path] && remaining > 0 {
			invalidate(path, "local_file_unavailable")
			delete(s.scenes.files, path)
			delete(s.scenes.attempts, path)
			continue
		}
		p := s.scenes.parsed[path]
		byName[strings.ToLower(p.name)] = append(byName[strings.ToLower(p.name)], p)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	known := map[string]bool{}
	for _, c := range s.state.Catalog {
		known[strings.ToLower(c.Name)] = true
	}
	names := []string{}
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !known[name] && len(s.state.Catalog) < 10000 {
			v := byName[name][0]
			s.state.Catalog = append(s.state.Catalog, scenarioFromLocal(v.name, v.assessment.FilePath, v.mechanics, v.assessment))
			changed = true
		}
	}
	for i := range s.state.Catalog {
		c := &s.state.Catalog[i]
		versions := byName[strings.ToLower(c.Name)]
		if len(versions) == 0 {
			// Persisted assessments must not survive deleted files after an app restart.
			// A truncated scan cannot prove absence, so retain its previous state.
			if remaining > 0 && c.LocalAssessment != nil && (c.LocalAssessment.Status == "file_parsed_model_unfitted" || c.LocalAssessment.Status == "ambiguous_local_versions") {
				status := "local_file_unavailable"
				paths := append([]string{c.LocalAssessment.FilePath}, c.LocalAssessment.FilePaths...)
				for _, path := range paths {
					if visited[path] {
						status = "waiting_for_stable_local_file"
					}
				}
				c.Mechanics = nil
				if c.Classification == "sce_description" {
					c.Skill, c.Technique, c.Classification, c.Enabled = "unknown", "unknown", "inferred", false
				}
				c.LocalAssessment = &LocalAssessment{Status: status, FilePaths: paths}
				changed = true
			}
			continue
		}
		p := versions[0]
		conflict := false
		for _, v := range versions[1:] {
			if v.assessment.FileSHA256 != p.assessment.FileSHA256 {
				conflict = true
			}
		}
		if conflict {
			if c.LocalAssessment == nil || c.LocalAssessment.Status != "ambiguous_local_versions" {
				c.Mechanics = nil
				paths := []string{}
				for _, v := range versions {
					paths = append(paths, v.assessment.FilePath)
				}
				c.LocalAssessment = &LocalAssessment{Status: "ambiguous_local_versions", FilePaths: paths, Issues: []string{"same_name_different_content_hashes"}}
				changed = true
			}
			continue
		}
		if c.LocalAssessment != nil && c.LocalAssessment.Status == "file_parsed_model_unfitted" && c.LocalAssessment.FileSHA256 == p.assessment.FileSHA256 && c.LocalAssessment.ComparisonSchema == comparisonSchema && c.LocalAssessment.Requirements != nil && c.LocalAssessment.Requirements.Schema == sceneanalysis.DescriptorSchema && c.LocalAssessment.Requirements.SemanticVersion == sceneanalysis.SemanticVersion {
			continue
		}
		c.Mechanics = p.mechanics
		c.LocalAssessment = p.assessment
		*c = enrichMechanics(*c)
		changed = true
	}
	if changed {
		refreshLocalRelations(s.state.Catalog)
		if err := s.save(); err != nil {
			s.scenes.dirty = true
			s.state.Error = "本地 SCE 评估缓存保存失败：" + err.Error()
		} else {
			s.scenes.dirty = false
			if strings.HasPrefix(s.state.Error, "本地 SCE 评估缓存保存失败：") {
				s.state.Error = ""
			}
		}
	}
}

func (s *Service) pollCurricula(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) > 2000 {
		entries = entries[:2000]
	}
	for _, entry := range entries {
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || (ext != ".json" && ext != ".plo") {
			continue
		}
		path := filepath.Join(dir, name)
		info, e := entry.Info()
		if e != nil || info.Size() > maxSourceBytes {
			continue
		}
		sig := fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
		if s.scenes.files[path] == "template:"+sig || s.scenes.files[path] == "non-template:"+sig {
			continue
		}
		b, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		items, e := ParsePlaylist(b, Source{URL: "local-playlist:" + name, Title: name})
		if e != nil || len(items) == 0 || items[0].ImportedCurriculum == nil {
			s.scenes.files[path] = "non-template:" + sig
			continue
		}
		s.mu.Lock()
		s.state.Catalog = mergeCatalog(s.state.Catalog, items)
		s.mergeCurricula(items)
		e = s.save()
		s.mu.Unlock()
		if e == nil {
			s.scenes.files[path] = "template:" + sig
		}
	}
}
