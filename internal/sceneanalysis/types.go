package sceneanalysis

type SCEField struct {
	Section string `json:"section"`
	Profile string `json:"profile,omitempty"`
	Key     string `json:"key"`
	Raw     string `json:"raw"`
	Line    int    `json:"line"`
}
type LocalAssessment struct {
	Requirements         *Descriptor           `json:"requirements,omitempty"`
	Container            *Container            `json:"container,omitempty"`
	BodySHA256           string                `json:"bodySHA256,omitempty"`
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
type FileMeasurement struct {
	Profile string  `json:"profile,omitempty"`
	Field   string  `json:"field"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Line    int     `json:"line"`
}
type PrecisionComparison struct {
	Reference      string  `json:"reference"`
	ReferenceHash  string  `json:"referenceHash"`
	Profile        string  `json:"profile"`
	RadiusRatio    float64 `json:"radiusRatio"`
	PrecisionDelta float64 `json:"precisionDelta"`
}

type TargetSize struct {
	Profile string   `json:"profile"`
	Radius  float64  `json:"radius"`
	Height  *float64 `json:"height,omitempty"`
}

type PrecisionRelation struct {
	FamilyFingerprint string                `json:"familyFingerprint"`
	Profiles          []PrecisionComparison `json:"profiles"`
	Direction         string                `json:"direction"`
	MaxDelta          float64               `json:"maxDelta"`
	Uniform           bool                  `json:"uniform"`
	UniformDelta      float64               `json:"uniformDelta,omitempty"`
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
