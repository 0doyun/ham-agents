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

var hamHookCategories = []string{"PreToolUse", "PostToolUse", "Notification", "StopFailure", "SessionStart", "Stop", "SessionEnd", "SubagentStart", "SubagentStop"}

// hasHamHooks returns true if ALL three hook categories already contain a "ham hook" command.
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
		cmd, _ := entryMap["command"].(string)
		if strings.Contains(cmd, "ham hook") {
			return true
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
		"PreToolUse":    {"command": "ham hook tool-start \"$TOOL_NAME\"", "timeout": float64(5000)},
		"PostToolUse":   {"command": "ham hook tool-done \"$TOOL_NAME\"", "timeout": float64(5000)},
		"Notification":  {"command": "ham hook notification", "timeout": float64(5000)},
		"StopFailure":   {"command": "ham hook stop-failure", "timeout": float64(5000)},
		"SessionStart":  {"command": "ham hook session-start", "timeout": float64(5000)},
		"Stop":          {"command": "ham hook session-end", "timeout": float64(5000)},
		"SessionEnd":    {"command": "ham hook session-end", "timeout": float64(5000)},
		"SubagentStart": {"command": "ham hook subagent-start", "timeout": float64(5000)},
		"SubagentStop":  {"command": "ham hook subagent-stop", "timeout": float64(5000)},
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
