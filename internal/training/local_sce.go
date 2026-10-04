package training

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type SCEField struct {
	Section string `json:"section"`
	Profile string `json:"profile,omitempty"`
	Key     string `json:"key"`
	Raw     string `json:"raw"`
	Line    int    `json:"line"`
}
type LocalAssessment struct {
	ComparisonSchema     int                   `json:"comparisonSchema,omitempty"`
	FamilyFingerprint    string                `json:"familyFingerprint,omitempty"`
	MapDataSHA256        string                `json:"mapDataSHA256,omitempty"`
	TargetSizes          []TargetSize          `json:"targetSizes,omitempty"`
	Measurements         []FileMeasurement     `json:"measurements,omitempty"`
	PrecisionComparisons []PrecisionComparison `json:"precisionComparisons,omitempty"`
	FilePaths            []string              `json:"filePaths,omitempty"`
	Status               string                `json:"status"`
	FileSHA256           string                `json:"fileSHA256,omitempty"`
	GameVersion          string                `json:"gameVersion,omitempty"`
	ObservedAt           string                `json:"observedAt,omitempty"`
	FilePath             string                `json:"filePath,omitempty"`
	Fields               []SCEField            `json:"fields,omitempty"`
	Issues               []string              `json:"issues,omitempty"`
}
type localParsedSCE struct {
	name       string
	mechanics  *Mechanics
	assessment *LocalAssessment
}
type sceSection struct {
	kind   string
	fields []SCEField
}

func (s sceSection) value(key string) string {
	value := ""
	count := 0
	for _, f := range s.fields {
		if f.Key == key {
			value = f.Raw
			count++
		}
	}
	if count != 1 {
		return ""
	}
	return value
}
func configNumber(raw string) (float64, bool) {
	v, e := strconv.ParseFloat(raw, 64)
	return v, e == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

// ParseLocalSCE reads configuration only. It never imports a .bin cache, invokes
// Steam/downloaders or claims a calibrated total-difficulty score.
func ParseLocalSCE(data []byte) (string, *Mechanics, *LocalAssessment, error) {
	if len(data) == 0 || len(data) > 16<<20 || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return "", nil, nil, fmt.Errorf("SCE 为空、过大或不是文本配置")
	}
	sections := []sceSection{{kind: "Root"}}
	scan := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	scan.Buffer(make([]byte, 4096), 1<<20)
	line := 0
	opaqueConfiguration := false
	for scan.Scan() {
		line++
		s := strings.TrimSpace(scan.Text())
		if s == "[Map Data]" {
			break
		}
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			sections = append(sections, sceSection{kind: s[1 : len(s)-1]})
			continue
		}
		if key, value, ok := strings.Cut(s, "="); ok {
			i := len(sections) - 1
			sections[i].fields = append(sections[i].fields, SCEField{Section: sections[i].kind, Key: strings.TrimSpace(key), Raw: strings.TrimSpace(value), Line: line})
		} else if s != "" && !strings.HasPrefix(s, ";") && !strings.HasPrefix(s, "#") && !strings.HasPrefix(s, "//") {
			opaqueConfiguration = true
		}
	}
	if err := scan.Err(); err != nil {
		return "", nil, nil, err
	}
	root := sections[0]
	name := root.value("Name")
	if name == "" || len(name) > 300 || strings.ContainsAny(name, "\r\n\x00") {
		return "", nil, nil, fmt.Errorf("SCE 缺少唯一有效内部 Name")
	}
	digest := sha256.Sum256(data)
	hash := hex.EncodeToString(digest[:])
	a := &LocalAssessment{Status: "file_parsed_model_unfitted", FileSHA256: hash, GameVersion: root.value("GameVersion"), Fields: append([]SCEField{}, root.fields...), Issues: []string{"map_geometry_and_total_difficulty_not_fitted"}}
	if opaqueConfiguration {
		a.Issues = append(a.Issues, "unsupported_configuration_line")
	}
	index := map[string]int{}
	for i := 1; i < len(sections); i++ {
		p := sections[i].value("Name")
		key := sections[i].kind + "\x00" + strings.ToLower(p)
		if _, exists := index[key]; exists && p != "" {
			return name, nil, nil, fmt.Errorf("重复 profile Name")
		}
		index[key] = i
	}
	active := map[int]bool{}
	var resolve func(string, string)
	resolve = func(kind, token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		for _, suffix := range []string{".bot", ".rot", ".wpn", ".dodge", ".aim", ".char", ".abilmov", ".abilwep", ".abilmelee"} {
			if strings.HasSuffix(token, suffix) {
				token = strings.TrimSuffix(token, suffix)
				break
			}
		}
		i, ok := index[kind+"\x00"+strings.ToLower(token)]
		if !ok {
			a.Issues = append(a.Issues, "unresolved:"+kind+":"+token)
			return
		}
		if active[i] {
			return
		}
		active[i] = true
		p := sections[i]
		if p.value("Name") != token {
			a.Issues = append(a.Issues, "case_variant_reference:"+token)
		}
		for _, f := range p.fields {
			f.Profile = p.value("Name")
			a.Fields = append(a.Fields, f)
		}
		refs := map[string]string{}
		switch kind {
		case "Bot Profile":
			refs = map[string]string{"CharacterProfile": "Character Profile", "DodgeProfileNames": "Dodge Profile", "AimingProfileNames": "Aim Profile"}
		case "Character Profile":
			refs = map[string]string{"WeaponProfileNames": "Weapon Profile", "AbilityProfileNames": "ability"}
		case "Bot Rotation Profile":
			refs = map[string]string{"ProfileNames": "Bot Profile"}
		case "Weapon Ability Profile":
			refs = map[string]string{"WeaponProfile": "Weapon Profile"}
		}
		keys := []string{}
		for key := range refs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, ref := range strings.Split(p.value(key), ";") {
				if ref == "" {
					continue
				}
				dest := refs[key]
				if dest == "ability" {
					switch {
					case strings.HasSuffix(ref, ".abilmov"):
						dest = "Movement Ability Profile"
					case strings.HasSuffix(ref, ".abilwep"):
						dest = "Weapon Ability Profile"
					case strings.HasSuffix(ref, ".abilmelee"):
						dest = "Melee Ability Profile"
					default:
						a.Issues = append(a.Issues, "unsupported_ability:"+ref)
						continue
					}
				}
				resolve(dest, ref)
			}
		}
	}
	resolve("Character Profile", root.value("PlayerProfile"))
	for _, token := range strings.Split(root.value("AddedBots"), ";") {
		if strings.HasSuffix(token, ".rot") {
			resolve("Bot Rotation Profile", token)
		} else {
			resolve("Bot Profile", token)
		}
	}
	m := &Mechanics{FileSHA256: hash, Status: "local_file_parsed_model_unfitted", Role: "unknown", GeometryStatus: "unknown", Tags: []string{}}
	desc := strings.ToLower(root.value("Description"))
	rules := map[string][]string{
		"short_transfer": {"two close, small targets"}, "wide_transfer": {"wide curved wall", "wideflicks"},
		"micro_adjustment": {"microadjustments", "wideflicks and micros", "two close, small targets"},
		"precision":        {"precision on tiny targets", "extra care", "two close, small targets"},
		"time_pressure":    {"before they reach you", "before they despawn or collide", "before it despawns", "falls to the ground"},
		"reflex_window":    {"short-lived target", "lifetime: 500 ms"}, "pacing": {"pacing to increase"},
		"phased_targets": {"number of targets increases every", "after killing 3 big bots"},
		"tracking":       {"hitscan tracking", "track vertically", "tracking while revolving"}, "projectile": {"fast projectile", "midair rockets"},
	}
	for tag, phrases := range rules {
		for _, p := range phrases {
			if strings.Contains(desc, p) {
				m.Tags = append(m.Tags, tag)
				break
			}
		}
	}
	if strings.Contains(desc, "static clicking scenario") {
		m.DeclaredSkill = "static"
	}
	if strings.EqualFold(root.value("ScoreMultAccuracy"), "true") {
		m.Tags = append(m.Tags, "accuracy_constraint")
	}
	if v, ok := configNumber(root.value("Timelimit")); ok && v >= 10 && v <= 3600 {
		m.DeclaredSeconds = int(math.Ceil(v))
	}
	player, ok := index["Character Profile\x00"+strings.ToLower(root.value("PlayerProfile"))]
	if ok {
		for _, w := range strings.Split(sections[player].value("WeaponProfileNames"), ";") {
			w = strings.TrimSuffix(w, ".wpn")
			wi, exists := index["Weapon Profile\x00"+strings.ToLower(w)]
			if !exists {
				continue
			}
			constrained := true
			for _, k := range []string{"MagazineMax", "AmmoPerShot", "AmmoReloadedOnKill", "ReloadTimeFromEmpty"} {
				v, valid := configNumber(sections[wi].value(k))
				constrained = constrained && valid && v > 0
			}
			if constrained {
				m.Tags = append(m.Tags, "reload_constraint")
				break
			}
		}
	}
	sort.Strings(m.Tags)
	sort.Strings(a.Issues)
	calculateFileEvidence(a)
	calculateComparison(data, sections, active, a)
	return name, m, a, nil
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
	s.localScanMu.Lock()
	defer s.localScanMu.Unlock()
	if s.localFiles == nil {
		s.localFiles = map[string]string{}
		s.localAttempts = map[string]time.Time{}
		s.localParsed = map[string]localParsedSCE{}
	}
	if playlists != "" {
		s.pollCurricula(playlists)
	}
	visited := map[string]bool{}
	changed := s.localDirty
	invalidate := func(path, status string) {
		delete(s.localParsed, path)
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
			old := s.localFiles[path]
			if old == "done:"+sig {
				return nil
			}
			if old != sig {
				invalidate(path, "waiting_for_stable_local_file")
				s.localFiles[path] = sig
				return nil
			}
			if now.Sub(info.ModTime()) < 2*time.Second {
				return nil
			}
			if attempt, ok := s.localAttempts[path]; ok && now.Sub(attempt) < time.Minute {
				return nil
			}
			s.localAttempts[path] = now
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
			s.localParsed[path] = localParsedSCE{name, m, a}
			s.localFiles[path] = "done:" + sig
			return nil
		})
	}
	// A same-name local SCE and Workshop SCE may be different revisions. Keep
	// that conflict unknown instead of choosing whichever directory ran last.
	byName := map[string][]localParsedSCE{}
	paths := []string{}
	for path := range s.localParsed {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if !visited[path] && remaining > 0 {
			invalidate(path, "local_file_unavailable")
			delete(s.localFiles, path)
			delete(s.localAttempts, path)
			continue
		}
		p := s.localParsed[path]
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
		if c.LocalAssessment != nil && c.LocalAssessment.Status == "file_parsed_model_unfitted" && c.LocalAssessment.FileSHA256 == p.assessment.FileSHA256 && c.LocalAssessment.ComparisonSchema == comparisonSchema {
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
			s.localDirty = true
			s.state.Error = "本地 SCE 评估缓存保存失败：" + err.Error()
		} else {
			s.localDirty = false
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
		if s.localFiles[path] == "template:"+sig || s.localFiles[path] == "non-template:"+sig {
			continue
		}
		b, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		items, e := ParsePlaylist(b, Source{URL: "local-playlist:" + name, Title: name})
		if e != nil || len(items) == 0 || items[0].ImportedCurriculum == nil {
			s.localFiles[path] = "non-template:" + sig
			continue
		}
		s.mu.Lock()
		s.state.Catalog = mergeCatalog(s.state.Catalog, items)
		s.mergeCurricula(items)
		e = s.save()
		s.mu.Unlock()
		if e == nil {
			s.localFiles[path] = "template:" + sig
		}
	}
}
