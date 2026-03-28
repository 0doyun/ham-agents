package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/store"
)

type doctorReport struct {
	HamAgentsHome string          `json:"ham_agents_home,omitempty"`
	ResolvedRoot  string          `json:"resolved_root"`
	RootSource    string          `json:"root_source"`
	Socket        doctorPathCheck `json:"socket"`
	State         doctorPathCheck `json:"state"`
	Events        doctorPathCheck `json:"events"`
	Settings      doctorPathCheck `json:"settings"`
	Launchd       string          `json:"launchd"`
	HookStatus    string          `json:"hook_status"`
}

type doctorPathCheck struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Kind      string `json:"kind"`
	Reachable bool   `json:"reachable,omitempty"`
}

func gatherDoctorReport(socketPath string) (doctorReport, error) {
	statePath, err := store.DefaultStatePath()
	if err != nil {
		return doctorReport{}, err
	}
	eventPath, err := store.DefaultEventLogPath()
	if err != nil {
		return doctorReport{}, err
	}
	settingsPath, err := store.DefaultSettingsPath()
	if err != nil {
		return doctorReport{}, err
	}

	homeValue := strings.TrimSpace(os.Getenv("HAM_AGENTS_HOME"))
	rootSource := "default"
	resolvedRoot := filepath.Dir(statePath)
	if homeValue != "" {
		rootSource = "env"
		resolvedRoot = homeValue
	}

	return doctorReport{
		HamAgentsHome: homeValue,
		ResolvedRoot:  resolvedRoot,
		RootSource:    rootSource,
		Socket:        inspectSocketPath(socketPath),
		State:         inspectRegularPath(statePath),
		Events:        inspectRegularPath(eventPath),
		Settings:      inspectRegularPath(settingsPath),
		Launchd:       inspectLaunchdStatus(),
		HookStatus:    inspectHookStatus(),
	}, nil
}

func renderDoctorReport(out io.Writer, report doctorReport, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, report)
	}

	hamAgentsHome := report.HamAgentsHome
	if hamAgentsHome == "" {
		hamAgentsHome = "(unset)"
	}

	lines := []string{
		"ham-agents doctor",
		fmt.Sprintf("root_source: %s", report.RootSource),
		fmt.Sprintf("ham_agents_home: %s", hamAgentsHome),
		fmt.Sprintf("resolved_root: %s", report.ResolvedRoot),
		formatDoctorPathLine("socket", report.Socket),
		formatDoctorPathLine("state", report.State),
		formatDoctorPathLine("events", report.Events),
		formatDoctorPathLine("settings", report.Settings),
		fmt.Sprintf("launchd: %s", report.Launchd),
		formatHookStatusLine(report.HookStatus),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func formatDoctorPathLine(label string, check doctorPathCheck) string {
	status := check.Kind
	if check.Kind == "unix_socket" {
		if check.Reachable {
			status = "reachable_socket"
		} else {
			status = "socket_not_listening"
		}
	}
	return fmt.Sprintf("%s: %s\t%s", label, status, check.Path)
}

func inspectRegularPath(path string) doctorPathCheck {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doctorPathCheck{Path: path, Exists: false, Kind: "missing"}
		}
		return doctorPathCheck{Path: path, Exists: false, Kind: "unreadable"}
	}

	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return doctorPathCheck{Path: path, Exists: true, Kind: kind}
}

func inspectLaunchdStatus() string {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return "unknown"
	}
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return "not_installed"
	}
	if err := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)).Run(); err != nil {
		return "installed_not_running"
	}
	return "running"
}

func inspectSocketPath(path string) doctorPathCheck {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doctorPathCheck{Path: path, Exists: false, Kind: "missing"}
		}
		return doctorPathCheck{Path: path, Exists: false, Kind: "unreadable"}
	}

	if info.Mode()&os.ModeSocket == 0 {
		kind := "file"
		if info.IsDir() {
			kind = "directory"
		}
		return doctorPathCheck{Path: path, Exists: true, Kind: kind}
	}

	check := doctorPathCheck{Path: path, Exists: true, Kind: "unix_socket"}
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err == nil {
		check.Reachable = true
		_ = conn.Close()
	}
	return check
}

func inspectHookStatus() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "unknown"
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings, err := readClaudeSettings(settingsPath, defaultSetupDependencies())
	if err != nil {
		return "settings_unreadable"
	}
	if len(settings) == 0 {
		return "not_configured"
	}
	if hasHamHooks(settings) {
		return "configured"
	}
	return "not_configured"
}

func formatHookStatusLine(status string) string {
	switch status {
	case "configured":
		return "hooks: configured"
	case "not_configured":
		return "hooks: not configured — running in fallback mode, run 'ham setup' to enable accurate state tracking"
	case "settings_unreadable":
		return "hooks: unable to read ~/.claude/settings.json"
	default:
		return fmt.Sprintf("hooks: %s", status)
	}
}
