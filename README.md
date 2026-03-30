# ham-agents

**Terminal AI sessions as a hamster team.**

ham-agents is a macOS menu bar app that manages your Claude Code sessions from one place. Run them, watch pixel hamsters work at their desks, get notified when they need you, and send messages — all without leaving your workflow.

<p align="center">
  <img src="assets/image.png" alt="ham-agents pixel office" width="420">
</p>

## What it does

- `ham run claude` — starts Claude Code and registers a hamster in the office
- `ham setup` — auto-configures Claude Code hooks for accurate state tracking
- Menu bar shows a pixel office where each hamster represents a running agent
- Hamsters sit at individual workstations with status-specific furniture (monitor, books, coffee)
- Sub-agents appear as mini hamsters surrounding their parent in an arc
- Click a hamster to see what it's doing, send it a message, or jump to its terminal
- macOS notifications when an agent errors or needs input
- All state tracked locally — nothing leaves your machine

## Quick start

### Requirements

- macOS 13+
- Go 1.21+
- Swift 5.10+
- Claude Code
- iTerm2 (recommended, for session targeting)

### Build and install

```bash
git clone https://github.com/0doyun/ham-agents.git
cd ham-agents

# Build CLI + daemon
go build -o ~/go/bin/ham ./go/cmd/ham
go build -o ~/go/bin/hamd ./go/cmd/hamd

# Build menu bar app
swift build --disable-sandbox
```

Make sure `~/go/bin` is in your PATH.

### Setup

```bash
ham setup
```

This detects Claude Code and adds hooks to `~/.claude/settings.json` for accurate state tracking. Your existing settings are preserved.

### Run

```bash
ham run claude
```

That's it. The daemon starts automatically via launchd, the menu bar app launches, and your hamster appears at its desk.

## CLI

```
ham run <provider>              # start an agent
ham setup                       # configure Claude Code hooks
ham list                        # list all tracked agents
ham status                      # summary with attention counts
ham ask <agent> "message"       # send a message to an agent
ham stop <agent>                # stop a managed agent
ham attach --pick-iterm-session # attach to an existing iTerm session
ham hook <event> [args]         # (internal) called by Claude Code hooks
ham doctor                      # check daemon, hooks, socket, launchd status
ham ui                          # launch the menu bar app manually
```

## Architecture

```
ham (Go CLI) ──── IPC ────► hamd (Go daemon)
     │                          │
     │ PTY + hooks              │ state tracking
     ▼                          ▼
  Claude Code              agent registry
  (hooks → ham hook)       event log
                           settings
                                │
                                ▼
                      ham-menubar (Swift)
                      pixel office UI
                      notifications
```

- **ham** — CLI that wraps Claude Code in a PTY, forwarding hook events to the daemon
- **hamd** — background daemon managed by launchd, tracks agent state, serves IPC
- **ham-menubar** — SwiftUI menu bar app with pixel hamster office, notifications, quick actions

### State tracking

With `ham setup`, Claude Code hooks fire on every tool use:

| Hook event | Agent status | Office visual |
|---|---|---|
| `PreToolUse Read/Grep/Glob` | reading | Book stack on desk |
| `PreToolUse Edit/Write/Bash` | running_tool | iMac monitor (green glow) |
| `PostToolUse` | thinking | iMac monitor |
| `PreToolUse Agent` | sub-agent spawned | Mini hamster appears |
| `PostToolUse Agent` | sub-agent finished | Mini hamster disappears |
| `Stop` | session end | Hamster removed |
| Process error exit | error | Red glow monitor + red dot |

Without hooks (fallback): PTY output keyword matching with lower confidence.

## Menu bar app

The popover shows a pixel office with a multi-row grid:

- Each hamster has their own workstation (desk + status furniture)
- Rows expand automatically: 1–3 agents → 1 row, 4–6 → 2 rows, 7–9 → 3 rows
- Sub-agents surround their parent in an arc formation
- Background: office wall with window, clock, whiteboard, poster

Status furniture (behind hamster, back-view perspective):

| Status | Furniture | Indicator |
|---|---|---|
| thinking / running_tool | iMac back + coffee mug | Yellow dot animation |
| reading | Book stack | — |
| error / disconnected | Red glow monitor | Red dot animation |
| waiting_input | Orange glow monitor | ❓ above hamster |
| idle / sleeping | Closed laptop | Zzz animation |

From the popover you can:
- Click a hamster to see its details
- Send a quick message directly to the agent's terminal
- Open the agent's iTerm tab or project folder
- Pause/resume notifications, edit role, stop tracking (via ⋯ menu)

## Development

```bash
# Run tests
go test ./...
swift test --disable-sandbox

# Run from source (without installing)
go run ./go/cmd/ham run claude
```

## Status

Alpha. Claude Code–first, with accurate hook-based state tracking.

- [x] `ham run claude` with interactive PTY
- [x] Claude Code hooks integration (`ham setup`)
- [x] Accurate state tracking via PreToolUse/PostToolUse/Stop hooks
- [x] Expanded official Claude Code hook coverage (Notification, StopFailure, SessionStart/End, SubagentStart/Stop)
- [x] Sub-agent detection and mini hamster visualization
- [x] Auto multi-row grid office with individual workstations
- [x] Automatic daemon bootstrap via launchd
- [x] Automatic menu bar app launch
- [x] iTerm2 session targeting (open tab, send message)
- [x] tmux session targeting (open pane, send message)
- [x] macOS notifications (error, waiting input)
- [x] OMC mode recognition (autopilot, ralph, team badges)
- [x] Autonomous mode heartbeat notifications
- [x] Agent lifecycle (register, track, stop, remove)
- [x] Detail panel with quick message, actions, recent events

Planned:

- [ ] Claude Agent Teams hook integration on top of the existing team model
- [ ] Worktree metadata MVP (capture + detail display before richer office grouping)
- [ ] Homebrew formula
- [ ] Multi-provider support (Codex, Gemini CLI)

## License

TBD
