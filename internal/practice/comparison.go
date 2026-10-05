package practice

import (
	"aimmeow/internal/models"
	"fmt"
)

// SettingsKey is the existing personal-observation contract. Keeping its
// format also preserves file/protocol bindings for saved execution contexts.
func SettingsKey(s models.RunStatsSummary) string {
	return fmt.Sprintf("%s|%s|%.6f|%.6f|%s|%.6f|%.6f|%.0f|%.3f|%.3f", s.GameVersion, s.SensScale, s.HorizSens, s.VertSens, s.FOVScale, s.FOV, s.DPI, s.Duration, s.AvgTargetScale, s.AvgTimeDilation)
}
func ComparisonKey(s models.RunStatsSummary) string { return s.Hash + "|" + SettingsKey(s) }
