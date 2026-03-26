package main

import (
	"os/exec"
	"strings"
)

// detectItermSessionID returns the current iTerm2 session ID via AppleScript,
// or an empty string if iTerm is not running or the query fails.
func detectItermSessionID() string {
	out, err := exec.Command("osascript", "-e",
		`tell application "iTerm" to get id of current session of current tab of current window`,
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
