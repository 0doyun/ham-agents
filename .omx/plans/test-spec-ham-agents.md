# Test Spec: ham-agents

## Verification Principles

- Every slice must leave the repository buildable.
- Changed logic should gain tests before or with the implementation.
- CLI/runtime/core changes need automated verification first.
- Menu bar UI later adds targeted manual verification where automation is thin.

## Current Baseline Checks

```bash
swift build
swift test
```

Hybrid transition adds:

```bash
go test ./...
```

## Planned Verification Matrix

### Repository Bootstrap
- `swift build`
- `swift test`
- Source layout matches architecture docs

### Hybrid Architecture Baseline
- Documentation matches `Swift UI + Go CLI/runtime`
- Go workspace bootstrap exists
- Swift UI bootstrap remains buildable until migration completes

### Managed Session Foundation
- Unit tests for agent model defaults and status enums
- Runtime tests for register/list/status behavior
- CLI smoke checks for `ham list` and `ham status`

### Runtime and Persistence
- Lifecycle transition tests
- Persistence round-trip tests
- Snapshot generation tests

### Notifications
- Unit tests for trigger selection
- Dedupe policy tests
- Manual notification smoke test on macOS when integrated

### iTerm2 Integration
- Adapter contract tests where possible
- Manual focus/open verification
- Graceful fallback verification when integration is unavailable

### Attached / Observed Modes
- Confidence/reason calculation tests
- Mode rendering tests in CLI/UI formatting layer

### Pixel Office
- State-to-animation mapping tests where practical
- Manual visual verification for layout and interaction

## Exit Criteria For Ralph Slice Completion

- Active slice checkbox state updated in `tasks.md`
- Relevant docs updated
- `swift build` passes
- `swift test` passes
- Diagnostics on changed files are clean
- Git commit happens only when repository is a real worktree
