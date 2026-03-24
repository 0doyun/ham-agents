package core

type NotificationSettings struct {
	Done              bool `json:"done"`
	Error             bool `json:"error"`
	WaitingInput      bool `json:"waiting_input"`
	QuietHoursEnabled bool `json:"quiet_hours_enabled"`
	PreviewText       bool `json:"preview_text"`
}

type Settings struct {
	Notifications NotificationSettings `json:"notifications"`
}

func DefaultSettings() Settings {
	return Settings{
		Notifications: NotificationSettings{
			Done:              true,
			Error:             true,
			WaitingInput:      true,
			QuietHoursEnabled: false,
			PreviewText:       false,
		},
	}
}
