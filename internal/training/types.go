// Adaptive training additions, September 2026. Distributed under GPL-3.0.
package training

type Source struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Retrieved string `json:"retrieved"`
}

// A scenario can be scored in several benchmark systems with different cutoffs.
type BenchmarkMembership struct {
	Name             string    `json:"name"`
	BenchmarkID      int       `json:"benchmarkId,omitempty"`
	System           string    `json:"system,omitempty"`
	NativeDifficulty string    `json:"nativeDifficulty,omitempty"`
	Category         string    `json:"category,omitempty"`
	Group            string    `json:"group,omitempty"`
	Thresholds       []float64 `json:"thresholds,omitempty"`
	Ranks            []string  `json:"ranks,omitempty"`
}

type Scenario struct {
	Name               string                `json:"name"`
	Skill              string                `json:"skill"`
	Family             string                `json:"family"`
	Difficulty         string                `json:"difficulty"`
	DifficultySource   string                `json:"difficultySource,omitempty"`
	Technique          string                `json:"technique,omitempty"`
	Seconds            int                   `json:"seconds"`
	Benchmark          string                `json:"benchmark"`
	Thresholds         []float64             `json:"thresholds,omitempty"`
	Benchmarks         []BenchmarkMembership `json:"benchmarks,omitempty"`
	RelatedBenchmarks  []string              `json:"relatedBenchmarks,omitempty"`
	VariantOf          string                `json:"variantOf,omitempty"`
	Preference         string                `json:"preference,omitempty"`
	PersonalDifficulty string                `json:"personalDifficulty,omitempty"`
	Sources            []Source              `json:"sources"`
	Classification     string                `json:"classification"`
	Enabled            bool                  `json:"enabled"`
	Mechanics          *Mechanics            `json:"mechanics,omitempty"`
	LocalAssessment    *LocalAssessment      `json:"localAssessment,omitempty"`
	ImportedCurriculum *Curriculum           `json:"-"`
}

// A content snapshot describes demands, not a calibrated difficulty score.
type Mechanics struct {
	FileSHA256      string   `json:"fileSHA256"`
	DeclaredSkill   string   `json:"declaredSkill,omitempty"`
	DeclaredSeconds int      `json:"declaredSeconds,omitempty"`
	Tags            []string `json:"tags"`
	Status          string   `json:"status"`
	Role            string   `json:"role"`
	GeometryStatus  string   `json:"geometryStatus"`
	AngularSize     *float64 `json:"angularSize"`
	TransitionAngle *float64 `json:"transitionAngle"`
}

type Candidate struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Sharecodes  []string `json:"sharecodes"`
}

type Discovery struct {
	Updated    string      `json:"updated"`
	Imported   int         `json:"imported"`
	Candidates []Candidate `json:"candidates"`
	Warnings   []string    `json:"warnings"`
}

type Preferences struct {
	PlanningPolicy string   `json:"planningPolicy,omitempty"`
	CurriculumID   string   `json:"curriculumId,omitempty"`
	Minutes        int      `json:"minutes"`
	ExecutionMode  string   `json:"executionMode"`
	Focus          string   `json:"focus"`
	Difficulty     string   `json:"difficulty"`
	Benchmark      string   `json:"benchmark"`
	Benchmarks     []string `json:"benchmarks,omitempty"`
	Variety        float64  `json:"variety"`
	ThresholdRatio float64  `json:"thresholdRatio"`
	AutoAdvance    bool     `json:"autoAdvance"`
	AutoDiscover   bool     `json:"autoDiscover"`
}

type SkillStatus struct {
	Skill    string  `json:"skill"`
	Priority float64 `json:"priority"`
	Minutes  float64 `json:"minutes"`
	Samples  int     `json:"samples"`
	Evidence string  `json:"evidence"`
}

type Block struct {
	AnchorScenario     string             `json:"anchorScenario,omitempty"`
	Timing             TimingEstimate     `json:"timing"`
	DifficultyEvidence DifficultyEvidence `json:"difficultyEvidence"`
	Signature          string             `json:"signature,omitempty"`
	Scenario           Scenario           `json:"scenario"`
	Role               string             `json:"role"`
	Budget             int                `json:"budget"`
	PlayCount          int                `json:"playCount"`
	Target             float64            `json:"target"`
	Reason             string             `json:"reason"`
	Cue                string             `json:"cue"`
	Recorded           float64            `json:"recorded"`
	Benchmark          string             `json:"benchmark,omitempty"`
	Runs               int                `json:"runs"`
	Best               float64            `json:"best"`
	Outcome            string             `json:"outcome"`
}

type TimingEstimate struct {
	Seconds       int     `json:"seconds"`
	Source        string  `json:"source"`
	Samples       int     `json:"samples"`
	RecentSeconds float64 `json:"recentSeconds"`
	WeeklySeconds float64 `json:"weeklySeconds"`
}

type DifficultyEvidence struct {
	Benchmarks []BenchmarkMembership `json:"benchmarks,omitempty"`
	Level      string                `json:"level"`
	Source     string                `json:"source"`
	Fit        string                `json:"fit"`
	Samples    int                   `json:"samples"`
}

type Plan struct {
	CurriculumID    string      `json:"curriculumId,omitempty"`
	CurriculumName  string      `json:"curriculumName,omitempty"`
	CurriculumStart int         `json:"curriculumStart,omitempty"`
	CurriculumEnd   int         `json:"curriculumEnd,omitempty"`
	CurriculumTotal int         `json:"curriculumTotal,omitempty"`
	Theme           string      `json:"theme,omitempty"`
	EndedAt         int64       `json:"endedAt,omitempty"`
	ID              string      `json:"id"`
	Created         string      `json:"created"`
	Preferences     Preferences `json:"preferences"`
	Blocks          []Block     `json:"blocks"`
	Warnings        []string    `json:"warnings"`
	Status          string      `json:"status"`
	Index           int         `json:"index"`
	Elapsed         float64     `json:"elapsed"`
	Recorded        float64     `json:"recorded"`
	BlockElapsed    float64     `json:"blockElapsed"`
	LastTick        int64       `json:"lastTick"`
	AcceptAfter     int64       `json:"acceptAfter"`
	Seen            []string    `json:"seen"`
	RemindedBlock   int         `json:"remindedBlock,omitempty"`
	RemindedEnd     bool        `json:"remindedEnd,omitempty"`
	Reminder        string      `json:"reminder,omitempty"`
}

type State struct {
	PlayerLevels     []PlayerLevel `json:"playerLevels"`
	Curricula        []Curriculum  `json:"curricula,omitempty"`
	Version          int           `json:"version"`
	Catalog          []Scenario    `json:"catalog"`
	Discovery        Discovery     `json:"discovery"`
	Preferences      Preferences   `json:"preferences"`
	Plan             *Plan         `json:"plan"`
	History          []Plan        `json:"history"`
	Skills           []SkillStatus `json:"skills"`
	SearchConfigured bool          `json:"searchConfigured"`
	Error            string        `json:"error"`
}

func defaults() Preferences {
	return Preferences{Minutes: 30, PlanningPolicy: "curriculum", ExecutionMode: "playlist", Focus: "auto", Difficulty: "any", Variety: 0.25, ThresholdRatio: 0.9, AutoDiscover: true}
}

var skills = []string{"static", "dynamic", "smooth", "reactive", "switching"}

func validSkill(s string) bool {
	for _, v := range skills {
		if s == v {
			return true
		}
	}
	return false
}
