# Unified Websocket Runtime Contract

The websocket runtime contract is the single execution contract shared by both websocket transport topologies:

- `ws_listener`: the control plane dials the machine listener directly
- `ws_reverse`: the machine daemon keeps a machine-channel session open and tunnels runtime envelopes through `type=runtime`

The contract is implemented in:

- domain model: `internal/domain/websocketruntime/contracts.go`
- protocol server/client: `internal/infra/machinetransport/runtime_protocol.go`
- shared contract suite: `internal/infra/machinetransport/runtime_contract_test.go`

## Message Model

Every runtime frame is a JSON envelope:

```json
{
  "version": 1,
  "type": "request|response|event|hello|hello_ack",
  "request_id": "uuid",
  "operation": "probe|preflight|workspace_prepare|workspace_reset|artifact_sync|command_open|session_input|session_signal|session_close|process_start|process_status|session_output|session_exit",
  "payload": {},
  "error": {
    "code": "invalid_request|protocol_version|workspace|artifact_sync|preflight|session_not_found|process_start|process_signal|transport_unavailable|unauthorized|unsupported|internal",
    "class": "auth|misconfiguration|transient|unsupported|internal",
    "message": "human readable detail",
    "retryable": false,
    "details": {}
  }
}
```

Handshake:

1. Client sends `hello` with supported protocol versions and declared capabilities.
2. Server replies with `hello_ack`, selecting one protocol version and echoing supported operations.
3. All later `request` messages are scoped by `request_id` and answered by a matching `response`.
4. Long-running session output and exits arrive as `event` envelopes.

## Operations

The contract covers all execution-critical runtime actions:

- Reachability and probe: `probe`
- Binary and CLI preflight: `preflight`
- Workspace lifecycle: `workspace_prepare`, `workspace_reset`
- Artifact movement: `artifact_sync`
- Command sessions: `command_open`, `session_input`, `session_signal`, `session_close`
- Process lifecycle: `process_start`, `process_status`, `session_output`, `session_exit`

Upper layers should target these operations rather than branching on websocket topology.

## Versioning And Compatibility

- `version` is mandatory on every envelope.
- Runtime peers must reject unknown protocol versions with `error.code=protocol_version` and `error.class=unsupported`.
- New protocol revisions must be additive where possible:
  - new operations may be introduced without changing existing operation semantics
  - new payload fields should be optional for older peers
  - removing or repurposing an existing operation requires a new protocol version
- `hello` negotiation is the compatibility gate. A peer may only use operations that were acknowledged in `hello_ack`.

## Error Taxonomy

The contract separates orchestration-facing error classes from operation-specific error codes.

- `auth`: credentials or authorization problems that require operator/user action
- `misconfiguration`: fixable input or environment issues such as bad workspace roots, invalid requests, or missing binaries
- `transient`: retryable runtime or transport failures
- `unsupported`: protocol/version/capability mismatches
- `internal`: unexpected implementation failures

Recommended interpretation:

- UX may surface `auth` and `misconfiguration` as user-fixable guidance
- orchestrators may retry only when `retryable=true`, typically for `transient`
- `unsupported` should stop rollout until both peers are upgraded to a compatible contract

## Reverse Session Recovery

Reverse websocket keeps exactly one active machine-channel runtime relay per machine.

- a newer daemon registration replaces the previous reverse runtime relay for that machine
- in-flight command or process sessions on the replaced relay terminate immediately with a transport error instead of falling back to SSH
- disconnects, heartbeat timeouts, and server shutdown close the reverse runtime client and fail in-flight sessions with a clear disconnect cause
- once the daemon reconnects, new runtime requests attach to the new relay session; upper layers decide whether to retry, but the transport contract does not silently change execution planes

## Validation

`TestUnifiedWebsocketRuntimeContractSuite` runs the same suite against:

- a direct listener websocket runtime
- a reverse machine-channel websocket runtime participant

The suite verifies:

- handshake and version negotiation
- probe and preflight
- workspace prepare/reset
- artifact sync
- command session open/stream/close
- process start/status/interrupt/exit

`TestReverseRuntimeRelayRegisterReplacesExistingClientSessions` and
`TestReverseRuntimeRelayRemoveDisconnectsManagedSessions` cover the
replacement/disconnect semantics for reverse-connected runtime sessions.
