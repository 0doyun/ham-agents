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

- Language: Swift
- Build system: Swift Package Manager
- Platform: macOS
- UI direction: SwiftUI/AppKit menu bar app
- Initial delivery strategy: managed mode first, then runtime, then menu bar, then richer orchestration

## Repository Layout

```text
Apps/
  HamMenuBarApp/          # macOS app target planning surface
Sources/
  HamCLI/                 # CLI entrypoint
  HamCore/                # shared domain models
  HamRuntime/             # session/runtime coordination
  HamPersistence/         # local persistence abstractions
  HamNotifications/       # notification layer
  HamInference/           # status inference engine
  HamAdapters/            # external integrations such as iTerm2
Tests/
  HamCoreTests/
  HamRuntimeTests/
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
