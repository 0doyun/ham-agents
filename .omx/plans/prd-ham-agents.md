# PRD: ham-agents

## Task Statement
Implement `ham-agents` toward the full product spec in `spec.md` while preserving a buildable, testable repository at every slice.

## Desired Outcome

- A working macOS-oriented product consisting of:
  - `ham` CLI
  - local runtime and persistence
  - menu bar application
  - notifications
  - iTerm2 integration
  - managed, attached, and observed agent tracking
  - pixel-office visualization

## Product Promise

`ham-agents` should let a user feel like they are operating a visible team of terminal AI agents instead of manually checking scattered terminal sessions.

## Scope Rules

- `spec.md` is the product truth.
- Work proceeds in vertical slices.
- The repository must stay green after each slice.
- No speculative dependencies without an active need.
- Record assumptions and progress continuously.

## Current Execution Phases

1. Repository bootstrap and architectural grounding
2. Hybrid architecture realignment (`Swift UI + Go runtime`)
3. Managed session foundation
4. Runtime and persistence
5. Menu bar baseline
6. Notifications
7. iTerm2 integration
8. Attached / observed modes
9. Inference refinement
10. Pixel office completion

## User Stories

### US-001 Repository Bootstrap
As the developer, I want a stable repository structure so autonomous execution can keep shipping slices without re-deciding architecture.

Acceptance criteria:
- Swift package exists
- Core modules are separated
- Tests run
- Ralph planning docs exist

### US-002 Hybrid Architecture Baseline
As the developer, I want the macOS UI and terminal runtime to be separated by responsibility so the implementation matches the product shape.

Acceptance criteria:
- Swift owns the menu bar app
- Go owns the CLI and daemon
- IPC direction is documented
- tasks and architecture docs match this split

### US-003 Managed Mode Foundation
As a user, I want the product to track managed agents first so the system can offer high-confidence status.

Acceptance criteria:
- `ham run` can register a managed agent
- `ham list` shows tracked agents
- `ham status` summarizes runtime state

### US-004 Local Runtime
As a user, I want agent state to survive beyond a single interaction so the app can behave like an orchestrator.

Acceptance criteria:
- Persistence boundary exists
- Runtime owns lifecycle transitions
- Event/snapshot model is testable

### US-005 Menu Bar Presence
As a user, I want a menu bar status surface so I can see the team at a glance.

Acceptance criteria:
- Menu bar app launches
- Current agent counts render
- Clicking opens a popover baseline

### US-006 Notifications
As a user, I want immediate alerts for done, error, and waiting-input states.

Acceptance criteria:
- Core notification triggers exist
- Noise control defaults are reasonable

### US-007 Terminal Integration
As a user, I want to reopen or focus tracked sessions from the product.

Acceptance criteria:
- iTerm2 adapter boundary exists
- Focus/open flow works for managed sessions

### US-008 Extended Tracking Modes
As a user, I want existing sessions and transcript-based sessions to appear in the same team model.

Acceptance criteria:
- attached and observed modes are supported
- mode and confidence are visible

### US-009 Pixel Office Experience
As a user, I want the team to feel alive without losing informational clarity.

Acceptance criteria:
- office zones map to state
- animation set is connected to runtime status
- detail interactions remain practical

## Risks

- Scope expands faster than verification capacity
- iTerm2 integration may require app-script or scripting bridge decisions later
- Pixel office may tempt premature UI work before runtime is stable
- Git automation cannot proceed until this folder is a real worktree
