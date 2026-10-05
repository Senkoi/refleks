package training

import (
	"aimmeow/internal/models"
	appsettings "aimmeow/internal/settings"
	"time"
)

type assessmentCache struct {
	evaluationKey string
	evaluatedAt   time.Time
}

func (s *Service) evaluateLocked(runs []models.RunRecord, now time.Time) {
	key := evaluationFingerprint(runs, s.dataRevision)
	if key == s.evaluationKey && now.Sub(s.evaluatedAt) < 5*time.Minute {
		return
	}
	s.syncSessionContextsLocked(runs, now)
	s.updateStudiesLocked(now)
	s.updateAnchorEvaluationsLocked(runs, now)
	s.auditTransferExposureLocked(runs, now)
	result := AssessPlayer(AssessmentInput{Catalog: s.state.Catalog, Runs: runs, Contexts: s.state.RunContexts, Preferences: s.state.Preferences, SessionGap: s.sessionGap, Now: now})
	s.state.PersonalAnchors = result.Anchors
	s.state.PlayerLevels = result.Levels
	s.state.ThemePriorities = result.Priorities
	s.state.TemplateTiers = result.Tiers
	s.state.Skills = result.Skills
	s.state.DemandCoverage = result.Coverage
	for i := range s.state.Catalog {
		s.state.Catalog[i].Evaluation = result.Catalog[s.state.Catalog[i].Name]
	}

	s.state.SearchConfigured = appsettings.GetEnv("AIMMEOW_BRAVE_API_KEY") != ""

	s.evaluationKey, s.evaluatedAt = key, now
}
