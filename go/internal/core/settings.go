package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MinQuietHour          = 0
	MaxQuietHour          = 23
	DefaultQuietStartHour = 22
	DefaultQuietEndHour   = 8
	DefaultTheme          = "auto"
	DefaultAnimationSpeed = 1.0
	DefaultHeartbeatMinutes = 0
)

type NotificationSettings struct {
	Done                bool `json:"done"`
	Error               bool `json:"error"`
	WaitingInput        bool `json:"waiting_input"`
	Silence             bool `json:"silence"`
	QuietHoursEnabled   bool `json:"quiet_hours_enabled"`
	QuietHoursStartHour int  `json:"quiet_hours_start_hour"`
	QuietHoursEndHour   int  `json:"quiet_hours_end_hour"`
	PreviewText         bool `json:"preview_text"`
	HeartbeatMinutes    int  `json:"heartbeat_minutes"`
}

type GeneralSettings struct {
	LaunchAtLogin              bool `json:"launch_at_login"`
	CompactMode                bool `json:"compact_mode"`
	ShowMenuBarAnimationAlways bool `json:"show_menu_bar_animation_always"`
}

type AppearanceSettings struct {
	Theme                    string  `json:"theme"`
	AnimationSpeedMultiplier float64 `json:"animation_speed_multiplier"`
	ReduceMotion             bool    `json:"reduce_motion"`
	HamsterSkin              string  `json:"hamster_skin"`
	Hat                      string  `json:"hat"`
	DeskTheme                string  `json:"desk_theme"`
}

type IntegrationSettings struct {
	ItermEnabled     bool            `json:"iterm_enabled"`
	TranscriptDirs   []string        `json:"transcript_dirs"`
	ProviderAdapters map[string]bool `json:"provider_adapters"`
}

type PrivacySettings struct {
	LocalOnlyMode             bool `json:"local_only_mode"`
	EventHistoryRetentionDays int  `json:"event_history_retention_days"`
	TranscriptExcerptStorage  bool `json:"transcript_excerpt_storage"`
}

func (a *AppearanceSettings) UnmarshalJSON(data []byte) error {
	type rawAppearanceSettings struct {
		Theme                    *string  `json:"theme"`
		AnimationSpeedMultiplier *float64 `json:"animation_speed_multiplier"`
		ReduceMotion             *bool    `json:"reduce_motion"`
		HamsterSkin              *string  `json:"hamster_skin"`
		Hat                      *string  `json:"hat"`
		DeskTheme                *string  `json:"desk_theme"`
	}

	var raw rawAppearanceSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	defaults := DefaultSettings().Appearance
	*a = defaults

	if raw.Theme != nil {
		a.Theme = strings.TrimSpace(*raw.Theme)
	}
	if raw.AnimationSpeedMultiplier != nil {
		a.AnimationSpeedMultiplier = *raw.AnimationSpeedMultiplier
	}
	if raw.ReduceMotion != nil {
		a.ReduceMotion = *raw.ReduceMotion
	}
	if raw.HamsterSkin != nil {
		a.HamsterSkin = strings.TrimSpace(*raw.HamsterSkin)
	}
	if raw.Hat != nil {
		a.Hat = strings.TrimSpace(*raw.Hat)
	}
	if raw.DeskTheme != nil {
		a.DeskTheme = strings.TrimSpace(*raw.DeskTheme)
	}

	return nil
}

func (a AppearanceSettings) Validate() error {
	switch strings.TrimSpace(a.Theme) {
	case "auto", "day", "night":
		if a.AnimationSpeedMultiplier < 0.25 || a.AnimationSpeedMultiplier > 3 {
			return fmt.Errorf("appearance animation speed must be between 0.25 and 3")
		}
		return nil
	default:
		return fmt.Errorf("appearance theme must be one of auto, day, or night")
	}
}

func (i *IntegrationSettings) UnmarshalJSON(data []byte) error {
	type rawIntegrationSettings struct {
		ItermEnabled     *bool            `json:"iterm_enabled"`
		TranscriptDirs   *[]string        `json:"transcript_dirs"`
		ProviderAdapters *map[string]bool `json:"provider_adapters"`
	}

	var raw rawIntegrationSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	defaults := DefaultSettings().Integrations
	*i = defaults

	if raw.ItermEnabled != nil {
		i.ItermEnabled = *raw.ItermEnabled
	}
	if raw.TranscriptDirs != nil {
		i.TranscriptDirs = append([]string(nil), (*raw.TranscriptDirs)...)
	}
	if raw.ProviderAdapters != nil {
		i.ProviderAdapters = map[string]bool{}
		for key, value := range *raw.ProviderAdapters {
			i.ProviderAdapters[strings.TrimSpace(key)] = value
		}
	}

	return nil
}

func (n *NotificationSettings) UnmarshalJSON(data []byte) error {
	type rawNotificationSettings struct {
		Done                *bool `json:"done"`
		Error               *bool `json:"error"`
		WaitingInput        *bool `json:"waiting_input"`
		Silence             *bool `json:"silence"`
		QuietHoursEnabled   *bool `json:"quiet_hours_enabled"`
		QuietHoursStartHour *int  `json:"quiet_hours_start_hour"`
		QuietHoursEndHour   *int  `json:"quiet_hours_end_hour"`
		PreviewText         *bool `json:"preview_text"`
		HeartbeatMinutes    *int  `json:"heartbeat_minutes"`
	}

	var raw rawNotificationSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	defaults := DefaultSettings().Notifications
	*n = defaults

	if raw.Done != nil {
		n.Done = *raw.Done
	}
	if raw.Error != nil {
		n.Error = *raw.Error
	}
	if raw.WaitingInput != nil {
		n.WaitingInput = *raw.WaitingInput
	}
	if raw.Silence != nil {
		n.Silence = *raw.Silence
	}
	if raw.QuietHoursEnabled != nil {
		n.QuietHoursEnabled = *raw.QuietHoursEnabled
	}
	if raw.QuietHoursStartHour != nil {
		n.QuietHoursStartHour = *raw.QuietHoursStartHour
	}
	if raw.QuietHoursEndHour != nil {
		n.QuietHoursEndHour = *raw.QuietHoursEndHour
	}
	if raw.PreviewText != nil {
		n.PreviewText = *raw.PreviewText
	}
	if raw.HeartbeatMinutes != nil {
		n.HeartbeatMinutes = *raw.HeartbeatMinutes
	}

	return nil
}

func (n NotificationSettings) Validate() error {
	if n.QuietHoursStartHour < MinQuietHour || n.QuietHoursStartHour > MaxQuietHour {
		return fmt.Errorf("quiet hours start hour must be between %d and %d", MinQuietHour, MaxQuietHour)
	}
	if n.QuietHoursEndHour < MinQuietHour || n.QuietHoursEndHour > MaxQuietHour {
		return fmt.Errorf("quiet hours end hour must be between %d and %d", MinQuietHour, MaxQuietHour)
	}
	switch n.HeartbeatMinutes {
	case 0, 10, 30, 60:
	default:
		return fmt.Errorf("heartbeat minutes must be one of 0, 10, 30, or 60")
	}
	return nil
}

type Settings struct {
	General       GeneralSettings      `json:"general"`
	Notifications NotificationSettings `json:"notifications"`
	Appearance    AppearanceSettings   `json:"appearance"`
	Integrations  IntegrationSettings  `json:"integrations"`
	Privacy       PrivacySettings      `json:"privacy"`
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	type rawSettings struct {
		General       json.RawMessage `json:"general"`
		Notifications json.RawMessage `json:"notifications"`
		Appearance    json.RawMessage `json:"appearance"`
		Integrations  json.RawMessage `json:"integrations"`
		Privacy       json.RawMessage `json:"privacy"`
	}

	var raw rawSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	defaults := DefaultSettings()
	*s = defaults

	if len(raw.General) > 0 {
		if err := json.Unmarshal(raw.General, &s.General); err != nil {
			return err
		}
	}
	if len(raw.Notifications) > 0 {
		if err := json.Unmarshal(raw.Notifications, &s.Notifications); err != nil {
			return err
		}
	}
	if len(raw.Appearance) > 0 {
		if err := json.Unmarshal(raw.Appearance, &s.Appearance); err != nil {
			return err
		}
	}
	if len(raw.Integrations) > 0 {
		if err := json.Unmarshal(raw.Integrations, &s.Integrations); err != nil {
			return err
		}
	}
	if len(raw.Privacy) > 0 {
		if err := json.Unmarshal(raw.Privacy, &s.Privacy); err != nil {
			return err
		}
	}

	return nil
}

func (s Settings) Validate() error {
	if err := s.Notifications.Validate(); err != nil {
		return err
	}
	return s.Appearance.Validate()
}

func DefaultSettings() Settings {
	return Settings{
		General: GeneralSettings{},
		Notifications: NotificationSettings{
			Done:                true,
			Error:               true,
			WaitingInput:        true,
			Silence:             false,
			QuietHoursEnabled:   false,
			QuietHoursStartHour: DefaultQuietStartHour,
			QuietHoursEndHour:   DefaultQuietEndHour,
			PreviewText:         false,
			HeartbeatMinutes:    DefaultHeartbeatMinutes,
		},
		Appearance: AppearanceSettings{
			Theme:                    DefaultTheme,
			AnimationSpeedMultiplier: DefaultAnimationSpeed,
			ReduceMotion:             false,
			HamsterSkin:              "default",
			Hat:                      "none",
			DeskTheme:                "classic",
		},
		Integrations: IntegrationSettings{
			ItermEnabled:     true,
			TranscriptDirs:   []string{},
			ProviderAdapters: map[string]bool{"claude": true, "generic_process": true, "transcript": true},
		},
		Privacy: PrivacySettings{
			LocalOnlyMode:             true,
			EventHistoryRetentionDays: 30,
			TranscriptExcerptStorage:  true,
		},
	}
}
