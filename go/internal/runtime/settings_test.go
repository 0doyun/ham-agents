package runtime_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestSettingsServicePersistsUpdates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(t.TempDir(), "settings.json")),
	)

	settings, err := service.Get(ctx)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	settings.Notifications.Done = false
	settings.Notifications.Silence = true

	updated, err := service.Update(ctx, settings)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if updated.Notifications.Done {
		t.Fatal("expected done notifications to be disabled")
	}
	if !updated.Notifications.Silence {
		t.Fatal("expected silence notifications to be enabled")
	}
}

func TestSettingsServiceRejectsInvalidQuietHours(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(t.TempDir(), "settings.json")),
	)

	settings := core.DefaultSettings()
	settings.Notifications.QuietHoursStartHour = 30

	if _, err := service.Update(ctx, settings); err == nil {
		t.Fatal("expected invalid quiet hours to be rejected")
	}
}

func TestSettingsServiceRejectsInvalidTheme(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(t.TempDir(), "settings.json")),
	)

	settings := core.DefaultSettings()
	settings.Appearance.Theme = "purple"

	if _, err := service.Update(ctx, settings); err == nil {
		t.Fatal("expected invalid theme to be rejected")
	}
}

func TestSettingsServiceRejectsInvalidAnimationSpeed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(t.TempDir(), "settings.json")),
	)

	settings := core.DefaultSettings()
	settings.Appearance.AnimationSpeedMultiplier = 5

	if _, err := service.Update(ctx, settings); err == nil {
		t.Fatal("expected invalid animation speed to be rejected")
	}
}
