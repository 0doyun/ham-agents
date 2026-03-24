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
- Platform: macOS
- IPC direction: Unix domain socket + JSON event stream
- Initial delivery strategy: managed mode first, then runtime/persistence, then menu bar, then richer orchestration

## Repository Layout

```text
Apps/
  HamMenuBarApp/          # Swift macOS app planning surface
go/
  cmd/ham/                # Go CLI entrypoint
  cmd/hamd/               # Go daemon entrypoint
  internal/...            # runtime, store, inference, adapters, ipc
Sources/
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
swift build
swift test
```

The target architecture will add:

```bash
go test ./...
```
