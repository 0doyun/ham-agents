package main

import "github.com/ham-agents/ham-agents/go/internal/adapters"

func detectSessionRef() string {
	if ref := adapters.NewTmuxAdapter(nil).CurrentPaneSessionRef(); ref != "" {
		return ref
	}
	if sessionID := detectItermSessionID(); sessionID != "" {
		return "iterm2://session/" + sessionID
	}
	return ""
}
