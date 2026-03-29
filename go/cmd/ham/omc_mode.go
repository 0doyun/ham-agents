package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var omcModePriority = []string{"autopilot", "ralph", "team", "ultrawork", "ecomode", "ultraqa"}

func detectOmcMode() string {
	for _, key := range []string{"OMC_MODE", "OMX_MODE", "OMX_ACTIVE_MODE", "OMX_ACTIVE_SKILL"} {
		if value := normalizeOmcMode(os.Getenv(key)); value != "" {
			return value
		}
	}
	if os.Getenv("OMX_TEAM_WORKER") != "" || os.Getenv("OMX_TEAM_STATE_ROOT") != "" || os.Getenv("OMX_TEAM_LEADER_CWD") != "" {
		return "team"
	}

	stateRoot := findNearestOmcStateRoot()
	if stateRoot == "" {
		return ""
	}
	if mode := detectActiveModeFromStateRoot(stateRoot); mode != "" {
		return mode
	}
	return detectSkillMode(filepath.Join(stateRoot, "skill-active-state.json"))
}

func normalizeOmcMode(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "autopilot", "ralph", "team", "ultrawork", "ecomode", "ultraqa":
		return trimmed
	default:
		return ""
	}
}

func findNearestOmcStateRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	current := wd
	for {
		candidate := filepath.Join(current, ".omx", "state")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func detectActiveModeFromStateRoot(stateRoot string) string {
	var session struct {
		SessionID string `json:"session_id"`
	}
	_ = readJSONFile(filepath.Join(stateRoot, "session.json"), &session)

	if session.SessionID != "" {
		for _, mode := range omcModePriority {
			if isActiveModeFile(filepath.Join(stateRoot, "sessions", session.SessionID, mode+"-state.json")) {
				return mode
			}
		}
	}

	for _, mode := range omcModePriority {
		if isActiveModeFile(filepath.Join(stateRoot, mode+"-state.json")) {
			return mode
		}
	}
	return ""
}

func detectSkillMode(path string) string {
	var state struct {
		Active bool   `json:"active"`
		Skill  string `json:"skill"`
	}
	if err := readJSONFile(path, &state); err != nil || !state.Active {
		return ""
	}
	return normalizeOmcMode(state.Skill)
}

func isActiveModeFile(path string) bool {
	var state struct {
		Active bool `json:"active"`
	}
	return readJSONFile(path, &state) == nil && state.Active
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
