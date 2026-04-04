# Gemini CLI Adaptation Guide For OpenASE

This guide describes how Gemini CLI should be integrated into OpenASE with a precise, protocol-first approach.

It follows the same design principles used for Claude Code integration:

1. Define explicit protocol types
2. Separate upstream fields from OpenASE-derived semantics
3. Document which layer each field comes from
4. Lock the behavior with protocol-level tests

## Scope

This guide is about Gemini CLI in headless mode, especially:

- `--output-format json`
- `--output-format stream-json`

Relevant upstream references:

- [`references/gemini-cli/docs/cli/headless.md`](../references/gemini-cli/docs/cli/headless.md)
- [`references/gemini-cli/packages/core/src/output/types.ts`](../references/gemini-cli/packages/core/src/output/types.ts)
- [`references/gemini-cli/packages/core/src/agent/types.ts`](../references/gemini-cli/packages/core/src/agent/types.ts)
- [`references/gemini-cli/packages/core/src/agent/event-translator.ts`](../references/gemini-cli/packages/core/src/agent/event-translator.ts)
- [`references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.ts`](../references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.ts)
- [`references/gemini-cli/docs/reference/tools.md`](../references/gemini-cli/docs/reference/tools.md)

## Current OpenASE State

OpenASE currently uses Gemini in a very thin one-shot mode:

- [`internal/chat/runtime_gemini.go`](../internal/chat/runtime_gemini.go)

Current behavior:

- invokes `gemini -p ... --output-format json`
- waits for the whole process to exit
- parses only final `response` and `stats`
- emits:
  - normalized assistant message
  - final `done`

What is missing today:

- no streaming deltas
- no `init`
- no tool call events
- no tool result events
- no warning/error event granularity
- no per-turn session metadata beyond OpenASE-local session bookkeeping
- no interruption / elicitation modeling

This is the main gap if we want Gemini to approach Claude/Codex parity.

## Upstream Layering

Gemini CLI has three distinct layers.

### 1. Internal model/runtime events

Internal Gemini agent logic emits `ServerGeminiStreamEvent` values such as:

- `ModelInfo`
- `Content`
- `Thought`
- `Citation`
- `ToolCallRequest`
- `ToolCallResponse`
- `Error`
- `Finished`
- `UserCancelled`
- `MaxSessionTurns`
- `AgentExecutionStopped`
- `AgentExecutionBlocked`
- `InvalidStream`

Reference:
- [`event-translator.ts`](../references/gemini-cli/packages/core/src/agent/event-translator.ts)

These are not the CLI transport contract. They are internal.

### 2. Internal agent protocol

Gemini translates internal events into `AgentEvent` values such as:

- `initialize`
- `session_update`
- `message`
- `agent_start`
- `agent_end`
- `tool_request`
- `tool_response`
- `tool_update`
- `usage`
- `error`
- `elicitation_request`
- `elicitation_response`
- `custom`

Reference:
- [`agent/types.ts`](../references/gemini-cli/packages/core/src/agent/types.ts)

Important detail:

- `tool_update`
- `usage`
- `session_update`
- `elicitation_request`

exist internally even if they do not all survive into the final headless `stream-json` transport.

### 3. Headless CLI transport contract

The final CLI `stream-json` contract is narrower and is what OpenASE should treat as the upstream wire protocol if it keeps using the Gemini CLI binary.

Reference contract:
- [`headless.md`](../references/gemini-cli/docs/cli/headless.md)
- [`output/types.ts`](../references/gemini-cli/packages/core/src/output/types.ts)

Event types:

- `init`
- `message`
- `tool_use`
- `tool_result`
- `error`
- `result`

This is the correct parse boundary for OpenASE if we integrate Gemini through the CLI process.

## Exact Headless Transport DTOs

The upstream `stream-json` DTOs are explicit.

### `init`

```json
{
  "type": "init",
  "timestamp": "ISO-8601",
  "session_id": "string",
  "model": "string"
}
```

Semantics:

- starts the headless stream
- identifies the Gemini session
- gives the selected model

### `message`

```json
{
  "type": "message",
  "timestamp": "ISO-8601",
  "role": "user | assistant",
  "content": "string",
  "delta": true
}
```

Notes:

- `delta` is optional
- in streaming mode, assistant text is emitted incrementally with `delta: true`
- user messages are also emitted once by the CLI wrapper

### `tool_use`

```json
{
  "type": "tool_use",
  "timestamp": "ISO-8601",
  "tool_name": "string",
  "tool_id": "string",
  "parameters": {}
}
```

Semantics:

- exact tool invocation request
- `tool_id` is the correlation key for the later `tool_result`

### `tool_result`

```json
{
  "type": "tool_result",
  "timestamp": "ISO-8601",
  "tool_id": "string",
  "status": "success | error",
  "output": "string",
  "error": {
    "type": "string",
    "message": "string"
  }
}
```

Important limitation:

- the headless transport only preserves a string `output`
- it does not preserve the richer internal `tool_response.displayContent`, `content`, and `data` structure
- if OpenASE needs Claude/Codex-level fidelity for diffs, structured file outputs, or media results, direct CLI `stream-json` is already a lossy boundary

### `error`

```json
{
  "type": "error",
  "timestamp": "ISO-8601",
  "severity": "warning | error",
  "message": "string"
}
```

Semantics:

- non-fatal warnings and surfaced runtime issues
- emitted for things like loop detection or non-fatal blocked execution

### `result`

```json
{
  "type": "result",
  "timestamp": "ISO-8601",
  "status": "success | error",
  "error": {
    "type": "string",
    "message": "string"
  },
  "stats": {
    "total_tokens": 0,
    "input_tokens": 0,
    "output_tokens": 0,
    "cached": 0,
    "input": 0,
    "duration_ms": 0,
    "tool_calls": 0,
    "models": {}
  }
}
```

Semantics:

- terminal event for the headless run
- carries aggregated usage
- `models` contains per-model breakdown

## What The CLI Drops

This matters a lot for design decisions.

The upstream non-interactive wrapper explicitly ignores several internal `AgentEvent` kinds:

- `initialize`
- `session_update`
- `agent_start`
- `tool_update`
- `elicitation_request`
- `elicitation_response`
- `usage`
- `custom`

Reference:
- [`nonInteractiveCliAgentSession.ts:591`](../references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.ts#L591)

So if OpenASE consumes only headless CLI `stream-json`, it cannot recover:

- internal thought/citation metadata beyond what becomes plain assistant text
- rich tool update progress
- explicit usage events before final result
- structured elicitation / approval requests
- custom events

That loss is upstream behavior, not an OpenASE parser bug.

## Tool Mapping Guidance

Gemini publishes exact built-in tool names in its tools reference:

- `run_shell_command`
- `glob`
- `grep_search`
- `list_directory`
- `read_file`
- `read_many_files`
- `replace`
- `write_file`
- `ask_user`
- `write_todos`
- `activate_skill`
- `get_internal_docs`
- `save_memory`
- `enter_plan_mode`
- `exit_plan_mode`
- `complete_task`
- `google_web_search`
- `web_fetch`

Reference:
- [`tools.md`](../references/gemini-cli/docs/reference/tools.md)

OpenASE should map these with exact allowlists, not fuzzy string matching.

Recommended derived categories:

- command tool:
  - `run_shell_command`
- file read tools:
  - `read_file`
  - `read_many_files`
  - `list_directory`
  - `glob`
  - `grep_search`
- file write tools:
  - `replace`
  - `write_file`
- ask-user / interrupt tools:
  - `ask_user`
- search/fetch tools:
  - `google_web_search`
  - `web_fetch`
- planning / bookkeeping tools:
  - `write_todos`
  - `enter_plan_mode`
  - `exit_plan_mode`
  - `complete_task`
  - `save_memory`
  - `activate_skill`
  - `get_internal_docs`

## Recommended OpenASE Parse Layer

If OpenASE keeps integrating Gemini via CLI, add a dedicated protocol file similar to Claude:

- `internal/chat/gemini_protocol.go`

Suggested explicit types:

- `geminiCLIInitEvent`
- `geminiCLIMessageEvent`
- `geminiCLIToolUseEvent`
- `geminiCLIToolResultEvent`
- `geminiCLIErrorEvent`
- `geminiCLIResultEvent`
- `geminiCLIStreamStats`
- `geminiCLIModelStats`

Design rules:

- Parse exact transport DTOs first
- Keep upstream field names recognizable
- Only after parsing, derive OpenASE semantics

Example derived semantics:

- `tool_use(tool_name=run_shell_command)` -> command tool request
- `tool_result(tool_id=...)` paired with previous `run_shell_command` -> command output block
- `tool_result` paired with `replace` or `write_file` -> file mutation result
- `tool_result.output` that looks like a unified diff -> derived diff event

That last item is still heuristic and must be documented as derived, not upstream-native.

## Recommended OpenASE Runtime Mapping

### For chat mode

Prefer `--output-format stream-json` over `json`.

Map as follows:

- `init`
  - emit session metadata / thread anchor update
- `message(role=user)`
  - usually ignore for transcript if OpenASE already owns the user message
  - optionally preserve as trace for perfect replay fidelity
- `message(role=assistant, delta=true)`
  - emit assistant snapshot/delta stream
- `tool_use`
  - emit tool call event
- `tool_result`
  - emit tool result event
  - if paired tool is `run_shell_command`, derive command output
  - if paired tool is `replace`/`write_file`, derive mutation summary
- `error`
  - emit warning/error trace
- `result`
  - emit final usage + completion/failure

### For ticket/runtime mode

If Gemini is ever used in orchestrated ticket runs, mirror the Claude split:

- protocol layer parses exact Gemini headless events
- adapter layer maps them into:
  - `OutputProduced`
  - `ToolCallRequested`
  - `TurnDiffUpdated`
  - `TokenUsageUpdated`
  - `TurnCompleted`
  - `TurnFailed`
  - `TaskStatus` only when the transport truly carries status-like information

Do not invent fake phase/task events that Gemini CLI never emitted.

## Current Gap Analysis Against This Guide

OpenASE today:

- uses `--output-format json`
- parses final `response`
- parses final usage from `stats`
- misses all streaming tool and warning/error events

To align with the upstream protocol, OpenASE should:

1. add `stream-json` mode support
2. parse exact Gemini headless transport DTOs
3. track `tool_id -> tool_name` correlation
4. preserve final `result.stats`
5. keep derived semantics explicit and allowlist-based

## Testing Strategy

Tests should be written at three layers.

### 1. Protocol parser tests

Add exact JSON fixture tests for:

- `init`
- assistant `message` delta
- `tool_use`
- `tool_result` success
- `tool_result` error
- `error`
- `result` success
- `result` error

These tests should assert exact fields, not just "no error".

### 2. Runtime mapping tests

Add mapping tests for:

- `run_shell_command` tool use/result -> command output
- `replace` / `write_file` tool use/result -> mutation trace
- `google_web_search` / `web_fetch` -> tool call cards, not command cards
- final `result.stats` -> usage event

### 3. Regression tests for known edge cases

From upstream tests and docs:

- assistant deltas arrive in multiple chunks
- cancelled tool calls may still surface as `tool_result.status = success` in stream-json legacy parity mode
  - reference: [`nonInteractiveCliAgentSession.test.ts:2378`](../references/gemini-cli/packages/cli/src/nonInteractiveCliAgentSession.test.ts#L2378)
- warning events may continue before final success result

## Recommended Implementation Order

1. Introduce `gemini_protocol.go` with exact transport DTOs
2. Add a Gemini stream parser that reads JSONL events
3. Switch chat runtime from `--output-format json` to `stream-json`
4. Keep the current JSON parser as fallback only if stream-json is unavailable
5. Add tool-id correlation and derived semantic mapping
6. Add protocol and runtime tests
7. Only then consider whether a richer non-CLI integration is needed

## Final Recommendation

If the goal is "precise and complete" integration through the Gemini CLI binary, OpenASE should treat `stream-json` as the authoritative wire contract.

If the goal is "Claude/Codex-level full fidelity", note that Gemini CLI headless transport is already lossy relative to Gemini's internal `AgentEvent` layer. In that case, the next step would be a richer integration boundary than CLI `stream-json`.
