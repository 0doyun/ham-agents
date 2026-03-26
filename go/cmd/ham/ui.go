package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type uiLaunchTarget struct {
	Executable string `json:"executable"`
}

func resolveUICommand(
	args []string,
	executablePath func() (string, error),
	lookupEnv func(string) (string, bool),
	getwd func() (string, error),
	lookPath func(string) (string, error),
) (target uiLaunchTarget, printOnly bool, asJSON bool, err error) {
	flags := flag.NewFlagSet("ui", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSONFlag := flags.Bool("json", false, "emit JSON")
	printFlag := flags.Bool("print", false, "print executable path")
	if err = flags.Parse(args); err != nil {
		return
	}
	if len(flags.Args()) > 0 {
		err = fmt.Errorf("unexpected argument %q", flags.Args()[0])
		return
	}

	executable, err := resolveUIExecutable(executablePath, lookupEnv, getwd, lookPath)
	if err != nil {
		return uiLaunchTarget{}, false, false, err
	}

	return uiLaunchTarget{Executable: executable}, *printFlag, *asJSONFlag, nil
}

func resolveUIExecutable(
	executablePath func() (string, error),
	lookupEnv func(string) (string, bool),
	getwd func() (string, error),
	lookPath func(string) (string, error),
) (string, error) {
	if override, ok := lookupEnv("HAM_UI_EXECUTABLE"); ok && strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override), nil
	}

	currentExecutable, err := executablePath()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(currentExecutable), "ham-menubar")
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() {
			return sibling, nil
		}
	}

	workingDirectory, err := getwd()
	if err == nil {
		buildPath := filepath.Join(workingDirectory, ".build", "arm64-apple-macosx", "debug", "ham-menubar")
		if info, statErr := os.Stat(buildPath); statErr == nil && !info.IsDir() {
			return buildPath, nil
		}
	}

	if found, lookErr := lookPath("ham-menubar"); lookErr == nil {
		return found, nil
	}

	return "", fmt.Errorf("ham-menubar executable could not be resolved")
}

type uiLaunchDependencies struct {
	executablePath func() (string, error)
	lookupEnv      func(string) (string, bool)
	getwd          func() (string, error)
	lookPath       func(string) (string, error)
	isRunning      func(string) (bool, error)
	start          func(detachedLaunchTarget) error
}

func defaultUILaunchDependencies() uiLaunchDependencies {
	return uiLaunchDependencies{
		executablePath: os.Executable,
		lookupEnv:      os.LookupEnv,
		getwd:          os.Getwd,
		lookPath:       exec.LookPath,
		isRunning:      isUIProcessRunning,
		start:          startDetachedProcess,
	}
}

func ensureUIRunning() error {
	return ensureUIRunningWith(defaultUILaunchDependencies())
}

func ensureUIRunningWith(deps uiLaunchDependencies) error {
	executable, err := resolveUIExecutable(deps.executablePath, deps.lookupEnv, deps.getwd, deps.lookPath)
	if err != nil {
		return err
	}

	running, err := deps.isRunning(executable)
	if err != nil {
		return err
	}
	if running {
		return nil
	}

	return deps.start(detachedLaunchTarget{Executable: executable})
}

func isUIProcessRunning(executable string) (bool, error) {
	processName := filepath.Base(executable)

	if pgrepPath, err := exec.LookPath("pgrep"); err == nil {
		cmd := exec.Command(pgrepPath, "-x", processName)
		if err := cmd.Run(); err == nil {
			return true, nil
		} else if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		} else {
			return false, err
		}
	}

	output, err := exec.Command("ps", "-A", "-o", "comm=").Output()
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if filepath.Base(strings.TrimSpace(line)) == processName {
			return true, nil
		}
	}
	return false, nil
}
