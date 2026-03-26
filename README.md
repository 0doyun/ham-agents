# ham-agents

**Terminal AI sessions as a hamster team.**

ham-agents is a macOS menu bar app that lets you manage multiple AI coding agents (Claude Code, Codex, etc.) from one place. Run them, watch pixel hamsters work at their desks, get notified when they need you, and send messages — all without leaving your workflow.

<p align="center">
  <img src="assets/image.png" alt="ham-agents pixel office" width="420">
</p>

## What it does

- `ham run claude` — starts Claude Code and registers a hamster in the office
- `ham run codex` — same for Codex (or any CLI-based AI agent)
- Menu bar shows a pixel office where each hamster represents a running agent
- Hamsters move between zones based on agent state (desk = working, kitchen = idle, alert = needs input)
- Click a hamster to see what it's doing, send it a message, or jump to its terminal
- macOS notifications when an agent finishes, errors, or needs input
- All state tracked locally — nothing leaves your machine

## Quick start

### Requirements

- macOS 13+
- Go 1.21+
- Swift 5.10+
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

### Run

```bash
ham run claude
```

That's it. The daemon starts automatically via launchd, the menu bar app launches, and your hamster appears at its desk.

## CLI

```
ham run <provider>              # start an agent (claude, codex, etc.)
ham list                        # list all tracked agents
ham status                      # summary with attention counts
ham ask <agent> "message"       # send a message to an agent
ham stop <agent>                # stop a managed agent
ham attach --pick-iterm-session # attach to an existing iTerm session
ham doctor                      # check daemon, socket, launchd status
ham ui                          # launch the menu bar app manually
```

## Architecture

```
ham (Go CLI) ──── IPC ────► hamd (Go daemon)
     │                          │
     │ PTY                      │ state inference
     ▼                          ▼
  provider               agent registry
  (claude, codex)        event log
                         settings
                              │
                              ▼
                    ham-menubar (Swift)
                    pixel office UI
                    notifications
```

- **ham** — CLI that wraps providers in a PTY, forwarding output to the daemon for state inference
- **hamd** — background daemon managed by launchd, tracks agent state, serves IPC
- **ham-menubar** — SwiftUI menu bar app with pixel hamster office, notifications, quick actions

## Menu bar app

The popover shows a pixel office with four zones:

| Zone | Meaning |
|------|---------|
| Desk | Agent is actively working (thinking, typing, running tools) |
| Library | Agent is reading files or code |
| Kitchen | Agent is idle or done |
| Alert | Agent has an error or needs your input |

From the popover you can:
- See each agent's current status and last output
- Send a quick message directly to the agent's terminal session
- Open the agent's iTerm tab
- Pause/resume notifications per agent
- Edit agent roles
- Stop tracking an agent

## Development

```bash
# Run tests
go test ./...
swift test --disable-sandbox

# Run from source (without installing)
go run ./go/cmd/ham run claude
```

## Status

Early alpha. Core flow works end-to-end:

- [x] `ham run claude` / `ham run codex` with interactive PTY
- [x] Automatic daemon bootstrap via launchd
- [x] Automatic menu bar app launch
- [x] Real-time state inference from provider output
- [x] Pixel hamster office with status-based zone placement
- [x] iTerm2 session targeting (open tab, send message)
- [x] macOS notifications (done, error, waiting input)
- [x] Agent lifecycle (register, track, stop, remove)

Not yet done:
- [ ] Homebrew formula
- [ ] Team/workspace grouping
- [ ] Multiple hamster skins and office customization in the wild
- [ ] Broader provider support beyond Claude Code and Codex

## License

TBD
