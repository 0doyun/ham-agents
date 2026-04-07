package ipc_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/ipc"
)

// TestContract_HookCommands_JSONRoundTrip verifies that every hook command's
// Request survives a JSON marshal/unmarshal round-trip with all fields intact.
// This guards against silent schema drift (e.g. a field renamed or omitempty
// eating a zero value that matters).
func TestContract_HookCommands_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  ipc.Request
	}{
		{
			name: "hook.tool-start",
			req: ipc.Request{
				Command:          ipc.CommandHookToolStart,
				AgentID:          "agent-001",
				SessionID:        "sess-abc",
				SessionRef:       "ref-1",
				ToolName:         "Bash",
				ToolInputPreview: "go test ./...",
				OmcMode:          "autopilot",
			},
		},
		{
			name: "hook.tool-done",
			req: ipc.Request{
				Command:          ipc.CommandHookToolDone,
				AgentID:          "agent-002",
				SessionID:        "sess-abc",
				SessionRef:       "ref-1",
				ToolName:         "Read",
				ToolInputPreview: "/etc/hosts",
				OmcMode:          "standard",
			},
		},
		{
			name: "hook.notification",
			req: ipc.Request{
				Command:          ipc.CommandHookNotification,
				AgentID:          "agent-003",
				SessionID:        "sess-def",
				SessionRef:       "ref-2",
				NotificationType: "permission_prompt",
				OmcMode:          "",
			},
		},
		{
			name: "hook.stop-failure",
			req: ipc.Request{
				Command:   ipc.CommandHookStopFailure,
				AgentID:   "agent-004",
				SessionID: "sess-def",
				SessionRef: "ref-2",
				ErrorType: "timeout",
				OmcMode:   "autopilot",
			},
		},
		{
			name: "hook.session-start",
			req: ipc.Request{
				Command:     ipc.CommandHookSessionStart,
				AgentID:     "agent-005",
				SessionID:   "sess-ghi",
				SessionRef:  "ref-3",
				ProjectPath: "/home/user/project",
				OmcMode:     "standard",
			},
		},
		{
			name: "hook.session-end",
			req: ipc.Request{
				Command:    ipc.CommandHookSessionEnd,
				AgentID:    "agent-006",
				SessionID:  "sess-ghi",
				SessionRef: "ref-3",
				OmcMode:    "standard",
			},
		},
		{
			name: "hook.agent-spawned",
			req: ipc.Request{
				Command:     ipc.CommandHookAgentSpawned,
				AgentID:     "agent-007",
				SessionID:   "sess-jkl",
				SessionRef:  "ref-4",
				Description: "subagent-executor",
				OmcMode:     "autopilot",
			},
		},
		{
			name: "hook.agent-finished",
			req: ipc.Request{
				Command:     ipc.CommandHookAgentFinished,
				AgentID:     "agent-008",
				SessionID:   "sess-jkl",
				SessionRef:  "ref-4",
				Description: "subagent-executor",
				LastMessage: "Task completed successfully.",
				OmcMode:     "autopilot",
			},
		},
		{
			name: "hook.stop",
			req: ipc.Request{
				Command:     ipc.CommandHookStop,
				AgentID:     "agent-009",
				SessionID:   "sess-mno",
				SessionRef:  "ref-5",
				LastMessage: "All done.",
				OmcMode:     "",
			},
		},
		{
			name: "hook.teammate-idle",
			req: ipc.Request{
				Command:      ipc.CommandHookTeammateIdle,
				AgentID:      "agent-010",
				SessionID:    "sess-mno",
				SessionRef:   "ref-5",
				TeammateName: "executor-1",
				TeamRole:     "executor",
				OmcMode:      "team",
			},
		},
		{
			name: "hook.task-created",
			req: ipc.Request{
				Command:         ipc.CommandHookTaskCreated,
				AgentID:         "agent-011",
				SessionID:       "sess-pqr",
				SessionRef:      "ref-6",
				TaskName:        "implement-feature",
				TaskDescription: "Add retry logic to the HTTP client.",
				OmcMode:         "team",
			},
		},
		{
			name: "hook.task-completed",
			req: ipc.Request{
				Command:    ipc.CommandHookTaskCompleted,
				AgentID:    "agent-012",
				SessionID:  "sess-pqr",
				SessionRef: "ref-6",
				TaskName:   "implement-feature",
				OmcMode:    "team",
			},
		},
		{
			name: "hook.tool-failed",
			req: ipc.Request{
				Command:     ipc.CommandHookToolFailed,
				AgentID:     "agent-013",
				SessionID:   "sess-stu",
				SessionRef:  "ref-7",
				ToolName:    "Bash",
				Description: "exit status 1",
				IsInterrupt: true,
				OmcMode:     "",
			},
		},
		{
			name: "hook.user-prompt",
			req: ipc.Request{
				Command:    ipc.CommandHookUserPrompt,
				AgentID:    "agent-014",
				SessionID:  "sess-stu",
				SessionRef: "ref-7",
				Prompt:     "What is the capital of France?",
				OmcMode:    "",
			},
		},
		{
			name: "hook.permission-request",
			req: ipc.Request{
				Command:    ipc.CommandHookPermissionReq,
				AgentID:    "agent-015",
				SessionID:  "sess-vwx",
				SessionRef: "ref-8",
				ToolName:   "Bash",
				OmcMode:    "",
			},
		},
		{
			name: "hook.permission-denied",
			req: ipc.Request{
				Command:     ipc.CommandHookPermissionDenied,
				AgentID:     "agent-016",
				SessionID:   "sess-vwx",
				SessionRef:  "ref-8",
				ToolName:    "Write",
				Description: "user rejected",
				OmcMode:     "",
			},
		},
		{
			name: "hook.pre-compact",
			req: ipc.Request{
				Command:        ipc.CommandHookPreCompact,
				AgentID:        "agent-017",
				SessionID:      "sess-yza",
				SessionRef:     "ref-9",
				CompactTrigger: "auto",
				OmcMode:        "",
			},
		},
		{
			name: "hook.post-compact (nested summary)",
			req: ipc.Request{
				Command:        ipc.CommandHookPostCompact,
				AgentID:        "agent-018",
				SessionID:      "sess-yza",
				SessionRef:     "ref-9",
				CompactTrigger: "auto",
				CompactSummary: "Summarized 200 messages into 5.",
				OmcMode:        "",
			},
		},
		{
			name: "hook.setup",
			req: ipc.Request{
				Command:    ipc.CommandHookSetup,
				AgentID:    "agent-019",
				SessionID:  "sess-bcd",
				SessionRef: "ref-10",
				OmcMode:    "standard",
			},
		},
		{
			name: "hook.elicitation",
			req: ipc.Request{
				Command:    ipc.CommandHookElicitation,
				AgentID:    "agent-020",
				SessionID:  "sess-bcd",
				SessionRef: "ref-10",
				OmcMode:    "",
			},
		},
		{
			name: "hook.elicitation-result",
			req: ipc.Request{
				Command:    ipc.CommandHookElicitationResult,
				AgentID:    "agent-021",
				SessionID:  "sess-efg",
				SessionRef: "ref-11",
				OmcMode:    "",
			},
		},
		{
			name: "hook.config-change",
			req: ipc.Request{
				Command:     ipc.CommandHookConfigChange,
				AgentID:     "agent-022",
				SessionID:   "sess-efg",
				SessionRef:  "ref-11",
				Description: "~/.claude/settings.json",
				OmcMode:     "",
			},
		},
		{
			name: "hook.worktree-create",
			req: ipc.Request{
				Command:      ipc.CommandHookWorktreeCreate,
				AgentID:      "agent-023",
				SessionID:    "sess-hij",
				SessionRef:   "ref-12",
				WorktreeName: "feature-branch",
				OmcMode:      "",
			},
		},
		{
			name: "hook.worktree-remove",
			req: ipc.Request{
				Command:      ipc.CommandHookWorktreeRemove,
				AgentID:      "agent-024",
				SessionID:    "sess-hij",
				SessionRef:   "ref-12",
				WorktreePath: "/tmp/worktrees/feature-branch",
				OmcMode:      "",
			},
		},
		{
			name: "hook.instructions-loaded",
			req: ipc.Request{
				Command:    ipc.CommandHookInstructions,
				AgentID:    "agent-025",
				SessionID:  "sess-klm",
				SessionRef: "ref-13",
				FilePath:   "/home/user/project/CLAUDE.md",
				OmcMode:    "",
			},
		},
		{
			name: "hook.cwd-changed",
			req: ipc.Request{
				Command:    ipc.CommandHookCwdChanged,
				AgentID:    "agent-026",
				SessionID:  "sess-klm",
				SessionRef: "ref-13",
				OldCwd:     "/home/user/old",
				NewCwd:     "/home/user/new",
				OmcMode:    "",
			},
		},
		{
			name: "hook.file-changed",
			req: ipc.Request{
				Command:    ipc.CommandHookFileChanged,
				AgentID:    "agent-027",
				SessionID:  "sess-nop",
				SessionRef: "ref-14",
				FilePath:   "/home/user/project/main.go",
				FileEvent:  "modified",
				OmcMode:    "",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("marshal Request: %v", err)
			}

			var got ipc.Request
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal Request: %v", err)
			}

			if !reflect.DeepEqual(tc.req, got) {
				t.Errorf("round-trip mismatch for %q:\n  want: %+v\n  got:  %+v", tc.name, tc.req, got)
			}

			if got.Command != tc.req.Command {
				t.Errorf("Command field lost: want %q, got %q", tc.req.Command, got.Command)
			}
			if got.AgentID != tc.req.AgentID {
				t.Errorf("AgentID field lost: want %q, got %q", tc.req.AgentID, got.AgentID)
			}
			if got.SessionID != tc.req.SessionID {
				t.Errorf("SessionID field lost: want %q, got %q", tc.req.SessionID, got.SessionID)
			}
		})
	}
}
