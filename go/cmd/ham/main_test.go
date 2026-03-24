package main

import (
	"strings"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestParseHourFlagAcceptsValidQuietHours(t *testing.T) {
	t.Parallel()

	hour, err := parseHourFlag("--quiet-start-hour=22", "--quiet-start-hour=")
	if err != nil {
		t.Fatalf("parse hour flag: %v", err)
	}
	if hour != 22 {
		t.Fatalf("expected hour 22, got %d", hour)
	}
}

func TestParseHourFlagRejectsInvalidQuietHours(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"--quiet-start-hour=-1",
		"--quiet-start-hour=24",
		"--quiet-start-hour=nope",
	}

	for _, argument := range testCases {
		argument := argument
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			if _, err := parseHourFlag(argument, "--quiet-start-hour="); err == nil {
				t.Fatalf("expected %q to fail", argument)
			}
		})
	}
}

func TestChooseAttachableSessionReturnsOnlySessionWithoutPrompt(t *testing.T) {
	t.Parallel()

	session, err := chooseAttachableSession(strings.NewReader(""), &strings.Builder{}, []core.AttachableSession{
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc"},
	})
	if err != nil {
		t.Fatalf("choose attachable session: %v", err)
	}
	if session.ID != "abc" {
		t.Fatalf("expected abc, got %q", session.ID)
	}
}

func TestChooseAttachableSessionReadsNumericSelection(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	session, err := chooseAttachableSession(strings.NewReader("2\n"), &output, []core.AttachableSession{
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true},
		{ID: "xyz", Title: "Shell", SessionRef: "iterm2://session/xyz"},
	})
	if err != nil {
		t.Fatalf("choose attachable session: %v", err)
	}
	if session.ID != "xyz" {
		t.Fatalf("expected xyz, got %q", session.ID)
	}
	if !strings.Contains(output.String(), "Select iTerm session") {
		t.Fatalf("expected prompt output, got %q", output.String())
	}
}
