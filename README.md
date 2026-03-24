# ham-agents

`ham-agents` is a local macOS orchestrator for terminal-based AI sessions.

The long-term product goal is to implement the full experience described in [`spec.md`](./spec.md): managed terminal agents, a local runtime, a menu bar app, notifications, iTerm2 integration, and the pixel-office UI.

This repository is structured so autonomous execution workflows such as `ralph` can keep moving in small, verifiable slices without losing the product direction.

## Source of Truth

- Product truth: `spec.md`
- Future direction reference: `roadmap.md`
- Active backlog and current slice: `tasks.md`
- Current implementation architecture: `docs/architecture.md`
- Working assumptions: `docs/assumptions.md`
- Execution history: `docs/progress.md`
- Ralph planning artifacts: `.omx/plans/`

## Current Technical Direction

- UI: SwiftUI/AppKit menu bar app
- CLI/runtime: Go
- Transitional bootstrap: SwiftPM package kept green during migration
- Platform: macOS
- IPC direction: Unix domain socket + JSON event stream
- Initial delivery strategy: managed mode first, then runtime/persistence, then menu bar, then richer orchestration

## Repository Layout

```text
apps/
  macos/HamMenuBarApp/    # Swift macOS app planning surface
go/
  cmd/ham/                # Go CLI entrypoint
  cmd/hamd/               # Go daemon entrypoint
  internal/core/          # agent domain model and runtime snapshot
  internal/runtime/       # managed registry service
  internal/store/         # local file-backed persistence
  internal/ipc/           # socket path/bootstrap IPC config
  internal/adapters/      # iTerm2 adapter boundary
Sources/
  HamCore/                # shared Swift models + daemon payload contracts
  HamAppServices/         # Swift daemon client + menu bar summary/view-model prep
  ...                     # transitional Swift bootstrap code
Tests/
  ...                     # transitional Swift bootstrap tests
docs/
  architecture.md
  assumptions.md
  progress.md
.omx/plans/
  prd-ham-agents.md
  test-spec-ham-agents.md
```

## Current Verification

The bootstrap slice should remain green with:

```bash
swift build --disable-sandbox
swift test --disable-sandbox
```

The Go bootstrap slice adds:

```bash
go test ./...
```

Daemon-backed smoke verification currently requires running `hamd serve --once=false` outside the Codex sandbox because Unix socket binding is blocked inside the sandboxed test environment.

Current daemon-backed CLI surface:

```bash
go run ./go/cmd/ham list
go run ./go/cmd/ham run claude reviewer --project /tmp/demo --role reviewer
go run ./go/cmd/ham attach iterm2://session/abc ops --project /tmp/demo --role reviewer
go run ./go/cmd/ham open --print <agent-id>
go run ./go/cmd/ham status --json
go run ./go/cmd/ham events --json --limit 5
```

Swift menu bar prep currently lives in `HamAppServices`, which provides:
- daemon request/response payload models shared with Go
- a Unix socket transport for `hamd`
- a summary service that can turn snapshot + events into menu bar counts/feed data
- a menu bar view model used by the baseline `ham-menubar` executable target

The current `ham-menubar` baseline:
- starts an initial refresh on launch
- polls daemon state on an interval through the shared view model
- supports manual refresh from the popover
- detects done / waiting_input / error status transitions and routes notification candidates through a sink boundary
- requests notification permission on first delivery attempt and can submit macOS notification requests through `UserNotifications`
- shows a selected-agent detail pane with recent event context inside the popover
- includes a baseline “Open Project Folder” action from the selected-agent detail pane
- shows current notification permission state and lets the user request permission from the popover
- uses `sessionRef` as a URL-based session target when available, otherwise falls back to opening the workspace in iTerm or Finder
- lets the user pause/resume notifications per selected agent inside the popover
- includes a quick-message field that prefers iTerm write automation and falls back to clipboard/session handoff
- shows quick-message result feedback so fallback/error behavior is visible instead of silent
- persists notification pause/resume through the daemon so mute state survives refreshes instead of living only in Swift UI memory
- lets the user edit and save the selected agent role through the daemon-backed detail pane
- lets the user stop tracking the selected agent from the popover
- lets the user edit notification toggles from the popover and have them affect actual delivery behavior
- already renders attached agents distinctly because mode/confidence now flow through from the daemon
- shows mode and confidence inline so managed/attached distinctions are visible in the baseline UI
- can also register observed sources and refresh their status heuristically from transcript/log contents
- keeps observed-source status fresher while `hamd` is serving by polling observed paths in the background
- resolves open targets for agents through the daemon and exposes that path via `ham open`
- reads and updates a minimal notification settings document from the popover
- enforces notification toggles and preview-text masking from that backend settings document
