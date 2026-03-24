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
  HamAppServices/         # Swift daemon client + menu bar summary prep
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
go run ./go/cmd/ham status --json
go run ./go/cmd/ham events --json --limit 5
```

Swift menu bar prep currently lives in `HamAppServices`, which provides:
- daemon request/response payload models shared with Go
- a Unix socket transport for `hamd`
- a summary service that can turn snapshot + events into menu bar counts/feed data
