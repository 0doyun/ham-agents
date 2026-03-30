package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeSetupDeps(home string) setupDependencies {
	return setupDependencies{
		lookPath:      func(name string) (string, error) { return "/usr/local/bin/" + name, nil },
		userHomeDir:   func() (string, error) { return home, nil },
		readFile:      os.ReadFile,
		writeFile:     os.WriteFile,
		mkdirAll:      os.MkdirAll,
		stat:          os.Stat,
		launchdStatus: func() string { return "running" },
	}
}

func TestSetupCreatesSettingsWhenNoneExist(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	deps := fakeSetupDeps(home)

	stdin := strings.NewReader("y\n")
	var stdout, stderr bytes.Buffer

	err := runSetupWith(nil, stdin, &stdout, &stderr, deps)
	if err != nil {
		t.Fatalf("runSetupWith: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks key missing or not an object")
	}

	for _, key := range hamHookCategories {
		arr, ok := hooks[key].([]interface{})
		if !ok || len(arr) == 0 {
			t.Fatalf("expected %s hook entries, got %v", key, hooks[key])
		}
		entry, ok := arr[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected object entry for %s", key)
		}
		cmd, _ := entry["command"].(string)
		if !strings.Contains(cmd, "ham hook") {
			t.Fatalf("expected ham hook command for %s, got %q", key, cmd)
		}
	}

	if !strings.Contains(stdout.String(), "Hooks added") {
		t.Fatalf("expected success message, got: %s", stdout.String())
	}
}

func TestSetupPreservesExistingSettings(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	existing := map[string]interface{}{
		"theme":   "dark",
		"verbose": true,
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"command": "other-tool pre", "timeout": float64(3000)},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("write existing settings: %v", err)
	}

	deps := fakeSetupDeps(home)
	stdin := strings.NewReader("y\n")
	var stdout, stderr bytes.Buffer

	err := runSetupWith(nil, stdin, &stdout, &stderr, deps)
	if err != nil {
		t.Fatalf("runSetupWith: %v", err)
	}

	resultData, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(resultData, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	// Existing keys preserved.
	if settings["theme"] != "dark" {
		t.Fatalf("theme lost: %v", settings["theme"])
	}
	if settings["verbose"] != true {
		t.Fatalf("verbose lost: %v", settings["verbose"])
	}

	// Existing hook preserved + ham hook appended.
	hooks := settings["hooks"].(map[string]interface{})
	preArr := hooks["PreToolUse"].([]interface{})
	if len(preArr) != 2 {
		t.Fatalf("expected 2 PreToolUse entries (existing + ham), got %d", len(preArr))
	}
	firstCmd := preArr[0].(map[string]interface{})["command"].(string)
	if firstCmd != "other-tool pre" {
		t.Fatalf("existing hook lost, got %q", firstCmd)
	}
	secondCmd := preArr[1].(map[string]interface{})["command"].(string)
	if !strings.Contains(secondCmd, "ham hook tool-start") {
		t.Fatalf("ham hook not appended, got %q", secondCmd)
	}
}

func TestSetupSkipsWhenHamHooksAlreadyExist(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"command": "ham hook tool-start \"$TOOL_NAME\"", "timeout": float64(5000)},
			},
			"PostToolUse": []interface{}{
				map[string]interface{}{"command": "ham hook tool-done \"$TOOL_NAME\"", "timeout": float64(5000)},
			},
			"Notification": []interface{}{
				map[string]interface{}{"command": "ham hook notification", "timeout": float64(5000)},
			},
			"StopFailure": []interface{}{
				map[string]interface{}{"command": "ham hook stop-failure", "timeout": float64(5000)},
			},
			"SessionStart": []interface{}{
				map[string]interface{}{"command": "ham hook session-start", "timeout": float64(5000)},
			},
			"Stop": []interface{}{
				map[string]interface{}{"command": "ham hook session-end", "timeout": float64(5000)},
			},
			"SessionEnd": []interface{}{
				map[string]interface{}{"command": "ham hook session-end", "timeout": float64(5000)},
			},
			"SubagentStart": []interface{}{
				map[string]interface{}{"command": "ham hook subagent-start", "timeout": float64(5000)},
			},
			"SubagentStop": []interface{}{
				map[string]interface{}{"command": "ham hook subagent-stop", "timeout": float64(5000)},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("write existing settings: %v", err)
	}

	deps := fakeSetupDeps(home)
	stdin := strings.NewReader("") // no input needed — should skip
	var stdout, stderr bytes.Buffer

	err := runSetupWith(nil, stdin, &stdout, &stderr, deps)
	if err != nil {
		t.Fatalf("runSetupWith: %v", err)
	}

	if !strings.Contains(stdout.String(), "already configured") {
		t.Fatalf("expected skip message, got: %s", stdout.String())
	}

	// Verify file was NOT rewritten (read original data back).
	resultData, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if !bytes.Equal(resultData, data) {
		t.Fatal("settings file was modified when it should have been left alone")
	}
}

func TestSetupClaudeCodeNotFound(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	deps := fakeSetupDeps(home)
	deps.lookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }

	stdin := strings.NewReader("")
	var stdout, stderr bytes.Buffer

	err := runSetupWith(nil, stdin, &stdout, &stderr, deps)
	if err != nil {
		t.Fatalf("runSetupWith: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Claude Code CLI not found") {
		t.Fatalf("expected not-found message, got: %s", output)
	}
	if !strings.Contains(output, "ham setup") {
		t.Fatalf("expected re-run hint, got: %s", output)
	}
}

func TestSetupAbortedByUser(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	deps := fakeSetupDeps(home)

	stdin := strings.NewReader("n\n")
	var stdout, stderr bytes.Buffer

	err := runSetupWith(nil, stdin, &stdout, &stderr, deps)
	if err != nil {
		t.Fatalf("runSetupWith: %v", err)
	}

	if !strings.Contains(stdout.String(), "Aborted") {
		t.Fatalf("expected abort message, got: %s", stdout.String())
	}

	// Settings file should not exist.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Fatal("settings file should not have been created")
	}
}

func TestHasHamHooksAllThreeConfigured(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse":    []interface{}{map[string]interface{}{"command": "ham hook tool-start \"$TOOL_NAME\""}},
			"PostToolUse":   []interface{}{map[string]interface{}{"command": "ham hook tool-done \"$TOOL_NAME\""}},
			"Notification":  []interface{}{map[string]interface{}{"command": "ham hook notification"}},
			"StopFailure":   []interface{}{map[string]interface{}{"command": "ham hook stop-failure"}},
			"SessionStart":  []interface{}{map[string]interface{}{"command": "ham hook session-start"}},
			"Stop":          []interface{}{map[string]interface{}{"command": "ham hook session-end"}},
			"SessionEnd":    []interface{}{map[string]interface{}{"command": "ham hook session-end"}},
			"SubagentStart": []interface{}{map[string]interface{}{"command": "ham hook subagent-start"}},
			"SubagentStop":  []interface{}{map[string]interface{}{"command": "ham hook subagent-stop"}},
		},
	}
	if !hasHamHooks(settings) {
		t.Fatal("expected hasHamHooks to return true when all required categories are present")
	}
}

func TestHasHamHooksPartialReturnsFalse(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"command": "ham hook tool-start \"$TOOL_NAME\""},
			},
		},
	}
	if hasHamHooks(settings) {
		t.Fatal("expected hasHamHooks to return false when only 1 of 3 categories present")
	}
}

func TestHasHamHooksNotConfigured(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"command": "other-tool pre"},
			},
		},
	}
	if hasHamHooks(settings) {
		t.Fatal("expected hasHamHooks to return false for non-ham hooks")
	}
}

func TestHasHamHooksNoHooksKey(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{"theme": "dark"}
	if hasHamHooks(settings) {
		t.Fatal("expected hasHamHooks to return false when no hooks key")
	}
}

func TestMergeHamHooksSkipsExistingCategories(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"command": "ham hook tool-start \"$TOOL_NAME\"", "timeout": float64(5000)},
			},
		},
	}

	mergeHamHooks(settings)

	hooks := settings["hooks"].(map[string]interface{})

	// PreToolUse should still have only 1 entry (no duplicate).
	preArr := hooks["PreToolUse"].([]interface{})
	if len(preArr) != 1 {
		t.Fatalf("expected 1 PreToolUse entry (existing), got %d", len(preArr))
	}

	for _, key := range []string{"PostToolUse", "Notification", "StopFailure", "SessionStart", "Stop", "SessionEnd", "SubagentStart", "SubagentStop"} {
		arr := hooks[key].([]interface{})
		if len(arr) != 1 {
			t.Fatalf("expected 1 %s entry (newly added), got %d", key, len(arr))
		}
		cmd := arr[0].(map[string]interface{})["command"].(string)
		if !strings.Contains(cmd, "ham hook") {
			t.Fatalf("expected ham hook command for %s, got %q", key, cmd)
		}
	}
}

func TestFormatHookStatusLine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   string
		contains string
	}{
		{"configured", "hooks: configured"},
		{"not_configured", "not configured — running in fallback mode"},
		{"settings_unreadable", "unable to read"},
	}
	for _, tc := range cases {
		line := formatHookStatusLine(tc.status)
		if !strings.Contains(line, tc.contains) {
			t.Fatalf("formatHookStatusLine(%q) = %q, want substring %q", tc.status, line, tc.contains)
		}
	}
}

func TestSetupLaunchdAdvice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   string
		contains string
	}{
		{"running", "running via launchd"},
		{"not_installed", "not installed via launchd"},
		{"installed_not_running", "installed via launchd but not running"},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			deps := fakeSetupDeps(home)
			deps.launchdStatus = func() string { return tc.status }

			// Trigger the already-configured path so launchd advice prints.
			claudeDir := filepath.Join(home, ".claude")
			_ = os.MkdirAll(claudeDir, 0o755)
			existing := map[string]interface{}{
				"hooks": map[string]interface{}{
					"PreToolUse":    []interface{}{map[string]interface{}{"command": "ham hook tool-start \"$TOOL_NAME\""}},
					"PostToolUse":   []interface{}{map[string]interface{}{"command": "ham hook tool-done \"$TOOL_NAME\""}},
					"Notification":  []interface{}{map[string]interface{}{"command": "ham hook notification"}},
					"StopFailure":   []interface{}{map[string]interface{}{"command": "ham hook stop-failure"}},
					"SessionStart":  []interface{}{map[string]interface{}{"command": "ham hook session-start"}},
					"Stop":          []interface{}{map[string]interface{}{"command": "ham hook session-end"}},
					"SessionEnd":    []interface{}{map[string]interface{}{"command": "ham hook session-end"}},
					"SubagentStart": []interface{}{map[string]interface{}{"command": "ham hook subagent-start"}},
					"SubagentStop":  []interface{}{map[string]interface{}{"command": "ham hook subagent-stop"}},
				},
			}
			data, _ := json.MarshalIndent(existing, "", "  ")
			_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

			stdin := strings.NewReader("")
			var stdout, stderr bytes.Buffer

			if err := runSetupWith(nil, stdin, &stdout, &stderr, deps); err != nil {
				t.Fatalf("runSetupWith: %v", err)
			}

			if !strings.Contains(stdout.String(), tc.contains) {
				t.Fatalf("expected %q in output, got: %s", tc.contains, stdout.String())
			}
		})
	}
}
