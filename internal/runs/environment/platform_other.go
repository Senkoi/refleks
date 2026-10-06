//go:build !windows

package environment

import (
	"time"

	"aimmeow/internal/models"
)

func collectPlatformEnvironment(env *models.RunEnvironment, start, end time.Time) {
	_ = env
	_ = start
	_ = end
}
