package core

import (
	"encoding/json"
	"fmt"
)

const (
	MinQuietHour          = 0
	MaxQuietHour          = 23
	DefaultQuietStartHour = 22
	DefaultQuietEndHour   = 8
)

type NotificationSettings struct {
	Done                bool `json:"done"`
	Error               bool `json:"error"`
	WaitingInput        bool `json:"waiting_input"`
	QuietHoursEnabled   bool `json:"quiet_hours_enabled"`
	QuietHoursStartHour int  `json:"quiet_hours_start_hour"`
	QuietHoursEndHour   int  `json:"quiet_hours_end_hour"`
	PreviewText         bool `json:"preview_text"`
}

func (n *NotificationSettings) UnmarshalJSON(data []byte) error {
	type rawNotificationSettings struct {
		Done                *bool `json:"done"`
		Error               *bool `json:"error"`
		WaitingInput        *bool `json:"waiting_input"`
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
}

func (s Settings) Validate() error {
	return s.Notifications.Validate()
}

func DefaultSettings() Settings {
	return Settings{
		Notifications: NotificationSettings{
			Done:                true,
			Error:               true,
			WaitingInput:        true,
			QuietHoursEnabled:   false,
			QuietHoursStartHour: DefaultQuietStartHour,
			QuietHoursEndHour:   DefaultQuietEndHour,
			PreviewText:         false,
		},
	}
}
