package training

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const storageVersion = 2

type statePart struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}
type stateManifest struct {
	Version        int                  `json:"version"`
	StorageVersion int                  `json:"storageVersion"`
	Parts          map[string]statePart `json:"parts"`
}
type stateRepository struct {
	path       string
	parts      map[string]statePart
	definition *PlanDefinition
	legacy     []byte
}
type catalogState struct {
	Catalog   []Scenario `json:"catalog"`
	Discovery Discovery  `json:"discovery"`
}
type analysisRecord struct {
	Mechanics  *Mechanics       `json:"mechanics"`
	Assessment *LocalAssessment `json:"assessment"`
}
type analysisState struct {
	Scenes map[string]analysisRecord `json:"scenes"`
}
type assessmentState struct {
	Coverage    []DemandCoverage              `json:"coverage"`
	Anchors     []PersonalAnchor              `json:"anchors"`
	Evaluations []AnchorEvaluation            `json:"evaluations"`
	Studies     []TrainingStudy               `json:"studies"`
	Contexts    map[string]RunContext         `json:"contexts"`
	Levels      []PlayerLevel                 `json:"levels"`
	Skills      []SkillStatus                 `json:"skills"`
	Priorities  map[string]ThemePriority      `json:"priorities"`
	Tiers       map[string]string             `json:"tiers"`
	Catalog     map[string]*CatalogAssessment `json:"catalog"`
}
type planningState struct {
	Preferences Preferences                `json:"preferences"`
	Curricula   []Curriculum               `json:"curricula"`
	Progress    map[string]RoutineProgress `json:"progress"`
}
type lifecycleState struct {
	Revision         uint64 `json:"revision"`
	Initializing     bool   `json:"initializing"`
	Notice           string `json:"notice"`
	Error            string `json:"error"`
	SearchConfigured bool   `json:"searchConfigured"`
}

var domainParts = []string{"catalog", "analysis", "assessment", "planning", "plan", "execution", "history", "lifecycle"}

func newStateRepository(path string) *stateRepository {
	return &stateRepository{path: path, parts: map[string]statePart{}}
}
func contentHash(data []byte) string   { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func (r *stateRepository) dir() string { return filepath.Join(filepath.Dir(r.path), "training-state") }
func (r *stateRepository) Load(defaultState State) (State, error) {
	raw, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultState, nil
	}
	if err != nil {
		return State{}, err
	}
	var m stateManifest
	if err = json.Unmarshal(raw, &m); err != nil {
		return State{}, fmt.Errorf("训练数据损坏，未覆盖原文件：%w", err)
	}
	if m.StorageVersion == 0 {
		var st State
		if err = json.Unmarshal(raw, &st); err != nil {
			return State{}, err
		}
		if st.Version != 1 {
			return State{}, fmt.Errorf("不支持的训练数据版本 %d", st.Version)
		}
		r.legacy = append([]byte(nil), raw...)
		r.definition = definePlan(st.Plan)
		return st, nil
	}
	if m.StorageVersion != storageVersion || m.Version != 2 {
		return State{}, fmt.Errorf("不支持的训练存储版本 %d", m.StorageVersion)
	}
	if len(m.Parts) != len(domainParts) {
		return State{}, fmt.Errorf("训练数据清单不完整，未覆盖原文件")
	}
	values := map[string][]byte{}
	for _, domain := range domainParts {
		ref, ok := m.Parts[domain]
		hash, err := hex.DecodeString(ref.SHA256)
		if !ok || err != nil || len(hash) != 32 || ref.File != domain+"-"+ref.SHA256+".json" {
			return State{}, fmt.Errorf("训练数据部件路径或校验值无效：%s", domain)
		}
		data, err := os.ReadFile(filepath.Join(r.dir(), ref.File))
		if err != nil {
			return State{}, fmt.Errorf("读取训练部件 %s 失败：%w", domain, err)
		}
		if contentHash(data) != ref.SHA256 {
			return State{}, fmt.Errorf("训练部件 %s 校验失败，未覆盖原文件", domain)
		}
		values[domain] = data
	}
	var catalog catalogState
	var analysis analysisState
	var assessment assessmentState
	var planning planningState
	var definition *PlanDefinition
	var execution *ExecutionProgress
	var history []Plan
	var lifecycle lifecycleState
	targets := map[string]interface{}{"catalog": &catalog, "analysis": &analysis, "assessment": &assessment, "planning": &planning, "plan": &definition, "execution": &execution, "history": &history, "lifecycle": &lifecycle}
	for _, domain := range domainParts {
		if err := json.Unmarshal(values[domain], targets[domain]); err != nil {
			return State{}, fmt.Errorf("训练部件 %s 无法解析：%w", domain, err)
		}
	}
	if (definition == nil) != (execution == nil) || definition != nil && (definition.ID != execution.PlanID || len(definition.Blocks) != len(execution.Blocks)) {
		return State{}, fmt.Errorf("固定计划与执行进度不匹配")
	}
	st := State{Version: 1, Catalog: catalog.Catalog, Discovery: catalog.Discovery, Preferences: planning.Preferences, Curricula: planning.Curricula, CurriculumProgress: planning.Progress, Plan: assemblePlan(definition, execution), History: history, AnchorEvaluations: assessment.Evaluations, PersonalAnchors: assessment.Anchors, TrainingStudies: assessment.Studies, RunContexts: assessment.Contexts, PlayerLevels: assessment.Levels, Skills: assessment.Skills, ThemePriorities: assessment.Priorities, TemplateTiers: assessment.Tiers, Revision: lifecycle.Revision, Initializing: lifecycle.Initializing, Notice: lifecycle.Notice, Error: lifecycle.Error, SearchConfigured: lifecycle.SearchConfigured}
	for i := range st.Catalog {
		c := &st.Catalog[i]
		a := analysis.Scenes[c.Name]
		c.Mechanics, c.LocalAssessment = a.Mechanics, a.Assessment
		c.Evaluation = assessment.Catalog[c.Name]
	}
	st.DemandCoverage = assessment.Coverage
	// The repository owns its immutable graph; assembled active state owns another.
	if st.Plan != nil {
		data, e := json.Marshal(st.Plan)
		if e != nil {
			return State{}, e
		}
		var detached Plan
		if e = json.Unmarshal(data, &detached); e != nil {
			return State{}, e
		}
		st.Plan = &detached
	}
	r.parts = m.Parts
	r.definition = definition
	return st, nil
}
func (r *stateRepository) Save(st State) error {
	catalog := catalogState{Catalog: append([]Scenario(nil), st.Catalog...), Discovery: st.Discovery}
	analysis := analysisState{Scenes: map[string]analysisRecord{}}
	assessment := assessmentState{Anchors: st.PersonalAnchors, Evaluations: st.AnchorEvaluations, Studies: st.TrainingStudies, Contexts: st.RunContexts, Levels: st.PlayerLevels, Skills: st.Skills, Priorities: st.ThemePriorities, Tiers: st.TemplateTiers, Catalog: map[string]*CatalogAssessment{}}
	assessment.Coverage = st.DemandCoverage
	for i := range catalog.Catalog {
		c := &catalog.Catalog[i]
		if c.LocalAssessment != nil || c.Mechanics != nil {
			analysis.Scenes[c.Name] = analysisRecord{c.Mechanics, c.LocalAssessment}
		}
		if c.Evaluation != nil {
			assessment.Catalog[c.Name] = c.Evaluation
		}
		c.LocalAssessment, c.Mechanics, c.Evaluation = nil, nil, nil
	}
	definition := r.definition
	if st.Plan == nil {
		definition = nil
	} else if definition == nil || definition.ID != st.Plan.ID {
		definition = definePlan(st.Plan)
		if definition == nil {
			return fmt.Errorf("固定计划无法序列化")
		}
	}
	values := map[string]interface{}{"catalog": catalog, "analysis": analysis, "assessment": assessment, "planning": planningState{st.Preferences, st.Curricula, st.CurriculumProgress}, "plan": definition, "execution": executionOf(st.Plan), "history": st.History, "lifecycle": lifecycleOf(st)}
	if err := r.commit(values); err != nil {
		return err
	}
	r.definition = definition
	return nil
}
func lifecycleOf(st State) lifecycleState {
	return lifecycleState{st.Revision, st.Initializing, st.Notice, st.Error, st.SearchConfigured}
}

// Pure clock checkpoints serialize only execution and lifecycle. Catalog,
// parser output, assessments and fixed definitions keep their existing bytes.
func (r *stateRepository) SaveExecution(st State) error {
	if len(r.parts) == 0 || st.Plan == nil || r.definition == nil || st.Plan.ID != r.definition.ID {
		return r.Save(st)
	}
	return r.commit(map[string]interface{}{"execution": executionOf(st.Plan), "lifecycle": lifecycleOf(st)})
}
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
func (r *stateRepository) commit(values map[string]interface{}) error {
	if err := os.MkdirAll(r.dir(), 0700); err != nil {
		return err
	}
	next := map[string]statePart{}
	for k, v := range r.parts {
		next[k] = v
	}
	for _, domain := range domainParts {
		v, ok := values[domain]
		if !ok {
			continue
		}
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		hash := contentHash(data)
		ref := statePart{domain + "-" + hash + ".json", hash}
		next[domain] = ref
		if old, ok := r.parts[domain]; ok && old == ref {
			continue
		}
		path := filepath.Join(r.dir(), ref.File)
		if existing, err := os.ReadFile(path); err == nil && contentHash(existing) == hash {
			continue
		}
		if err = atomicWrite(path, data); err != nil {
			return err
		}
	}
	// Preserve the exact readable v1 source once, before the atomic manifest swap.
	if len(r.legacy) > 0 {
		if err := r.preserveLegacy(); err != nil {
			return err
		}
	}
	// Old executables must reject this manifest, rather than mistake it for
	// an empty v1 State and silently overwrite a migrated installation.
	manifest := stateManifest{Version: 2, StorageVersion: storageVersion, Parts: next}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(r.path, data); err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, p := range next {
		keep[p.File] = true
	}
	for _, p := range r.parts {
		keep[p.File] = true
	}
	r.parts = next
	r.legacy = nil
	// Retain the previous generation too. Never delete before commit or touch
	// unrelated files. Orphan parts from interrupted writes are harmless.
	entries, _ := os.ReadDir(r.dir())
	for _, entry := range entries {
		if entry.IsDir() || keep[entry.Name()] {
			continue
		}
		for _, domain := range domainParts {
			if strings.HasPrefix(entry.Name(), domain+"-") && strings.HasSuffix(entry.Name(), ".json") {
				hash := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), domain+"-"), ".json")
				if bytes, e := hex.DecodeString(hash); e == nil && len(bytes) == 32 {
					_ = os.Remove(filepath.Join(r.dir(), entry.Name()))
				}
				break
			}
		}
	}
	return nil
}

func (r *stateRepository) preserveLegacy() error {
	path := r.path + ".legacy-v1.json"
	if existing, err := os.ReadFile(path); err == nil {
		if contentHash(existing) == contentHash(r.legacy) {
			return nil
		}
		path = r.path + ".legacy-v1-" + contentHash(r.legacy)[:12] + ".json"
		if existing, err := os.ReadFile(path); err == nil {
			if contentHash(existing) == contentHash(r.legacy) {
				return nil
			}
			return fmt.Errorf("已有备份与待迁移数据不一致，未覆盖原文件")
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(r.legacy)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}
