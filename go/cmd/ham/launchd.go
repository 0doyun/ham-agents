package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const launchdLabel = "com.ham-agents.hamd"

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{ .Label }}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{ .Executable }}</string>
		<string>serve</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>StandardOutPath</key>
	<string>{{ .LogPath }}</string>
	<key>StandardErrorPath</key>
	<string>{{ .LogPath }}</string>
	<key>ProcessType</key>
	<string>Background</string>
</dict>
</plist>
`))

type plistData struct {
	Label      string
	Executable string
	LogPath    string
}

func launchdPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"), nil
}

func launchdLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Library", "Logs", "ham-agents", "hamd.log"), nil
}

func installDaemonViaLaunchd() error {
	hamdPath, err := resolveHamdForInstall()
	if err != nil {
		return err
	}

	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}
	logPath, err := launchdLogPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	var buf strings.Builder
	if err := plistTemplate.Execute(&buf, plistData{
		Label:      launchdLabel,
		Executable: hamdPath,
		LogPath:    logPath,
	}); err != nil {
		return fmt.Errorf("render plist: %w", err)
	}

	if err := os.WriteFile(plistPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	// Unload first in case it's already loaded (ignore errors).
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d", os.Getuid()), plistPath).Run()

	if err := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), plistPath).Run(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}

	fmt.Fprintf(os.Stderr, "hamd: started via launchd\n")
	return nil
}

func uninstallDaemonFromLaunchd() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("hamd is not installed via launchd")
		return nil
	}

	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d", os.Getuid()), plistPath).Run()

	if err := os.Remove(plistPath); err != nil {
		return fmt.Errorf("remove plist: %w", err)
	}

	fmt.Println("hamd uninstalled from launchd")
	return nil
}

func printDaemonStatus() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Println("launchd: not installed")
		return nil
	}

	output, err := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)).CombinedOutput()
	if err != nil {
		fmt.Println("launchd: installed but not running")
		return nil
	}

	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "state =") || strings.HasPrefix(trimmed, "pid =") {
			fmt.Printf("launchd: %s\n", trimmed)
		}
	}
	return nil
}

func resolveHamdForInstall() (string, error) {
	// Prefer environment override.
	for _, key := range []string{"HAM_DAEMON_EXECUTABLE", "HAMD_EXECUTABLE"} {
		if override, ok := os.LookupEnv(key); ok && strings.TrimSpace(override) != "" {
			return strings.TrimSpace(override), nil
		}
	}

	// Sibling binary next to the current ham CLI.
	if currentExe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(currentExe), "hamd")
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() {
			return sibling, nil
		}
	}

	// PATH lookup.
	if found, err := exec.LookPath("hamd"); err == nil {
		return found, nil
	}

	return "", fmt.Errorf("hamd executable not found — build it first with `go build -o /usr/local/bin/hamd ./go/cmd/hamd`")
}
