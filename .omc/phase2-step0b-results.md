# Phase 2 Step 0b — Permission Interception Spike Results

**Branch:** `dev/phase2-spike`
**Date:** 2026-04-07
**Question:** When `hamd` receives a `hook.permission-request` and delays its response, does the actual tool call inside Claude Code block and wait, or proceed anyway?

---

## TL;DR — Verdict: 2 layers separated

**Verdict — 2 layers separated:**
- **Transport layer: BLOCKS** — hamd IPC request/response cycle is synchronous. Python proxy experiment confirmed server-side delay propagates 1:1 to client (0/500/1000/5000ms matrix, <10ms overhead).
- **Application layer: Fire-and-forget today** — hamd currently returns empty `Response{}` without a PermissionDecision field; no wait primitive in CommandHookPermissionReq handler; runHook emits empty stdout. P2-3 implementation must add the decision payload and wait primitive on top of the existing sync transport.

**P2-3 verdict**: proceed with design as written in ham-studio.md. Required implementation: 4 additions (see below).

---

## Phase 1 — Static analysis

### 1. Official Claude Code hook contract

Source: `docs/external/claude-code-hooks-2026-04-08.md` (archived copy; fetched 2026-04-08 from https://code.claude.com/docs/en/hooks, which is the canonical redirect destination of https://docs.anthropic.com/en/docs/claude-code/hooks)

- **`PermissionRequest`** is a documented, dedicated hook event that "Runs when the user is shown a permission dialog. Use PermissionRequest decision control to allow or deny on behalf of the user."
- Hook handler reads JSON from **stdin**, writes a decision to **stdout**, and Claude Code waits for the process to **exit** before resolving the dialog.
- Decision schema:
  ```json
  {"hookSpecificOutput": {"hookEventName": "PermissionRequest",
     "decision": {"behavior": "allow"|"deny",
                  "updatedInput": {...},
                  "message": "...",
                  "interrupt": true}}}
  ```
- Per the "Exit code 2 behavior per event" table: **PermissionRequest = Yes (can block) → "Denies the permission"**.
- Default timeout for command hooks: **600 seconds**. So Claude Code is willing to wait up to 10 minutes for a permission decision.

**Conclusion:** Claude Code blocks on hook handler process exit. This is the spec, not an implementation detail.

### 2. hamd IPC server — `go/internal/ipc/server.go:478-488`

```go
case CommandHookPermissionReq:
    if err := s.prepareHookRequest(ctx, &request); err != nil {
        if errors.Is(err, errNoAgent) { return Response{}, nil }
        return Response{}, err
    }
    if err := s.registry.RecordHookPermissionRequest(ctx, request.AgentID, request.ToolName, request.OmcMode); err != nil {
        return Response{}, err
    }
    return Response{}, nil
```

The handler records the event and returns `Response{}` (empty). No decision field, no wait primitive, no condition variable, no channel for an external "approve/deny" signal. **This is currently fire-and-forget at the application level.**

### 3. Registry mutator — `go/internal/runtime/managed_state.go:632-655`

`RecordHookPermissionRequest` flips `agent.Status = AgentStatusWaitingInput`, sets `StatusReason = "Approve <tool>?"`, emits an `EventTypeAgentStatusUpdated` event, and returns. **No blocking primitive of any kind.** It's a state-write only.

### 4. ham CLI hook subcommand — `go/cmd/ham/commands.go:386-388`

```go
case "permission-request":
    toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
    hookErr = client.HookPermissionRequest(ctx, agentID, payload.SessionID, sessionRef, toolName, detectOmcMode())
```

`runHook` synchronously calls `client.HookPermissionRequest` (`go/internal/ipc/ipc.go:513-516`) which uses `c.request(ctx, ...)` — a synchronous request/response over the unix socket. **The CLI does block on hamd's response.** When hamd returns, runHook prints nothing useful to stdout for Claude Code (no `hookSpecificOutput` JSON), and exits 0.

### 5. The contract chain summary

```
Claude Code spawns: $ ham hook permission-request   ← spawns subprocess, blocks on Wait()
        ham CLI ─→ unix socket request  ─→ hamd     ← blocks on conn.Read()
                                              ↓
                                       record event, return {}
        ham CLI ─← unix socket response ─← hamd
        ham CLI exits 0 with empty stdout
Claude Code reads stdout (empty) → no decision → falls through to native dialog
```

**Static verdict:** the wait *path* exists and is synchronous. The decision *payload* does not exist yet — neither hamd nor the ham CLI emits any `hookSpecificOutput` JSON. P2-3 needs to add: (a) a wait primitive in hamd's handler, (b) a decision payload in `Response`, (c) JSON emission in `runHook`'s permission-request branch.

---

## Phase 2 — Dynamic experiment

**Method:** Built `/tmp/hamd-spike` from current `dev/phase2-spike` HEAD. Started in isolated `HAM_AGENTS_HOME=/tmp/ham-spike-1`. Probed the IPC socket with a Python client. To simulate server-side delay without modifying hamd source, ran a Python proxy on a second unix socket that forwards client→hamd, waits N ms after receiving hamd's reply, then forwards back to the client. Measured client-perceived round-trip time.

### Experiment A — Direct IPC probe (no delay)

Registered a managed agent, then issued 5 sequential `hook.permission-request` calls:

```
register.managed              dt=0.90ms  resp={"agent": {"id": "managed-..."}}
hook.permission-request #0    dt=0.42ms  resp={}
hook.permission-request #1    dt=0.30ms  resp={}
hook.permission-request #2    dt=0.29ms  resp={}
hook.permission-request #3    dt=0.27ms  resp={}
hook.permission-request #4    dt=0.27ms  resp={}
```

Confirms: hamd accepts the command, returns an **empty** body, completes in <0.5ms. No application-level waiting.

### Experiment B — Injected server-side delay via proxy

| Server delay | Client wait? | Client observed | Tool result |
|--------------|--------------|-----------------|-------------|
| 0 ms         | yes (sync)   | 0.6 ms          | response received, body `{}` |
| 500 ms       | yes (sync)   | 502.9 ms        | response received, body `{}` |
| 1000 ms      | yes (sync)   | 1002.0 ms       | response received, body `{}` |
| 5000 ms      | yes (sync)   | 5009.9 ms       | response received, body `{}` |

Latency overhead: <10 ms across all rows. The client (acting as `ham hook permission-request` would) blocks 1:1 with server-side delay. Since `ham hook` CLI uses the same synchronous `c.request` primitive, and Claude Code blocks on `ham hook` process exit, the same latency would propagate end-to-end to the Claude Code permission dialog.

### Subprocess Wall-Clock Measurement

**Method**: Built `/tmp/ham-spike` (from `go/cmd/ham`) and `/tmp/hamd-spike` (from `go/cmd/hamd`) at dev/phase2-spike HEAD. Started hamd in isolated `HAM_AGENTS_HOME=/tmp/ham-spike-run3`. Registered a managed agent via direct IPC (newline-delimited JSON socket). Timed three sequential `ham hook permission-request` subprocess invocations using the `time` builtin with `HAM_AGENT_ID` set (the env var the CLI reads for agent identity).

**Command**:
```
time HAM_AGENT_ID=<agent-id> HAM_AGENTS_HOME=/tmp/ham-spike-run3 /tmp/ham-spike hook permission-request Bash < /dev/null
```

**Results**:

| Run | Wall-clock time | Exit code | Stdout |
|-----|----------------|-----------|--------|
| 1   | 11 ms          | 0         | (empty) |
| 2   | 9 ms           | 0         | (empty) |
| 3   | 11 ms          | 0         | (empty) |

Mean: ~10ms. Consistent across runs.

**Interpretation**: The ~10ms includes Go binary startup (~7ms) + unix socket connect/request/response roundtrip (<1ms, consistent with Experiment A). The subprocess path is synchronous end-to-end: `os.exec.Wait()` in Claude Code blocks until the ham process exits, which only happens after the IPC response returns from hamd. No goroutine detachment, no background work — `c.request()` is a single synchronous `json.NewEncoder/Decoder` exchange on a blocking unix socket connection.

**Cleanup**: hamd PID killed, `/tmp/ham-spike*` directories and binaries removed, `git diff go/` clean.

---

## Verdict: **BLOCKS**

The full chain — Claude Code → `ham hook` subprocess → `ham` IPC client → `hamd` socket handler — is end-to-end synchronous. Any delay introduced inside `hamd`'s `CommandHookPermissionReq` handler will hold up the Claude Code permission dialog, up to the documented 600 s default hook timeout.

---

## Implication for P2-3 (Approval Interception)

**P2-3 Approval Interception can proceed as designed in `docs/spec/ham-studio.md`** — the transport supports synchronous request/response with arbitrary latency.

The design must include four concrete additions, none of which exist today:

1. **`Response` struct gains a permission-decision field** (`go/internal/ipc/ipc.go:118`) — e.g.
   ```go
   PermissionDecision *PermissionDecision `json:"permission_decision,omitempty"`
   ```
   carrying `behavior` (`allow`/`deny`), `reason`, and optional `updated_input`.

2. **New IPC command `decision.permission`** (Studio → hamd) with `{agent_id, request_id, behavior, message}` payload — allows the ham-studio UI to push a user's approval/denial choice back into the waiting handler. Implies tracking pending permission requests by `(agent_id, request_id)` in `managed_state.go`, with cleanup on context cancellation.

3. **`CommandHookPermissionReq` handler gains a wait primitive** (`go/internal/ipc/server.go:478`) — record the event, then `select` on either an external decision channel (filled by the `decision.permission` command above) or a context-deadline. Return the decision in the response body. Default to `behavior=ask` (no decision) on timeout so Claude Code falls through to the native dialog.

4. **`runHook` permission-request branch emits JSON to stdout** (`go/cmd/ham/commands.go:386-388`) — when hamd returns a non-nil `PermissionDecision`, marshal it to the Claude Code wire format:
   ```json
   {"hookSpecificOutput": {"hookEventName": "PermissionRequest",
      "decision": {"behavior": "deny", "message": "<reason>"}}}
   ```
   and write to `os.Stdout` before exiting 0.

**Inconclusive area:** this spike cannot prove that Claude Code handles a >60s hook latency without showing a fallback UI. Spec says 600 s timeout, but real-world UX may vary. Recommend a follow-up manual smoke test with a real Claude Code session before P2-3 ships.

---

## Cleanup

- Background hamd PID 41109 sent SIGINT, confirmed dead via `ps -p`.
- `/tmp/ham-spike-1/`, `/tmp/hamd-spike`, `/tmp/hooks-doc.txt` removed.
- `git diff go/` is empty — no hamd source modified.
- No files in `~/.claude/`, `~/.ham`, or any user state directory touched.
