package settings

import (
	"testing"

	"aimmeow/internal/models"
)

func TestBrandEnvironmentPrecedenceAndLegacyFallback(t *testing.T) {
	t.Setenv("AIMMEOW_BRAVE_API_KEY", "new-key")
	t.Setenv("REFLEKS_BRAVE_API_KEY", "old-key")
	for _, key := range []string{"AIMMEOW_BRAVE_API_KEY", "REFLEKS_BRAVE_API_KEY"} {
		if got := GetEnv(key); got != "new-key" {
			t.Fatalf("%s resolved to %q", key, got)
		}
	}
	t.Setenv("AIMMEOW_BRAVE_API_KEY", "")
	if got := GetEnv("AIMMEOW_BRAVE_API_KEY"); got != "old-key" {
		t.Fatalf("legacy fallback = %q", got)
	}
}

func TestImportedSyncOptInCannotEnableUnconfiguredService(t *testing.T) {
	t.Setenv("AIMMEOW_RUNS_SYNC_URL", "")
	t.Setenv("REFLEKS_RUNS_SYNC_URL", "")
	// An old JSON file (or IPC payload) may claim both capabilities and opt-in.
	s := Sanitize(models.Settings{RunSyncEnabled: true, RunSyncAvailable: true})
	if s.RunSyncAvailable || s.RunSyncEnabled {
		t.Fatalf("unconfigured upload remained enabled: %+v", s)
	}
	t.Setenv("AIMMEOW_RUNS_SYNC_URL", "https://example.test/sync")
	s = Sanitize(models.Settings{RunSyncEnabled: true})
	if !s.RunSyncAvailable || !s.RunSyncEnabled {
		t.Fatal("explicitly configured and enabled service was blocked")
	}
	if Default().RunSyncEnabled {
		t.Fatal("new installations must require upload opt-in")
	}
}
