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
}

type AppearanceSettings struct {
	Theme string `json:"theme"`
}

type IntegrationSettings struct {
	ItermEnabled bool `json:"iterm_enabled"`
}

func (a *AppearanceSettings) UnmarshalJSON(data []byte) error {
	type rawAppearanceSettings struct {
		Theme *string `json:"theme"`
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

	return nil
}

func (a AppearanceSettings) Validate() error {
	switch strings.TrimSpace(a.Theme) {
	case "auto", "day", "night":
		return nil
	default:
		return fmt.Errorf("appearance theme must be one of auto, day, or night")
	}
}

func (i *IntegrationSettings) UnmarshalJSON(data []byte) error {
	type rawIntegrationSettings struct {
		ItermEnabled *bool `json:"iterm_enabled"`
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

	return nil
}

func (n NotificationSettings) Validate() error {
	if n.QuietHoursStartHour < MinQuietHour || n.QuietHoursStartHour > MaxQuietHour {
		return fmt.Errorf("quiet hours start hour must be between %d and %d", MinQuietHour, MaxQuietHour)
	}
	if n.QuietHoursEndHour < MinQuietHour || n.QuietHoursEndHour > MaxQuietHour {
		return fmt.Errorf("quiet hours end hour must be between %d and %d", MinQuietHour, MaxQuietHour)
	}
	return nil
}

type Settings struct {
	Notifications NotificationSettings `json:"notifications"`
	Appearance    AppearanceSettings   `json:"appearance"`
	Integrations  IntegrationSettings  `json:"integrations"`
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	type rawSettings struct {
		Notifications json.RawMessage `json:"notifications"`
		Appearance    json.RawMessage `json:"appearance"`
		Integrations  json.RawMessage `json:"integrations"`
	}

	var raw rawSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	defaults := DefaultSettings()
	*s = defaults

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
		Notifications: NotificationSettings{
			Done:                true,
			Error:               true,
			WaitingInput:        true,
			Silence:             false,
			QuietHoursEnabled:   false,
			QuietHoursStartHour: DefaultQuietStartHour,
			QuietHoursEndHour:   DefaultQuietEndHour,
			PreviewText:         false,
		},
		Appearance: AppearanceSettings{
			Theme: DefaultTheme,
		},
		Integrations: IntegrationSettings{
			ItermEnabled: true,
		},
	}
}
