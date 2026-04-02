package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// setupDependencies allows tests to inject fakes for filesystem and binary lookups.
type setupDependencies struct {
	lookPath         func(string) (string, error)
	userHomeDir      func() (string, error)
	readFile         func(string) ([]byte, error)
	writeFile        func(string, []byte, os.FileMode) error
	mkdirAll         func(string, os.FileMode) error
	stat             func(string) (os.FileInfo, error)
	launchdStatus    func() string
	launchdKickstart func() error
	claudeVersion    func() (string, error)
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
		launchdKickstart: func() error {
			return exec.Command("launchctl", "kickstart", fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)).Run()
		},
		claudeVersion: func() (string, error) {
			out, err := exec.Command("claude", "--version").Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		},
	}
}

func runSetup(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return runSetupWith(args, stdin, stdout, stderr, defaultSetupDependencies())
}

func runSetupWith(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, deps setupDependencies) error {
	forceAll := false
	for _, arg := range args {
		if arg == "--full" {
			forceAll = true
		}
	}

	// Step 1: Check for Claude Code binary.
	claudePath, err := deps.lookPath("claude")
	if err != nil {
		fmt.Fprintln(stdout, "Claude Code CLI not found in PATH.")
		fmt.Fprintln(stdout, "Install it first: https://docs.anthropic.com/en/docs/claude-code")
		fmt.Fprintln(stdout, "Then re-run: ham setup")
		return nil
	}
	fmt.Fprintf(stdout, "Found Claude Code: %s\n", claudePath)

	// Step 2: Resolve hook categories based on Claude Code version.
	categories := resolveHookCategories(deps, forceAll)
	fmt.Fprintf(stdout, "Registering %d hook categories.\n", len(categories))

	// Step 3: Resolve settings path.
	home, err := deps.userHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Step 4: Read existing settings (or start fresh).
	settings, err := readClaudeSettings(settingsPath, deps)
	if err != nil {
		return err
	}

	// Step 5: Check if ham hooks are already configured.
	if hasHamHooksForCategories(settings, categories) {
		fmt.Fprintln(stdout, "ham hooks are already configured in Claude Code settings.")
		printLaunchdAdvice(stdout, deps)
		return nil
	}

	// Step 6: Show what we'll add and ask for confirmation.
	entries := hookEntries()
	fmt.Fprintln(stdout, "\nham will add the following hooks to ~/.claude/settings.json:")
	fmt.Fprintln(stdout, "")
	for _, cat := range categories {
		if entry, ok := entries[cat]; ok {
			innerHooks := entry["hooks"].([]interface{})
			innerCmd := innerHooks[0].(map[string]interface{})["command"].(string)
			fmt.Fprintf(stdout, "  %-20s %s\n", cat+":", innerCmd)
		}
	}
	fmt.Fprintln(stdout, "")

	if !confirmPrompt(stdin, stdout, "Apply these hooks?") {
		fmt.Fprintln(stdout, "Aborted.")
		return nil
	}

	// Step 7: Merge hooks and write.
	mergeHamHooksForCategories(settings, categories)

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

	// Step 8: Launchd advice.
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

// hasHamHooks returns true if ALL base hook categories already contain a "ham hook" command.
func hasHamHooks(settings map[string]interface{}) bool {
	return hasHamHooksForCategories(settings, hamHookCategories)
}

// hasHamHooksForCategories returns true if ALL given categories already contain a "ham hook" command.
func hasHamHooksForCategories(settings map[string]interface{}, categories []string) bool {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, key := range categories {
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

// removeHamHooks removes all ham hook entries from settings, preserving non-ham hooks.
// Returns the number of categories that had ham hooks removed.
func removeHamHooks(settings map[string]interface{}) int {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return 0
	}
	removed := 0
	for key, val := range hooks {
		arr, ok := val.([]interface{})
		if !ok {
			continue
		}
		filtered := make([]interface{}, 0, len(arr))
		for _, entry := range arr {
			if !entryIsHamHook(entry) {
				filtered = append(filtered, entry)
			}
		}
		if len(filtered) < len(arr) {
			removed++
			if len(filtered) == 0 {
				delete(hooks, key)
			} else {
				hooks[key] = filtered
			}
		}
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
	return removed
}

// entryIsHamHook returns true if the hook entry contains a "ham hook" command.
func entryIsHamHook(entry interface{}) bool {
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return false
	}
	if cmd, ok := entryMap["command"].(string); ok && strings.Contains(cmd, "ham hook") {
		return true
	}
	innerHooks, ok := entryMap["hooks"].([]interface{})
	if !ok {
		return false
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
	return false
}

// mergeHamHooks adds ham hook entries for the base 12 categories.
func mergeHamHooks(settings map[string]interface{}) {
	mergeHamHooksForCategories(settings, hamHookCategories)
}

// mergeHamHooksForCategories adds ham hook entries for the given categories, preserving existing hooks.
// Skips categories that already contain a ham hook command.
func mergeHamHooksForCategories(settings map[string]interface{}, categories []string) {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = map[string]interface{}{}
		settings["hooks"] = hooks
	}

	allEntries := hookEntries()
	for _, key := range categories {
		hookEntry, ok := allEntries[key]
		if !ok {
			continue
		}
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

// hookEntries returns all known ham hook matcher group entries keyed by category.
func hookEntries() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"PreToolUse":        hamHookMatcherGroup("ham hook tool-start \"$TOOL_NAME\""),
		"PostToolUse":       hamHookMatcherGroup("ham hook tool-done \"$TOOL_NAME\""),
		"PostToolUseFailure": hamHookMatcherGroup("ham hook tool-failed"),
		"Notification":      hamHookMatcherGroup("ham hook notification"),
		"StopFailure":       hamHookMatcherGroup("ham hook stop-failure"),
		"SessionStart":      hamHookMatcherGroup("ham hook session-start"),
		"Stop":              hamHookMatcherGroup("ham hook stop"),
		"SessionEnd":        hamHookMatcherGroup("ham hook session-end"),
		"SubagentStart":     hamHookMatcherGroup("ham hook subagent-start"),
		"SubagentStop":      hamHookMatcherGroup("ham hook subagent-stop"),
		"TeammateIdle":      hamHookMatcherGroup("ham hook teammate-idle"),
		"TaskCreated":       hamHookMatcherGroup("ham hook task-created"),
		"TaskCompleted":     hamHookMatcherGroup("ham hook task-completed"),
		"UserPromptSubmit":  hamHookMatcherGroup("ham hook user-prompt"),
		"PermissionRequest": hamHookMatcherGroup("ham hook permission-request"),
		"PermissionDenied":  hamHookMatcherGroup("ham hook permission-denied"),
		"PreCompact":        hamHookMatcherGroup("ham hook pre-compact"),
		"PostCompact":       hamHookMatcherGroup("ham hook post-compact"),
		"Setup":             hamHookMatcherGroup("ham hook setup"),
		"Elicitation":       hamHookMatcherGroup("ham hook elicitation"),
		"ElicitationResult": hamHookMatcherGroup("ham hook elicitation-result"),
		"ConfigChange":      hamHookMatcherGroup("ham hook config-change"),
		"WorktreeCreate":    hamHookMatcherGroup("ham hook worktree-create"),
		"WorktreeRemove":    hamHookMatcherGroup("ham hook worktree-remove"),
		"InstructionsLoaded": hamHookMatcherGroup("ham hook instructions-loaded"),
		"CwdChanged":        hamHookMatcherGroup("ham hook cwd-changed"),
		"FileChanged":       hamHookMatcherGroup("ham hook file-changed"),
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

// hookMinVersion maps each hook category to the minimum Claude Code version that supports it.
var hookMinVersion = map[string][3]int{
	"PreToolUse":        {1, 0, 38},
	"PostToolUse":       {1, 0, 38},
	"PostToolUseFailure": {1, 0, 38},
	"Stop":              {1, 0, 38},
	"SubagentStop":      {1, 0, 41},
	"PreCompact":        {1, 0, 48},
	"UserPromptSubmit":  {1, 0, 54},
	"SessionStart":      {1, 0, 62},
	"SessionEnd":        {1, 0, 85},
	"Notification":      {2, 0, 37},
	"SubagentStart":     {2, 0, 43},
	"PermissionRequest": {2, 0, 45},
	"Setup":             {2, 1, 10},
	"TeammateIdle":      {2, 1, 33},
	"TaskCompleted":     {2, 1, 33},
	"ConfigChange":      {2, 1, 49},
	"WorktreeCreate":    {2, 1, 50},
	"WorktreeRemove":    {2, 1, 50},
	"InstructionsLoaded": {2, 1, 69},
	"Elicitation":       {2, 1, 76},
	"ElicitationResult": {2, 1, 76},
	"PostCompact":       {2, 1, 76},
	"StopFailure":       {2, 1, 78},
	"CwdChanged":        {2, 1, 83},
	"FileChanged":       {2, 1, 83},
	"TaskCreated":       {2, 1, 84},
	"PermissionDenied":  {2, 1, 89},
}

// allHookCategories lists every hook category in registration order.
var allHookCategories = []string{
	"PreToolUse", "PostToolUse", "PostToolUseFailure", "Stop",
	"SubagentStop", "PreCompact", "UserPromptSubmit",
	"SessionStart", "SessionEnd", "Notification",
	"SubagentStart", "PermissionRequest", "Setup",
	"TeammateIdle", "TaskCompleted", "ConfigChange",
	"WorktreeCreate", "WorktreeRemove", "InstructionsLoaded",
	"Elicitation", "ElicitationResult", "PostCompact",
	"StopFailure", "CwdChanged", "FileChanged",
	"TaskCreated", "PermissionDenied",
}

// versionAtLeast returns true if cur >= req using semver comparison.
func versionAtLeast(cur, req [3]int) bool {
	if cur[0] != req[0] {
		return cur[0] > req[0]
	}
	if cur[1] != req[1] {
		return cur[1] > req[1]
	}
	return cur[2] >= req[2]
}

var versionRegexp = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// parseClaudeVersion extracts major.minor.patch from a version string like "2.1.90 (Claude Code)".
func parseClaudeVersion(raw string) (major, minor, patch int, err error) {
	m := versionRegexp.FindStringSubmatch(raw)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("no version found in %q", raw)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return major, minor, patch, nil
}

// resolveHookCategories returns the hook categories to register based on Claude Code version.
// If forceAll is true or version detection succeeds, returns the appropriate set.
// Falls back to the base 12 categories on detection failure.
func resolveHookCategories(deps setupDependencies, forceAll bool) []string {
	if forceAll {
		return allHookCategories
	}
	if deps.claudeVersion == nil {
		return hamHookCategories
	}
	raw, err := deps.claudeVersion()
	if err != nil {
		return hamHookCategories
	}
	major, minor, patch, err := parseClaudeVersion(raw)
	if err != nil {
		return hamHookCategories
	}
	cur := [3]int{major, minor, patch}
	var cats []string
	for _, cat := range allHookCategories {
		if req, ok := hookMinVersion[cat]; ok && versionAtLeast(cur, req) {
			cats = append(cats, cat)
		}
	}
	if len(cats) == 0 {
		return hamHookCategories
	}
	return cats
}

func printLaunchdAdvice(out io.Writer, deps setupDependencies) {
	status := deps.launchdStatus()
	switch status {
	case "running":
		fmt.Fprintln(out, "hamd: running via launchd ✓")
	case "installed_not_running":
		fmt.Fprint(out, "hamd: starting via launchd... ")
		if err := deps.launchdKickstart(); err != nil {
			fmt.Fprintf(out, "failed: %v\n", err)
		} else {
			fmt.Fprintln(out, "started ✓")
		}
	case "not_installed":
		fmt.Fprintln(out, "hamd: not installed via launchd.")
		fmt.Fprintln(out, "  The daemon will auto-start when you open a claude session.")
	default:
		fmt.Fprintf(out, "hamd: launchd status unknown (%s)\n", status)
	}
}
