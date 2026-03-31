<p align="center">
  <img src="assets/image.png" alt="ham-agents pixel office" width="420">
</p>

<h1 align="center">ham-agents</h1>

<p align="center">
  <strong>Your AI agents, visualized as a hamster team.</strong>
  <br>
  A macOS menu bar app that turns Claude Code sessions into a pixel office.
  <br><br>
  <a href="#install">Install</a> &middot;
  <a href="#how-it-works">How it works</a> &middot;
  <a href="#cli">CLI</a> &middot;
  <a href="#features">Features</a> &middot;
  <a href="#development">Development</a>
</p>

---

## Install

```bash
brew tap 0doyun/ham
brew install ham
ham setup
```

That's it. Next time you start Claude Code, the hamster office appears in your menu bar.

> **Requirements:** macOS 13+ (Apple Silicon) · [Claude Code](https://docs.anthropic.com/en/docs/claude-code)

## How it works

1. **`ham setup`** registers hooks in Claude Code so every tool use, notification, and session event is tracked
2. **`hamd`** (daemon) starts automatically and maintains agent state locally
3. **`ham-menubar`** launches on session start — a pixel office where each hamster is one of your agents

Each hamster sits at their own desk. What's on the desk tells you what they're doing:

| Status | Desk | Indicator |
|---|---|---|
| Thinking / Running tool | iMac + coffee mug | Yellow glow |
| Reading files | Book stack | — |
| Waiting for input | Orange glow monitor | ❓ above hamster |
| Error / Disconnected | Red glow monitor | Red dot |
| Idle / Sleeping | Closed laptop | Zzz |

Click any hamster to see details, send a message, or jump to its terminal.

## Features

### Agent Teams

<p align="center">
  <img src="assets/image-2.png" alt="ham-agents team mode" width="520">
</p>

When you use Claude Agent Teams, ham-agents shows it:

- **Team lead** gets a crown badge
- **Task progress** (e.g. `0/3`) displayed per agent
- **Sub-agents** appear as mini hamsters surrounding their parent

### Notifications

- macOS notifications when an agent errors or needs input
- Configurable quiet hours, per-agent mute, heartbeat pings
- Notification preview text in the menu bar

### Multi-session management

- Run multiple Claude Code sessions in parallel
- Each gets its own hamster and workstation
- Grid auto-expands: 1–3 agents → 1 row, 4–6 → 2 rows, 7–9 → 3 rows
- Attach to existing iTerm2 tabs or tmux panes

### Everything local

All state is stored in `~/Library/Application Support/ham-agents/`. Nothing leaves your machine.

## CLI

```
ham run <provider>              # start an agent
ham setup                       # configure Claude Code hooks
ham list                        # list all tracked agents
ham status                      # summary with attention counts
ham ask <agent> "message"       # send a message to an agent
ham stop <agent>                # stop a managed agent
ham attach --pick-iterm-session # attach to an existing iTerm session
ham doctor                      # check daemon, hooks, socket status
ham ui                          # launch the menu bar app manually
ham team create <name>          # create a team
ham team add <team> <agent>     # add agent to team
```

## Architecture

```
ham (CLI) ──── IPC ────► hamd (daemon)
  │                          │
  │ hooks                    │ state tracking
  ▼                          ▼
Claude Code              agent registry
                         event log
                         settings
                              │
                              ▼
                    ham-menubar (Swift)
                    pixel office · notifications
```

| Component | Language | Role |
|---|---|---|
| `ham` | Go | CLI — wraps providers, forwards hook events |
| `hamd` | Go | Daemon — agent state, IPC server, launchd managed |
| `ham-menubar` | Swift | Menu bar UI — pixel office, notifications, quick actions |

## Development

```bash
git clone https://github.com/0doyun/ham-agents.git
cd ham-agents

# Build from source
go build -o ~/go/bin/ham ./go/cmd/ham
go build -o ~/go/bin/hamd ./go/cmd/hamd
swift build --disable-sandbox

# Run tests
go test ./...
swift test --disable-sandbox
```

## License

MIT
