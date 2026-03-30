package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// setupDependencies allows tests to inject fakes for filesystem and binary lookups.
type setupDependencies struct {
	lookPath      func(string) (string, error)
	userHomeDir   func() (string, error)
	readFile      func(string) ([]byte, error)
	writeFile     func(string, []byte, os.FileMode) error
	mkdirAll      func(string, os.FileMode) error
	stat          func(string) (os.FileInfo, error)
	launchdStatus func() string
}

func defaultSetupDependencies() setupDependencies {
	return setupDependencies{
		lookPath:      exec.LookPath,
		userHomeDir:   os.UserHomeDir,
		readFile:      os.ReadFile,
		writeFile:     os.WriteFile,
		mkdirAll:      os.MkdirAll,
		stat:          os.Stat,
		launchdStatus: inspectLaunchdStatus,
	}
}

func runSetup(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return runSetupWith(args, stdin, stdout, stderr, defaultSetupDependencies())
}

func runSetupWith(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, deps setupDependencies) error {
	_ = args // reserved for future flags (e.g. --json, --force)

	// Step 1: Check for Claude Code binary.
	claudePath, err := deps.lookPath("claude")
	if err != nil {
		fmt.Fprintln(stdout, "Claude Code CLI not found in PATH.")
		fmt.Fprintln(stdout, "Install it first: https://docs.anthropic.com/en/docs/claude-code")
		fmt.Fprintln(stdout, "Then re-run: ham setup")
		return nil
	}
	fmt.Fprintf(stdout, "Found Claude Code: %s\n", claudePath)

	// Step 2: Resolve settings path.
	home, err := deps.userHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Step 3: Read existing settings (or start fresh).
	settings, err := readClaudeSettings(settingsPath, deps)
	if err != nil {
		return err
	}

	// Step 4: Check if ham hooks are already configured.
	if hasHamHooks(settings) {
		fmt.Fprintln(stdout, "ham hooks are already configured in Claude Code settings.")
		printLaunchdAdvice(stdout, deps)
		return nil
	}

	// Step 5: Show what we'll add and ask for confirmation.
	fmt.Fprintln(stdout, "\nham will add the following hooks to ~/.claude/settings.json:")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "  PreToolUse:    ham hook tool-start \"$TOOL_NAME\"")
	fmt.Fprintln(stdout, "  PostToolUse:   ham hook tool-done \"$TOOL_NAME\"")
	fmt.Fprintln(stdout, "  Notification:  ham hook notification")
	fmt.Fprintln(stdout, "  StopFailure:   ham hook stop-failure")
	fmt.Fprintln(stdout, "  SessionStart:  ham hook session-start")
	fmt.Fprintln(stdout, "  Stop:          ham hook session-end")
	fmt.Fprintln(stdout, "  SessionEnd:    ham hook session-end")
	fmt.Fprintln(stdout, "  SubagentStart: ham hook subagent-start")
	fmt.Fprintln(stdout, "  SubagentStop:  ham hook subagent-stop")
	fmt.Fprintln(stdout, "  TeammateIdle:  ham hook teammate-idle")
	fmt.Fprintln(stdout, "  TaskCreated:   ham hook task-created")
	fmt.Fprintln(stdout, "  TaskCompleted: ham hook task-completed")
	fmt.Fprintln(stdout, "")

	if !confirmPrompt(stdin, stdout, "Apply these hooks?") {
		fmt.Fprintln(stdout, "Aborted.")
		return nil
	}

	// Step 6: Merge hooks and write.
	mergeHamHooks(settings)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')

	if err := deps.mkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create .claude directory: %w", err)
	}
	if err := deps.writeFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	fmt.Fprintln(stdout, "Hooks added to ~/.claude/settings.json")

	// Step 7: Launchd advice.
	printLaunchdAdvice(stdout, deps)

	return nil
}

// readClaudeSettings reads and parses the Claude Code settings file.
// Returns an empty map if the file does not exist.
func readClaudeSettings(path string, deps setupDependencies) (map[string]interface{}, error) {
	data, err := deps.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return settings, nil
}

var hamHookCategories = []string{"PreToolUse", "PostToolUse", "Notification", "StopFailure", "SessionStart", "Stop", "SessionEnd", "SubagentStart", "SubagentStop", "TeammateIdle", "TaskCreated", "TaskCompleted"}

// hasHamHooks returns true if ALL hook categories already contain a "ham hook" command.
func hasHamHooks(settings map[string]interface{}) bool {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, key := range hamHookCategories {
		if !categoryHasHamHook(hooks, key) {
			return false
		}
	}
	return true
}

// categoryHasHamHook checks whether a specific hook category contains a "ham hook" command.
// Supports both legacy flat format {"command": "..."} and new matcher group format
// {"matcher": "", "hooks": [{"type": "command", "command": "..."}]}.
func categoryHasHamHook(hooksMap map[string]interface{}, key string) bool {
	arr, ok := hooksMap[key].([]interface{})
	if !ok {
		return false
	}
	for _, entry := range arr {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		// Legacy flat format: {"command": "ham hook ..."}
		if cmd, ok := entryMap["command"].(string); ok && strings.Contains(cmd, "ham hook") {
			return true
		}
		// New matcher group format: {"hooks": [{"command": "ham hook ..."}]}
		innerHooks, ok := entryMap["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, inner := range innerHooks {
			innerMap, ok := inner.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := innerMap["command"].(string); ok && strings.Contains(cmd, "ham hook") {
				return true
			}
		}
	}
	return false
}

// mergeHamHooks adds ham hook entries to the settings map, preserving existing hooks.
// Skips categories that already contain a ham hook command.
func mergeHamHooks(settings map[string]interface{}) {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = map[string]interface{}{}
		settings["hooks"] = hooks
	}

	hamHookEntries := map[string]map[string]interface{}{
		"PreToolUse":    hamHookMatcherGroup("ham hook tool-start \"$TOOL_NAME\""),
		"PostToolUse":   hamHookMatcherGroup("ham hook tool-done \"$TOOL_NAME\""),
		"Notification":  hamHookMatcherGroup("ham hook notification"),
		"StopFailure":   hamHookMatcherGroup("ham hook stop-failure"),
		"SessionStart":  hamHookMatcherGroup("ham hook session-start"),
		"Stop":          hamHookMatcherGroup("ham hook session-end"),
		"SessionEnd":    hamHookMatcherGroup("ham hook session-end"),
		"SubagentStart": hamHookMatcherGroup("ham hook subagent-start"),
		"SubagentStop":  hamHookMatcherGroup("ham hook subagent-stop"),
		"TeammateIdle":  hamHookMatcherGroup("ham hook teammate-idle"),
		"TaskCreated":   hamHookMatcherGroup("ham hook task-created"),
		"TaskCompleted": hamHookMatcherGroup("ham hook task-completed"),
	}

	for key, hookEntry := range hamHookEntries {
		if categoryHasHamHook(hooks, key) {
			continue
		}
		existing, ok := hooks[key].([]interface{})
		if !ok {
			existing = []interface{}{}
		}
		existing = append(existing, hookEntry)
		hooks[key] = existing
	}
}

// confirmPrompt shows a Y/n prompt and returns true if the user confirms.
func confirmPrompt(in io.Reader, out io.Writer, message string) bool {
	fmt.Fprintf(out, "%s [Y/n] ", message)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}

// hamHookMatcherGroup builds a Claude Code hook matcher group entry.
// Schema: {"matcher": "", "hooks": [{"type": "command", "command": "...", "timeout": 5000}]}
func hamHookMatcherGroup(command string) map[string]interface{} {
	return map[string]interface{}{
		"matcher": "",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": command,
				"timeout": float64(5000),
			},
		},
	}
}

func printLaunchdAdvice(out io.Writer, deps setupDependencies) {
	status := deps.launchdStatus()
	switch status {
	case "running":
		fmt.Fprintln(out, "hamd: running via launchd ✓")
	case "installed_not_running":
		fmt.Fprintln(out, "hamd: installed via launchd but not running.")
		fmt.Fprintln(out, "  Start it with: launchctl kickstart gui/$(id -u)/com.ham-agents.hamd")
	case "not_installed":
		fmt.Fprintln(out, "hamd: not installed via launchd.")
		fmt.Fprintln(out, "  The daemon will auto-start when you run `ham run`, or install it manually.")
	default:
		fmt.Fprintf(out, "hamd: launchd status unknown (%s)\n", status)
	}
}
