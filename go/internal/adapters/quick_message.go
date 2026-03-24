package adapters

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type CommandRunner interface {
	Run(name string, args ...string) error
	RunWithInput(name string, input string, args ...string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (ExecRunner) RunWithInput(name string, input string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin = strings.NewReader(input)
	return command.Run()
}

type QuickMessageSender struct {
	runner CommandRunner
}

func NewQuickMessageSender(runner CommandRunner) QuickMessageSender {
	if runner == nil {
		runner = ExecRunner{}
	}
	return QuickMessageSender{runner: runner}
}

func (s QuickMessageSender) Send(target core.OpenTarget, message string) (string, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "", fmt.Errorf("message is required")
	}

	if s.tryTerminalWrite(target, trimmed) == nil {
		return "sent via iTerm automation", nil
	}

	if err := s.runner.RunWithInput("pbcopy", trimmed); err != nil {
		return "", fmt.Errorf("fallback clipboard copy failed: %w", err)
	}

	if err := s.openTarget(target); err != nil {
		return "copied to clipboard but could not open target", nil
	}

	return "copied to clipboard and opened target", nil
}

func (s QuickMessageSender) tryTerminalWrite(target core.OpenTarget, message string) error {
	switch target.Kind {
	case core.OpenTargetKindExternalURL:
		if err := s.runner.Run("open", target.Value); err != nil {
			return err
		}
		return s.runner.Run("osascript", "-e", appleScriptWrite(message))
	case core.OpenTargetKindItermSession:
		if err := s.runner.Run("open", target.Value); err != nil {
			return err
		}
		return s.runner.Run("osascript", "-e", appleScriptWriteToSession(target.SessionID, message))
	case core.OpenTargetKindWorkspace:
		if err := s.runner.Run("open", "-a", "iTerm", target.Value); err != nil {
			return err
		}
		return s.runner.Run("osascript", "-e", appleScriptWrite(message))
	default:
		return fmt.Errorf("unsupported target kind %q", target.Kind)
	}
}

func (s QuickMessageSender) openTarget(target core.OpenTarget) error {
	switch target.Kind {
	case core.OpenTargetKindExternalURL, core.OpenTargetKindItermSession, core.OpenTargetKindWorkspace:
		return s.runner.Run("open", target.Value)
	default:
		return fmt.Errorf("unsupported target kind %q", target.Kind)
	}
}

func appleScriptWrite(message string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(message)
	var script bytes.Buffer
	script.WriteString(`tell application "iTerm"` + "\n")
	script.WriteString(`    activate` + "\n")
	script.WriteString(`    tell current window` + "\n")
	script.WriteString(`        tell current session` + "\n")
	script.WriteString(`            write text "` + escaped + `"` + "\n")
	script.WriteString(`        end tell` + "\n")
	script.WriteString(`    end tell` + "\n")
	script.WriteString(`end tell`)
	return script.String()
}

func appleScriptWriteToSession(sessionID string, message string) string {
	if strings.TrimSpace(sessionID) == "" {
		return appleScriptWrite(message)
	}

	escapedMessage := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(message)
	escapedSessionID := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(sessionID)
	var script bytes.Buffer
	script.WriteString(`tell application "iTerm"` + "\n")
	script.WriteString(`    activate` + "\n")
	script.WriteString(`    repeat with aWindow in windows` + "\n")
	script.WriteString(`        repeat with aTab in tabs of aWindow` + "\n")
	script.WriteString(`            repeat with aSession in sessions of aTab` + "\n")
	script.WriteString(`                if id of aSession is "` + escapedSessionID + `" then` + "\n")
	script.WriteString(`                    tell aSession to write text "` + escapedMessage + `"` + "\n")
	script.WriteString(`                    return` + "\n")
	script.WriteString(`                end if` + "\n")
	script.WriteString(`            end repeat` + "\n")
	script.WriteString(`        end repeat` + "\n")
	script.WriteString(`    end repeat` + "\n")
	script.WriteString(`    tell current window` + "\n")
	script.WriteString(`        tell current session` + "\n")
	script.WriteString(`            write text "` + escapedMessage + `"` + "\n")
	script.WriteString(`        end tell` + "\n")
	script.WriteString(`    end tell` + "\n")
	script.WriteString(`end tell`)
	return script.String()
}
