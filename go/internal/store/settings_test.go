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
	settings.Notifications.Silence = true
	settings.Notifications.HeartbeatMinutes = 30
	settings.Notifications.QuietHoursStartHour = 21
	settings.Notifications.QuietHoursEndHour = 7
	settings.General.LaunchAtLogin = true
	settings.General.CompactMode = true
	settings.General.ShowMenuBarAnimationAlways = true
	settings.Appearance.Theme = "night"
	settings.Appearance.AnimationSpeedMultiplier = 1.5
	settings.Appearance.ReduceMotion = true
	settings.Appearance.HamsterSkin = "golden"
	settings.Appearance.Hat = "cap"
	settings.Appearance.DeskTheme = "night-shift"
	settings.Integrations.ItermEnabled = false
	settings.Integrations.TranscriptDirs = []string{"/tmp/logs", "/tmp/more"}
	settings.Integrations.ProviderAdapters = map[string]bool{"claude": true, "transcript": false}
	settings.Privacy.LocalOnlyMode = false
	settings.Privacy.EventHistoryRetentionDays = 14
	settings.Privacy.TranscriptExcerptStorage = false

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
	if !reloaded.Notifications.Silence {
		t.Fatal("expected silence notifications to persist")
	}
	if reloaded.Notifications.HeartbeatMinutes != 30 {
		t.Fatalf("expected heartbeat minutes 30, got %d", reloaded.Notifications.HeartbeatMinutes)
	}
	if reloaded.Notifications.QuietHoursStartHour != 21 {
		t.Fatalf("expected quiet start hour 21, got %d", reloaded.Notifications.QuietHoursStartHour)
	}
	if reloaded.Notifications.QuietHoursEndHour != 7 {
		t.Fatalf("expected quiet end hour 7, got %d", reloaded.Notifications.QuietHoursEndHour)
	}
	if !reloaded.General.LaunchAtLogin || !reloaded.General.CompactMode || !reloaded.General.ShowMenuBarAnimationAlways {
		t.Fatalf("expected general settings to persist, got %#v", reloaded.General)
	}
	if reloaded.Appearance.Theme != "night" {
		t.Fatalf("expected theme night, got %q", reloaded.Appearance.Theme)
	}
	if reloaded.Appearance.AnimationSpeedMultiplier != 1.5 {
		t.Fatalf("expected animation speed 1.5, got %f", reloaded.Appearance.AnimationSpeedMultiplier)
	}
	if !reloaded.Appearance.ReduceMotion {
		t.Fatal("expected reduce motion to persist")
	}
	if reloaded.Appearance.HamsterSkin != "golden" || reloaded.Appearance.Hat != "cap" || reloaded.Appearance.DeskTheme != "night-shift" {
		t.Fatalf("expected appearance extras to persist, got %#v", reloaded.Appearance)
	}
	if reloaded.Integrations.ItermEnabled {
		t.Fatal("expected iTerm integration to persist as disabled")
	}
	if len(reloaded.Integrations.TranscriptDirs) != 2 || reloaded.Integrations.TranscriptDirs[0] != "/tmp/logs" {
		t.Fatalf("expected transcript dirs to persist, got %#v", reloaded.Integrations.TranscriptDirs)
	}
	if reloaded.Integrations.ProviderAdapters["transcript"] {
		t.Fatalf("expected provider adapter override to persist, got %#v", reloaded.Integrations.ProviderAdapters)
	}
	if reloaded.Privacy.LocalOnlyMode || reloaded.Privacy.EventHistoryRetentionDays != 14 || reloaded.Privacy.TranscriptExcerptStorage {
		t.Fatalf("expected privacy settings to persist, got %#v", reloaded.Privacy)
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
	if settings.Notifications.Silence {
		t.Fatal("expected silence flag to default to disabled when absent")
	}
	if settings.Notifications.QuietHoursEndHour != core.DefaultQuietEndHour {
		t.Fatalf("expected quiet end default %d, got %d", core.DefaultQuietEndHour, settings.Notifications.QuietHoursEndHour)
	}
	if settings.Notifications.HeartbeatMinutes != core.DefaultHeartbeatMinutes {
		t.Fatalf("expected heartbeat default %d, got %d", core.DefaultHeartbeatMinutes, settings.Notifications.HeartbeatMinutes)
	}
	if settings.Appearance.Theme != core.DefaultTheme {
		t.Fatalf("expected default theme %q, got %q", core.DefaultTheme, settings.Appearance.Theme)
	}
	if settings.Appearance.AnimationSpeedMultiplier != core.DefaultAnimationSpeed {
		t.Fatalf("expected default animation speed %f, got %f", core.DefaultAnimationSpeed, settings.Appearance.AnimationSpeedMultiplier)
	}
	if settings.Appearance.ReduceMotion {
		t.Fatal("expected reduce motion to default to disabled")
	}
	if settings.Appearance.HamsterSkin != "default" || settings.Appearance.Hat != "none" || settings.Appearance.DeskTheme != "classic" {
		t.Fatalf("expected default appearance extras, got %#v", settings.Appearance)
	}
	if !settings.Integrations.ItermEnabled {
		t.Fatal("expected iTerm integration default to be enabled")
	}
	if len(settings.Integrations.TranscriptDirs) != 0 {
		t.Fatalf("expected default transcript dirs to be empty, got %#v", settings.Integrations.TranscriptDirs)
	}
	if !settings.Privacy.LocalOnlyMode || settings.Privacy.EventHistoryRetentionDays != 30 || !settings.Privacy.TranscriptExcerptStorage {
		t.Fatalf("unexpected default privacy %#v", settings.Privacy)
	}
}
