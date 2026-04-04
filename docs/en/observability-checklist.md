# Structured Observability Checklist

OpenASE emits structured logs at boundary inputs and ambiguous runtime transitions. This checklist is the minimum contract for new code and for observability audits.

## Core Fields

Include stable correlation fields whenever they are available at the boundary:

- `component`
- `operation`
- `error`
- `request_id`
- `trace_id`
- `span_id`
- `project_id`
- `org_id`
- `ticket_id`
- `workflow_id`
- `run_id`
- `agent_id`
- `provider_id`
- `machine_id`
- `session_id`
- `transport_mode`
- `workspace_root`
- `failure_stage`

## Boundary Rules

- Log once at the boundary where raw external or unstable input enters the system.
- Prefer structured field/value pairs over free-form message text.
- Capture identifiers, payload shape, key counts, or config keys instead of dumping large or sensitive payloads.
- Do not scatter repeated validation logs through downstream business logic after parsing succeeds.
- Use `warn` for malformed or degraded boundary inputs and `error` for server-side failures or impossible states.

## Required Review Areas

- HTTP/API parsing: request path, route, request ID, status code, error code, path params, query keys, content length.
- CLI/config loading: config file path, override keys, failing field or parser, server mode when known.
- SSE and inbound event decoding: topic, event type, payload bytes, published time, scoped IDs.
- Workflow and ticket hooks: hook name, scope, policy, command, duration, outcome, blocking state, related IDs.
- Scheduler/retry/reconciliation: skip reason, capacity snapshot, active counts, retry backoff, pause reason, released runtime IDs.
- Upstream provider/API calls: method, endpoint or operation, status code, upstream request ID, relevant project/provider IDs.
- Notification delivery: rule ID, channel ID/type, config keys, event type, dispatch outcome.
- Historical/partial DB mappings: log when persisted data cannot be parsed into the expected domain form.

## Parse, Not Validate

- Parse unstable input into domain types once.
- Log parse failures at that parse boundary.
- Keep business services operating on already-parsed domain types instead of repeating defensive checks.

## Noise Control

- Do not add info logs for every happy-path helper call.
- Prefer boundary logs, transition logs, warnings, and failure summaries.
- When a skip or failure can happen repeatedly, log the gating reason and capacity snapshot, not raw payload dumps.
