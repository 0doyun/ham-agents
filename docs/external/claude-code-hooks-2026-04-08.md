# Claude Code Hooks Documentation (fetched 2026-04-08)

Source: https://code.claude.com/docs/en/hooks
(canonical redirect from https://docs.anthropic.com/en/docs/claude-code/hooks → 301 to https://code.claude.com/docs/en/hooks)
Retrieved at: 2026-04-08 00:00 KST
Purpose: preserved evidence for Phase 2 PTY spike (dev/phase2-spike) permission interception verdict

---

## PermissionRequest Hook — Behavior

**PermissionRequest** hooks run when a permission dialog is about to be shown to the user. Unlike PreToolUse hooks that fire before tool execution regardless of permission status, PermissionRequest hooks specifically trigger at the moment Claude Code needs user confirmation.

These hooks allow you to:
- Automatically allow or deny permission requests on behalf of the user
- Modify tool inputs before execution
- Apply permission rules so the user isn't prompted again in the future

Matches on tool name (same values as PreToolUse): `Bash`, `Edit`, `Write`, `Read`, `Glob`, `Grep`, `Agent`, `WebFetch`, `WebSearch`, `AskUserQuestion`, `ExitPlanMode`, and any MCP tool names.

## hookSpecificOutput JSON Decision Schema

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow",
      "updatedInput": {
        "command": "npm run lint"
      },
      "updatedPermissions": [
        {
          "type": "addRules",
          "rules": [{ "toolName": "Bash", "ruleContent": "npm run *" }],
          "behavior": "allow",
          "destination": "localSettings"
        }
      ],
      "message": "Permission denied because..."
    }
  }
}
```

### Field Descriptions

| Field                | Required | Values                             | Description                                                                                                               |
|:---------------------|:---------|:-----------------------------------|:--------------------------------------------------------------------------------------------------------------------------|
| `behavior`           | yes      | `"allow"` or `"deny"`             | Grants or denies the permission                                                                                           |
| `updatedInput`       | no       | object                             | For `"allow"` only: modifies the tool's input parameters before execution. Replaces the entire input object              |
| `updatedPermissions` | no       | array of permission update entries | For `"allow"` only: array of permission update entries to apply, such as adding allow rules                              |
| `message`            | no       | string                             | For `"deny"` only: tells Claude why the permission was denied                                                             |
| `interrupt`          | no       | boolean                            | For `"deny"` only: if `true`, stops Claude                                                                                |

## Exit Code Table

| Exit Code | Behavior                                                                                                     |
|:----------|:-------------------------------------------------------------------------------------------------------------|
| 0         | Success. Claude Code parses stdout for JSON output fields. JSON output is only processed on exit 0           |
| 2         | Blocking error. Denies the permission. stderr text is fed back to Claude as an error message                 |
| Other     | Non-blocking error. stderr is shown in verbose mode (`Ctrl+O`) and execution continues                       |

## Timeout Values

- **Command hooks**: **600 seconds** (default)
- **Agent hooks**: 60 seconds (default)
- **HTTP hooks**: 30 seconds (default)
- Override with the `timeout` field in your hook configuration (in seconds)

**Key claim supporting Phase 2 verdict**: PermissionRequest is a synchronous blocking event. Claude Code waits for the hook handler process to exit before resolving the dialog. The 600s default timeout for command hooks means the process can block for up to 10 minutes without Claude Code timing out.

## Example: Auto-approve Bash commands matching a pattern

```bash
#!/bin/bash
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')

if [[ "$COMMAND" =~ ^npm\ run ]]; then
  jq -n '{
    hookSpecificOutput: {
      hookEventName: "PermissionRequest",
      decision: {
        behavior: "allow",
        updatedInput: {
          command: "'"$COMMAND"' --safe-mode"
        }
      }
    }
  }'
else
  exit 0  # allow without modification
fi
```
