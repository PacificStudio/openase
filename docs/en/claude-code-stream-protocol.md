# Claude Code Stream Protocol In OpenASE

This document describes the Claude Code `stream-json` protocol as OpenASE understands it today.

It separates three concerns on purpose:

1. Claude CLI transport events actually read by OpenASE
2. Claude message block schemas embedded inside `assistant` and `user` messages
3. OpenASE-derived semantics such as "this tool result is a command output"

The implementation lives in:

- [`internal/infra/adapter/claudecode/adapter.go`](../internal/infra/adapter/claudecode/adapter.go)
- [`internal/provider/claudecode.go`](../internal/provider/claudecode.go)
- [`internal/orchestrator/claude_protocol.go`](../internal/orchestrator/claude_protocol.go)
- [`internal/orchestrator/agent_adapter_claudecode.go`](../internal/orchestrator/agent_adapter_claudecode.go)

## Sources

OpenASE uses two evidence sources:

- Claude Code reference bundle:
  - [`references/claude-code-source/claude-code-2.1.88/cli.js.map`](../references/claude-code-source/claude-code-2.1.88/cli.js.map)
  - [`references/claude-code-source/claude-code-2.1.88/sdk-tools.d.ts`](../references/claude-code-source/claude-code-2.1.88/sdk-tools.d.ts)
- Observed local Claude Code runs and adapter fixtures in:
  - [`internal/orchestrator/agent_adapter_test.go`](../internal/orchestrator/agent_adapter_test.go)

When a field is not visible in the reference schemas but is present in local bridge output, this document calls it an "observed bridge extension".

## Transport Events

The Claude adapter reads line-delimited JSON from CLI `--output-format stream-json`.

Top-level event kinds currently parsed by OpenASE:

- `assistant`
- `user`
- `result`
- `rate_limit_event`
- `stream_event`
- `system`
- `task_started`
- `task_progress`
- `task_notification`

The event envelope fields OpenASE preserves are:

- `uuid`
  Meaning: event identity for this emitted frame.
- `session_id`
  Meaning: Claude session / thread identity.
- `parent_tool_use_id`
  Meaning: linkage to the tool-use context that produced a follow-up message.
  Important: this is a relation id, not the event id.
- `raw`
  Meaning: the original JSON frame stored for trace/debug purposes.

## Message Blocks

`assistant.message.content` and `user.message.content` are parsed into typed blocks.

Block kinds currently modeled explicitly:

- `text`
- `tool_use`
- `server_tool_use`
- `mcp_tool_use`
- `tool_result`

Important fields:

- `tool_use.id`
  Meaning: stable tool-call identity created by Claude.
- `tool_result.tool_use_id`
  Meaning: the tool call this result answers.
- `tool_use.name`
  Meaning: the concrete tool name Claude invoked.
- `tool_use.input`
  Meaning: structured tool arguments.

OpenASE preserves these separately from its own derived semantics. It does not rename them into OpenASE concepts at the parse layer.

## Task And Session Events

The reference bundle exposes SDK-side schemas for the following logical events:

- `system / task_started`
- `system / task_progress`
- `system / task_notification`
- `system / session_state_changed`

The Claude CLI bridge consumed by OpenASE flattens some of them into top-level transport kinds such as:

- `task_started`
- `task_progress`
- `task_notification`

Reference-backed task/session fields:

- `task_id`
- `tool_use_id`
- `description`
- `task_type`
- `workflow_name`
- `prompt`
- `usage.total_tokens`
- `usage.tool_uses`
- `usage.duration_ms`
- `last_tool_name`
- `summary`
- `status` on task notifications
- `output_file`
- `state` on session state changes

Observed bridge extensions seen in local runs/tests and preserved by OpenASE:

- `turn_id`
- `stream`
- `command`
- `text`
- `snapshot`
- legacy top-level `message`
- legacy top-level `status`

These observed bridge extensions are intentionally kept explicit in [`claude_protocol.go`](../internal/orchestrator/claude_protocol.go), so readers can tell they are not reference-backed schema guarantees.

## OpenASE-Derived Semantics

OpenASE derives a few higher-level runtime events from Claude protocol data.

These are not native Claude protocol fields:

- `ToolCallRequested`
  Derived from `tool_use` / `server_tool_use` / `mcp_tool_use` blocks.
- `command` output stream
  Derived only when the tool name is in an explicit allowlist of command-capable tools and the tool input contains `cmd` or `command`.
- `turn_diff_updated`
  Derived when a tool result text looks like a unified diff.

The command-tool allowlist is intentionally explicit, not fuzzy:

- `functions.exec_command`
- `exec_command`
- `bash`

This keeps OpenASE from silently reclassifying arbitrary future tools as command tools just because their names contain words like `shell` or `terminal`.

## Item Identity Rules

OpenASE distinguishes:

- event identity: `uuid`
- tool linkage: `parent_tool_use_id`
- tool call identity: `tool_use.id` / `tool_result.tool_use_id`

When Claude does not provide a usable event id for snapshot grouping, OpenASE synthesizes one. That synthesis is a fallback for UI stability, not part of the Claude protocol contract.

## Non-Goals

OpenASE currently does not claim that every Claude stream field is fully modeled.

Known gaps:

- Claude-specific block variants beyond the typed set above may still be preserved only in raw payloads.
- Diff extraction remains a derived heuristic based on unified diff text, not a protocol-native event.
- Some bridge-only fields may evolve upstream; when they do, update this document and the typed parsers together.
