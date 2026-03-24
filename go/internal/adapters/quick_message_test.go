package adapters

import (
	"errors"
	"strings"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestQuickMessageSenderUsesTerminalWriteWhenCommandsSucceed(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	sender := NewQuickMessageSender(runner)

	result, err := sender.Send(core.OpenTarget{
		Kind:      core.OpenTargetKindItermSession,
		Value:     "iterm2://session/abc",
		SessionID: "abc",
	}, "hello")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result != "sent via iTerm automation" {
		t.Fatalf("unexpected result %q", result)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(runner.calls))
	}
	if runner.calls[0].name != "open" || runner.calls[1].name != "osascript" {
		t.Fatalf("unexpected calls %#v", runner.calls)
	}
	if len(runner.calls[1].args) < 2 || !strings.Contains(runner.calls[1].args[1], `id of aSession is "abc"`) {
		t.Fatalf("expected targeted iTerm session script, got %#v", runner.calls[1].args)
	}
}

func TestQuickMessageSenderFallsBackToClipboardWhenTerminalWriteFails(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{failOpenOnce: true}
	sender := NewQuickMessageSender(runner)

	result, err := sender.Send(core.OpenTarget{
		Kind:  core.OpenTargetKindWorkspace,
		Value: "/tmp/project",
	}, "hello")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result != "copied to clipboard and opened target" {
		t.Fatalf("unexpected result %q", result)
	}
	if len(runner.inputCalls) != 1 || runner.inputCalls[0].name != "pbcopy" {
		t.Fatalf("expected pbcopy fallback, got %#v", runner.inputCalls)
	}
}

type commandCall struct {
	name string
	args []string
}

type recordingRunner struct {
	calls        []commandCall
	inputCalls   []commandCall
	failOpenOnce bool
}

func (r *recordingRunner) Run(name string, args ...string) error {
	r.calls = append(r.calls, commandCall{name: name, args: args})
	if r.failOpenOnce && name == "open" {
		r.failOpenOnce = false
		return errors.New("open failed")
	}
	return nil
}

func (r *recordingRunner) RunWithInput(name string, input string, args ...string) error {
	_ = input
	r.inputCalls = append(r.inputCalls, commandCall{name: name, args: args})
	return nil
}
