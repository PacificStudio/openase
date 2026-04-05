# Remote Runtime v1 Rollout Guide

This guide documents the final Remote Runtime v1 model: remote execution is websocket-only, SSH stays in the helper lane, and operators choose between a direct-connect listener or a reverse-connect machine-agent daemon. It covers supported topologies, migration from legacy machine records, validation, observability, daemon install, upgrade and rollback, and the staged rollout checklist.

Terminology used in this guide:

- `direct_connect`: the control plane can reach the machine.
- `reverse_connect`: the machine daemon can dial back to the control plane.
- `websocket`: the intended remote execution path.
- `ssh_compat`: a legacy stored execution value that should be migrated to websocket. SSH remains helper-only for bootstrap, diagnostics, and emergency repair.

## Architecture Summary

Remote Runtime v1 splits remote machine behavior into three separate concerns:

- topology: `direct_connect` or `reverse_connect`
- runtime plane: websocket only
- helper lane: optional SSH bootstrap and diagnostics

That separation maps onto the stored machine fields and runtime entrypoints like this:

| Topology | Stored machine state | Runtime entrypoint | Notes |
| --- | --- | --- | --- |
| Direct-connect listener | `reachability_mode=direct_connect`, `execution_mode=websocket`, `connection_mode=ws_listener` | Control plane dials the saved `advertised_endpoint` | Use this when OpenASE can reach the machine directly |
| Reverse-connect daemon | `reachability_mode=reverse_connect`, `execution_mode=websocket`, `connection_mode=ws_reverse` | Remote host runs `openase machine-agent run` and keeps a machine-channel session open | Use this when the machine can dial out but should not expose an inbound listener |
| Legacy compatibility record | `execution_mode=ssh_compat` | None for normal runtime execution | Migration-only state; ticket execution must not fall back to SSH |

## Automated Validation Matrix

Run the focused transport matrix from the repo root:

```bash
scripts/ci/remote_transport_matrix.sh
```

The fast matrix currently covers:

| Scenario | Coverage |
| --- | --- |
| Unified websocket runtime contract across listener and reverse topologies | `TestUnifiedWebsocketRuntimeContractSuite` |
| SSH bootstrap + reverse websocket machine session | `TestMachineConnectWebsocketPublishesActivityAndMetrics` |
| Reverse websocket runtime happy path with hooks and artifact sync | `TestRuntimeLauncherLaunchesWebsocketReverseRuntimeWithHooksAndArtifactSync` |
| SSH bootstrap helper behavior | `TestRunMachineSSHBootstrapUploadsBinaryEnvAndService` |
| SSH diagnostics helper behavior | `TestRunMachineSSHDiagnosticsReportsBootstrapAndRegistrationIssues` |
| SSH bootstrap + listener websocket runtime | `TestRuntimeLauncherLaunchesWebsocketListenerRuntimeWithHooksAndArtifactSync` |
| Direct SSH runtime rejection | `TestRuntimeLauncherRunTickRejectsSSHRuntimeExecution` |
| Reverse websocket launch without SSH fallback when no reverse session is connected | `TestRuntimeLauncherDoesNotFallBackToSSHWhenWebsocketReverseTransportUnavailable` |
| Remote binary / preflight failure | `TestRuntimeLauncherRecordsWebsocketPreflightFailureStageInActivityAndMetrics` |
| Daemon auth failure | `TestMachineConnectWebsocketAuthFailurePublishesActivityAndMetric` |

What each happy-path runtime case verifies:

- machine registration or reachability
- workspace prepare
- agent process launch
- output streaming or command handshake
- cleanup or disconnect bookkeeping

## CI And Local Validation Strategy

Use the validation layers in this order:

1. `scripts/ci/remote_transport_matrix.sh`
2. `make remote-runtime-container` or `scripts/ci/remote_runtime_container_harness.sh ...` for Linux + Docker Compose hosts
3. targeted operator checks such as `openase machine test <machine-uuid>` and `openase machine ssh-diagnostics <machine-uuid>` during rollout

The intent is:

- fast matrix in normal repo and pull-request validation
- container harness for local-only or manual/nightly rollout-grade checks
- operator commands for machine-specific diagnosis after deployment

## Local Container Harness

Use the slow local-only container harness when you need real daemon startup,
container networking, remote filesystem permissions, SSH transport, and process
execution instead of in-process fakes:

```bash
make remote-runtime-container
```

Run targeted cases directly when you only need one slice:

```bash
scripts/ci/remote_runtime_container_harness.sh listener
scripts/ci/remote_runtime_container_harness.sh reverse ssh
```

The container harness:

- stays out of the normal pull-request CI workflow
- writes case logs plus compose service logs under `.artifacts/remote-runtime-container/`
- requires Linux plus Docker Compose
- uses `scripts/ci/remote_runtime_container.compose.yml`
- is available in a separate manual/nightly workflow: `.github/workflows/remote-runtime-container.yml`

Current container cases:

| Scenario | Coverage |
| --- | --- |
| Direct-connect websocket runtime over a real listener container | `TestWebsocketListenerRuntimeContainerE2E` |
| Reverse-connect websocket runtime over a real machine-agent container | `TestWebsocketReverseRuntimeContainerE2E` |
| SSH bootstrap + diagnostics helper over a real SSH container | `TestMachineSSHHelperContainerE2E` |

The runtime contract itself is now shared by both websocket topologies:

- listener websocket validates the contract over a direct websocket listener
- reverse websocket validates the same contract over machine-channel `runtime` envelopes

For the wire contract, versioning rules, and error taxonomy, see [`docs/en/websocket-runtime-contract.md`](./websocket-runtime-contract.md).

## Observability Contract

### Metrics

OpenASE now emits the following transport-specific metrics:

- `openase.machine_channel.active_sessions`
  - tags: `transport_mode`
- `openase.machine_channel.websocket_reconnect_total`
  - tags: `transport_mode`
- `openase.machine_channel.events_total`
  - tags: `event`, `transport_mode`
- `openase.runtime.launch_failures_total`
  - tags: `failure_stage`, `transport_mode`

Recommended alerting:

- reconnect rate spike on `openase.machine_channel.websocket_reconnect_total`
- non-zero sustained `openase.runtime.launch_failures_total` for `failure_stage in {workspace_transport, openase_preflight, agent_cli_preflight, process_start}`
- unexpected drop or churn in `openase.machine_channel.active_sessions`

### Structured Logs

Transport-related logs should include these correlation fields whenever they are known:

- `machine_id`
- `session_id`
- `transport_mode`
- `workspace_root`
- `failure_stage`
- `ticket_id`
- `run_id`
- `agent_id`

The runtime launcher now logs failure-stage metadata on launch failures, and websocket daemon registration logs carry machine and session identifiers.

### Activity Events

Project activity now records:

- `machine.connected`
- `machine.reconnected`
- `machine.disconnected`
- `machine.daemon_auth_failed`
- `agent.failed`
  - includes `failure_stage`, `transport_mode`, `machine_id`, and `workspace_root` when the failure happened during launch

Use project activity together with logs when you need to answer:

- Which machine lost or replaced a websocket session?
- Did the daemon fail authentication or just reconnect?
- Did the runtime fail in transport setup, workspace prepare, binary preflight, or process start?

## Deployment Prerequisites

### Control Plane URL, TLS, and DNS

- The machine daemon accepts a control-plane base URL, API base URL, or direct websocket endpoint via `--control-plane-url`.
- For production, prefer a stable HTTPS origin with valid DNS and TLS, then pass that origin to the daemon.
- Reverse websocket requires outbound reachability from the remote machine to the control plane.
- Listener websocket requires inbound reachability from the control plane to the machine-advertised websocket endpoint.
- If TLS termination sits in front of OpenASE, make sure the advertised hostname matches the listener certificate and that websocket upgrade headers survive the proxy.

### Reachability And Compatibility Prerequisites

Reverse websocket:

- best when the machine can dial out but cannot expose an inbound listener
- requires a machine channel token
- may keep SSH credentials on the machine record only when operators want helper bootstrap or diagnostics

Listener websocket:

- best when the control plane can directly reach the machine
- requires a machine-advertised websocket endpoint
- may keep SSH credentials configured for helper bootstrap or diagnostics when operators want direct repair access

SSH compatibility:

- is no longer a supported runtime fallback path
- should be treated as legacy record state plus helper infrastructure, not as the remote execution model

## Migrating Existing Machine Records

### Inventory Current State

List machines and identify any record that still stores `execution_mode=ssh_compat` or is missing the topology-specific fields needed for websocket execution:

```bash
openase machine list
```

### Migrate A Direct-connect Machine

Use this path when the control plane can dial the machine directly:

1. Save `reachability_mode=direct_connect`.
2. Save `execution_mode=websocket`.
3. Save a valid `advertised_endpoint`.
4. Resave the machine so `connection_mode` resolves to `ws_listener`.
5. Run:

```bash
openase machine test <machine-uuid>
```

Keep SSH credentials only if you still want helper bootstrap or diagnostics access.

### Migrate A Reverse-connect Machine

Use this path when the machine should dial back to the control plane:

1. Save `reachability_mode=reverse_connect`.
2. Save `execution_mode=websocket`.
3. Save a valid `workspace_root`.
4. Issue a dedicated machine channel token:

```bash
openase machine issue-channel-token \
  --machine-id <machine-uuid> \
  --ttl 24h \
  --format shell
```

5. On the remote host, paste the exported `OPENASE_MACHINE_*` variables and start:

```bash
openase machine-agent run
```

6. Confirm the daemon session and machine checks:

```bash
openase machine test <machine-uuid>
openase machine ssh-diagnostics <machine-uuid>
```

If you already have SSH helper access, `openase machine ssh-bootstrap <machine-uuid>` is the supported way to upload the current binary and install or refresh the reverse daemon service.

### Operator Workflow Changes

After migration, operators should assume all of the following:

- ticket execution stays on websocket for both remote topologies
- direct-connect failures are listener reachability problems first, not SSH fallback candidates
- reverse-connect failures are daemon registration or session health problems first
- SSH commands repair or bootstrap remote access, but they are outside the runtime execution plane

## Bootstrap And Daemon Install

### 1. Build Or Install The OpenASE Binary

Build the current binary from source:

```bash
make build-web
```

### 2. Create Or Update The Machine Record

Before starting the daemon, make sure the machine record has:

- the intended reachability plus execution pair:
  - `reverse_connect + websocket`, or
  - `direct_connect + websocket`
- a valid `workspace_root`
- SSH helper credentials preserved only when you still need bootstrap, diagnostics, or emergency repair access
- `agent_cli_path` when the remote CLI is not discoverable from `PATH`

### 3. Issue A Dedicated Machine Channel Token

On the control-plane host:

```bash
./bin/openase machine issue-channel-token \
  --machine-id <machine-uuid> \
  --ttl 24h \
  --format shell
```

This prints shell exports for:

- `OPENASE_MACHINE_ID`
- `OPENASE_MACHINE_CHANNEL_TOKEN`
- `OPENASE_MACHINE_CONTROL_PLANE_URL`

### 4. Start The Reverse Websocket Daemon

If the control plane can already reach the machine over SSH, you can use the helper flow instead of copying files manually:

```bash
./bin/openase machine ssh-bootstrap <machine-uuid>
```

This uploads the current `openase` binary, writes the machine-agent environment file, installs the per-user service, and restarts it.

Manual fallback for operators who are not using the helper:

On the remote machine:

```bash
export OPENASE_MACHINE_ID=<machine-uuid>
export OPENASE_MACHINE_CHANNEL_TOKEN=<issued-token>
export OPENASE_MACHINE_CONTROL_PLANE_URL=https://openase.example.com

/usr/local/bin/openase machine-agent run \
  --agent-cli-path /usr/local/bin/codex \
  --openase-binary-path /usr/local/bin/openase
```

Recommended `systemd --user` unit:

```ini
[Unit]
Description=OpenASE machine-agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/openase machine-agent run --agent-cli-path /usr/local/bin/codex --openase-binary-path /usr/local/bin/openase
Restart=always
RestartSec=5
Environment=OPENASE_MACHINE_ID=<machine-uuid>
Environment=OPENASE_MACHINE_CHANNEL_TOKEN=<issued-token>
Environment=OPENASE_MACHINE_CONTROL_PLANE_URL=https://openase.example.com

[Install]
WantedBy=default.target
```

For listener websocket mode, deploy the listener endpoint first, then save the advertised websocket URL on the machine record and verify that the control plane can dial it.

## Upgrade And Rollback

### Upgrade

1. Build the target `openase` binary.
2. On the remote machine, install the new binary beside the current one.
3. Restart the `machine-agent` service.
4. Confirm:
   - `machine.connected` or `machine.reconnected` activity appears
   - `openase.machine_channel.active_sessions` returns to the expected level
   - `openase.runtime.launch_failures_total` stays flat during a canary runtime launch

### Rollback

1. Stop the daemon service.
2. Restore the previous binary.
3. Restart the daemon.
4. If websocket launch is still unstable, use `openase machine ssh-diagnostics <machine-uuid>` and `openase machine ssh-bootstrap <machine-uuid>` to repair daemon configuration before widening rollout again.

## Troubleshooting

| Symptom | Likely signal | What to check |
| --- | --- | --- |
| Daemon never registers | `machine.daemon_auth_failed`, `auth_failed` metric | token revoked, token expired, wrong machine ID, wrong control-plane URL |
| Frequent reconnects | `machine.reconnected`, reconnect counter | control-plane restarts, network flaps, proxy idle timeout, heartbeat interval mismatch |
| Runtime fails before launch | `agent.failed` with `failure_stage` | see whether the stage is `workspace_transport`, `workspace_root`, `repo_auth`, `openase_preflight`, `agent_cli_preflight`, or `process_start` |
| Listener websocket cannot dial | launcher logs with `failure_stage=workspace_transport` or `preflight_transport` | DNS resolution, TLS chain, advertised endpoint correctness, firewall reachability |
| Remote binary missing | `failure_stage=openase_preflight` | `.openase/bin/openase` exists in the prepared workspace and can resolve `OPENASE_REAL_BIN` or `PATH` |
| Git clone fails | `failure_stage=repo_auth` | repository credential projection, `GH_TOKEN`, deploy key, or remote git transport policy |
| Workspace root invalid | `failure_stage=workspace_root` | saved machine `workspace_root`, permissions, mounted filesystem availability |

Use `openase machine ssh-diagnostics <machine-uuid>` when you need a quick helper-only readout for workspace permissions, remote binary presence, service status, or recent logs.

## Rollout Checklist

### Stage 1: Transport Compatibility

- run `scripts/ci/remote_transport_matrix.sh`
- confirm direct SSH runtime is rejected
- confirm reverse websocket launch failures do not fall back to SSH
- confirm websocket preflight failures are classified with `failure_stage`

### Stage 2: Reverse Websocket Canary

- enable `ws_reverse` on a small machine subset
- keep SSH credentials only where operators want helper bootstrap or diagnostics
- run at least one reverse websocket happy-path runtime per rollout batch
- verify `machine.connected` and reconnect behavior under a daemon restart
- confirm a forced websocket transport failure stays classified as a websocket-side launch error

### Stage 3: Listener Expansion

- enable `ws_listener` only after reverse websocket canaries are stable
- verify DNS, TLS, and direct control-plane reachability before each listener rollout
- run at least one happy-path listener runtime per rollout batch

### Stage 4: Broad Rollout With Optional SSH Helper

- do not treat SSH as part of the runtime execution plan during the rollout window
- keep dashboard views for:
  - launch success rate by `transport_mode`
  - reconnect recovery
  - orphan or stuck runtime count
  - active websocket machine sessions

### Success Criteria

- runtime launch success rate remains at or above the SSH baseline for two consecutive rollout windows
- reconnect recovery restores sessions without operator action for the target machine cohort
- `openase.runtime.launch_failures_total` stays below the agreed error budget and is dominated by known, actionable stages
- orphan process or stuck-session count does not increase after websocket enablement
- project activity and logs are sufficient to identify machine, session, transport, and failure stage without logging into the machine first
