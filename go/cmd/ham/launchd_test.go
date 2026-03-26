package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlistTemplateRendersValidXML(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	err := plistTemplate.Execute(&buf, plistData{
		Label:      "com.ham-agents.hamd",
		Executable: "/usr/local/bin/hamd",
		LogPath:    "/tmp/hamd.log",
	})
	if err != nil {
		t.Fatalf("render plist: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<string>com.ham-agents.hamd</string>") {
		t.Fatalf("plist missing label: %s", output)
	}
	if !strings.Contains(output, "<string>/usr/local/bin/hamd</string>") {
		t.Fatalf("plist missing executable: %s", output)
	}
	if !strings.Contains(output, "<string>serve</string>") {
		t.Fatalf("plist missing serve arg: %s", output)
	}
	if !strings.Contains(output, "<key>KeepAlive</key>") {
		t.Fatalf("plist missing KeepAlive: %s", output)
	}
	if !strings.Contains(output, "<key>RunAtLoad</key>") {
		t.Fatalf("plist missing RunAtLoad: %s", output)
	}
}

func TestLaunchdPlistPathIsUnderLaunchAgents(t *testing.T) {
	t.Parallel()

	path, err := launchdPlistPath()
	if err != nil {
		t.Fatalf("plist path: %v", err)
	}
	if !strings.Contains(path, "LaunchAgents") {
		t.Fatalf("expected LaunchAgents in path, got %q", path)
	}
	if filepath.Base(path) != "com.ham-agents.hamd.plist" {
		t.Fatalf("unexpected plist filename: %q", filepath.Base(path))
	}
}

func TestResolveHamdForInstallUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("HAM_DAEMON_EXECUTABLE", "/tmp/test-hamd")
	path, err := resolveHamdForInstall()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if path != "/tmp/test-hamd" {
		t.Fatalf("expected /tmp/test-hamd, got %q", path)
	}
}

func TestResolveHamdForInstallFailsWhenNotFound(t *testing.T) {
	t.Setenv("HAM_DAEMON_EXECUTABLE", "")
	t.Setenv("HAMD_EXECUTABLE", "")
	t.Setenv("PATH", "/nonexistent")

	_, err := resolveHamdForInstall()
	if err == nil {
		t.Fatal("expected error when hamd not found")
	}
}

func TestUninstallDaemonFromLaunchdWhenNotInstalled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "Library", "LaunchAgents"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := uninstallDaemonFromLaunchd(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
}
