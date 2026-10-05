package training

import "aimmeow/internal/sceneanalysis"

var precisionReferences = sceneanalysis.PrecisionReferences()

type SCEField = sceneanalysis.SCEField
type LocalAssessment = sceneanalysis.LocalAssessment
type FileMeasurement = sceneanalysis.FileMeasurement
type PrecisionComparison = sceneanalysis.PrecisionComparison
type TargetSize = sceneanalysis.TargetSize
type PrecisionRelation = sceneanalysis.PrecisionRelation
type Mechanics = sceneanalysis.Mechanics

func configNumber(raw string) (float64, bool) { return sceneanalysis.ConfigNumber(raw) }
func ParseLocalSCE(data []byte) (string, *Mechanics, *LocalAssessment, error) {
	return sceneanalysis.ParseLocalSCE(data)
}
func calculateFileEvidence(a *LocalAssessment) { sceneanalysis.CalculateFileEvidence(a) }
