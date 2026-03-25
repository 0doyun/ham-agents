package core

import "time"

type Team struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"display_name"`
	MemberAgentIDs []string  `json:"member_agent_ids"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (t Team) Matches(ref string) bool {
	return t.ID == ref || t.DisplayName == ref
}
