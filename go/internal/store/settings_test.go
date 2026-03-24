package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestFileSettingsStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.json")
	settingsStore := store.NewFileSettingsStore(path)

	settings, err := settingsStore.Load(ctx)
	if err != nil {
		t.Fatalf("load default settings: %v", err)
	}
	settings.Notifications.PreviewText = true
	settings.Notifications.QuietHoursStartHour = 21
	settings.Notifications.QuietHoursEndHour = 7
	settings.Appearance.Theme = "night"

	if err := settingsStore.Save(ctx, settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	reloaded, err := settingsStore.Load(ctx)
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if !reloaded.Notifications.PreviewText {
		t.Fatal("expected preview text to persist")
	}
	if reloaded.Notifications.QuietHoursStartHour != 21 {
		t.Fatalf("expected quiet start hour 21, got %d", reloaded.Notifications.QuietHoursStartHour)
	}
	if reloaded.Notifications.QuietHoursEndHour != 7 {
		t.Fatalf("expected quiet end hour 7, got %d", reloaded.Notifications.QuietHoursEndHour)
	}
	if reloaded.Appearance.Theme != "night" {
		t.Fatalf("expected theme night, got %q", reloaded.Appearance.Theme)
	}
}

func TestFileSettingsStoreBackfillsQuietHoursDefaultsForLegacyFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.json")
	payload := `{
  "notifications": {
    "done": true,
    "error": true,
    "waiting_input": true,
    "quiet_hours_enabled": true,
    "preview_text": false
  },
  "appearance": {}
}
`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}

	settingsStore := store.NewFileSettingsStore(path)
	settings, err := settingsStore.Load(ctx)
	if err != nil {
		t.Fatalf("load legacy settings: %v", err)
	}

	if settings.Notifications.QuietHoursStartHour != core.DefaultQuietStartHour {
		t.Fatalf("expected quiet start default %d, got %d", core.DefaultQuietStartHour, settings.Notifications.QuietHoursStartHour)
	}
	if settings.Notifications.QuietHoursEndHour != core.DefaultQuietEndHour {
		t.Fatalf("expected quiet end default %d, got %d", core.DefaultQuietEndHour, settings.Notifications.QuietHoursEndHour)
	}
	if settings.Appearance.Theme != core.DefaultTheme {
		t.Fatalf("expected default theme %q, got %q", core.DefaultTheme, settings.Appearance.Theme)
	}
}
