package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveDaemonLaunchTargetPrefersEnvironmentOverride(t *testing.T) {
	t.Parallel()

	target, err := resolveDaemonLaunchTarget(
		func() (string, error) { return "/tmp/ham", nil },
		func(key string) (string, bool) {
			if key == "HAM_DAEMON_EXECUTABLE" {
				return "/tmp/custom/hamd", true
			}
			return "", false
		},
		func() (string, error) { return "/tmp/project", nil },
		func(string) (string, error) { return "", fmt.Errorf("missing") },
	)
	if err != nil {
		t.Fatalf("resolve daemon target: %v", err)
	}
	if target.Executable != "/tmp/custom/hamd" {
		t.Fatalf("unexpected target %#v", target)
	}
	if len(target.Args) != 1 || target.Args[0] != "serve" {
		t.Fatalf("expected serve args, got %#v", target.Args)
	}
}

func TestResolveDaemonLaunchTargetUsesSiblingBinary(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	hamPath := filepath.Join(root, "ham")
	hamdPath := filepath.Join(root, "hamd")
	if err := os.WriteFile(hamPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write ham binary: %v", err)
	}
	if err := os.WriteFile(hamdPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write hamd binary: %v", err)
	}

	target, err := resolveDaemonLaunchTarget(
		func() (string, error) { return hamPath, nil },
		func(string) (string, bool) { return "", false },
		func() (string, error) { return "/tmp/project", nil },
		func(string) (string, error) { return "", fmt.Errorf("missing") },
	)
	if err != nil {
		t.Fatalf("resolve daemon target: %v", err)
	}
	if target.Executable != hamdPath {
		t.Fatalf("unexpected target %#v", target)
	}
}

func TestResolveDaemonLaunchTargetFallsBackToGoRunFromRepoRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "go", "cmd", "hamd"), 0o755); err != nil {
		t.Fatalf("mkdir hamd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go", "cmd", "hamd", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write hamd main: %v", err)
	}
	workingDirectory := filepath.Join(root, "nested", "dir")
	if err := os.MkdirAll(workingDirectory, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}

	target, err := resolveDaemonLaunchTarget(
		func() (string, error) { return "", fmt.Errorf("missing") },
		func(string) (string, bool) { return "", false },
		func() (string, error) { return workingDirectory, nil },
		func(name string) (string, error) {
			if name == "go" {
				return "/usr/bin/go", nil
			}
			return "", fmt.Errorf("missing")
		},
	)
	if err != nil {
		t.Fatalf("resolve daemon target: %v", err)
	}
	if target.Executable != "/usr/bin/go" {
		t.Fatalf("unexpected executable %#v", target)
	}
	if target.Dir != root {
		t.Fatalf("expected repo root %q, got %q", root, target.Dir)
	}
	if got, want := target.Args, []string{"run", "./go/cmd/hamd", "serve"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("unexpected args %#v", got)
	}
}

func TestEnsureDaemonWithDependenciesStartsAndWaitsForSocket(t *testing.T) {
	t.Parallel()

	current := time.Unix(10, 0)
	dialCalls := 0
	startCalls := 0

	err := ensureDaemonWithDependencies("/tmp/hamd.sock", daemonBootstrapDependencies{
		executablePath: func() (string, error) { return "", fmt.Errorf("missing") },
		lookupEnv: func(key string) (string, bool) {
			if key == "HAM_DAEMON_EXECUTABLE" {
				return "/tmp/hamd", true
			}
			return "", false
		},
		getwd:    func() (string, error) { return "/tmp", nil },
		lookPath: func(string) (string, error) { return "", fmt.Errorf("missing") },
		dial: func(string, time.Duration) (net.Conn, error) {
			dialCalls++
			if dialCalls < 3 {
				return nil, fmt.Errorf("connection refused")
			}
			left, right := net.Pipe()
			_ = right.Close()
			return left, nil
		},
		start: func(target detachedLaunchTarget) error {
			startCalls++
			if target.Executable != "/tmp/hamd" {
				t.Fatalf("unexpected start target %#v", target)
			}
			return nil
		},
		sleep: func(duration time.Duration) {
			current = current.Add(duration)
		},
		now: func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("ensure daemon: %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("expected one start call, got %d", startCalls)
	}
}

func TestEnsureDaemonWithDependenciesTimesOutWhenSocketNeverAppears(t *testing.T) {
	t.Parallel()

	current := time.Unix(20, 0)
	startCalls := 0

	err := ensureDaemonWithDependencies("/tmp/hamd.sock", daemonBootstrapDependencies{
		executablePath: func() (string, error) { return "", fmt.Errorf("missing") },
		lookupEnv: func(key string) (string, bool) {
			if key == "HAM_DAEMON_EXECUTABLE" {
				return "/tmp/hamd", true
			}
			return "", false
		},
		getwd:    func() (string, error) { return "/tmp", nil },
		lookPath: func(string) (string, error) { return "", fmt.Errorf("missing") },
		dial: func(string, time.Duration) (net.Conn, error) {
			return nil, fmt.Errorf("connection refused")
		},
		start: func(detachedLaunchTarget) error {
			startCalls++
			return nil
		},
		sleep: func(duration time.Duration) {
			current = current.Add(duration)
		},
		now: func() time.Time { return current },
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if startCalls != 1 {
		t.Fatalf("expected one start call, got %d", startCalls)
	}
	if got := err.Error(); !containsAll(got, "timed out waiting for daemon socket", "/tmp/hamd.sock") {
		t.Fatalf("unexpected error %q", got)
	}
}

func TestEnsureUIRunningStartsWhenAppIsNotRunning(t *testing.T) {
	t.Parallel()

	startCalls := 0
	err := ensureUIRunningWith(uiLaunchDependencies{
		executablePath: func() (string, error) { return "/tmp/ham", nil },
		lookupEnv: func(key string) (string, bool) {
			if key == "HAM_UI_EXECUTABLE" {
				return "/tmp/ham-menubar", true
			}
			return "", false
		},
		getwd:     func() (string, error) { return "/tmp", nil },
		lookPath:  func(string) (string, error) { return "", fmt.Errorf("missing") },
		isRunning: func(string) (bool, error) { return false, nil },
		start: func(target detachedLaunchTarget) error {
			startCalls++
			if target.Executable != "/tmp/ham-menubar" {
				t.Fatalf("unexpected target %#v", target)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ensure ui running: %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("expected one start call, got %d", startCalls)
	}
}

func TestEnsureUIRunningSkipsLaunchWhenAppIsAlreadyRunning(t *testing.T) {
	t.Parallel()

	startCalls := 0
	err := ensureUIRunningWith(uiLaunchDependencies{
		executablePath: func() (string, error) { return "/tmp/ham", nil },
		lookupEnv: func(key string) (string, bool) {
			if key == "HAM_UI_EXECUTABLE" {
				return "/tmp/ham-menubar", true
			}
			return "", false
		},
		getwd:     func() (string, error) { return "/tmp", nil },
		lookPath:  func(string) (string, error) { return "", fmt.Errorf("missing") },
		isRunning: func(string) (bool, error) { return true, nil },
		start: func(detachedLaunchTarget) error {
			startCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ensure ui running: %v", err)
	}
	if startCalls != 0 {
		t.Fatalf("expected no start call, got %d", startCalls)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
