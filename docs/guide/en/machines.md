# Machines

## What Is This?

A Machine is the execution environment for agents. OpenASE now separates two concerns that used to be mixed together:

- **Reachability mode**: how the control plane and the machine reach each other.
- **Runtime plane**: websocket for remote execution, local process for the reserved local machine.
- **Helper lane**: optional SSH access for bootstrap, diagnostics, and emergency repair.

For remote machines, the product model is:

- **Direct Connect**: the control plane can dial the machine.
- **Reverse Connect**: the machine daemon can dial back to the control plane.
- **WebSocket execution**: the preferred remote execution path.
- **SSH helper**: optional access for bootstrap, diagnostics, and emergency repair. It is not part of the normal runtime execution plane.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Machine** | A registered execution environment with identity, reachability, workspace, and helper access details. |
| **Reachability Mode** | `local`, `direct_connect`, or `reverse_connect`. |
| **Execution Mode** | `local_process` or `websocket`. |
| **SSH Helper** | Optional SSH credentials used for bootstrap, diagnostics, or emergency repair while remote runtime stays on websocket. |
| **Health Status** | Current reachability and resource status (Online / Offline / Degraded / Maintenance). |
| **Probe** | Tests the currently implemented access path and collects diagnostic information. |

## Supported Remote Topologies

| Topology | Stored semantics | Runtime path | When to use it |
|---------|------------------|--------------|----------------|
| **Direct-connect listener** | `reachability_mode=direct_connect`, `execution_mode=websocket` | Control plane dials the advertised websocket listener endpoint | The control plane can reach the machine directly over the network |
| **Reverse-connect daemon** | `reachability_mode=reverse_connect`, `execution_mode=websocket` | `openase machine-agent run` keeps a reverse websocket session open | The machine can dial out but should not expose an inbound listener |

## Adding A Machine

1. Go to the Machines page.
2. Click `Add Machine`.
3. Choose the reachability mode:
   - `Local`
   - `Direct Connect`
   - `Reverse Connect`
4. Choose the execution path:
   - Use `Local Process` for the reserved local machine.
   - Use `WebSocket` for remote machines.
5. Fill in the required machine identity and workspace fields.
6. For direct-connect machines, save the advertised listener endpoint.
7. For reverse-connect machines, save the record first, then issue a machine channel token and start `openase machine-agent run` on the remote host.
8. Add SSH helper credentials only if you want `openase machine ssh-bootstrap` or `openase machine ssh-diagnostics`.
9. Run `Connection test` to verify the currently configured path.

## Operational Guidance

- Prefer `direct_connect + websocket` when the control plane can reach the machine endpoint.
- Prefer `reverse_connect + websocket` when the machine can dial out but should not expose an inbound listener.
- Keep SSH helper credentials only when you need bootstrap access, diagnostics, or emergency repair.
- Existing direct-connect records must keep a valid `advertised_endpoint`; reverse-connect records must keep an active daemon registration path.

## Migrating Older Remote Records

1. For direct-connect machines, save a valid websocket `advertised_endpoint`.
2. For reverse-connect machines, issue a machine channel token and start `openase machine-agent run`.
3. Run `openase machine test <machine-id>` after each topology update.
4. Keep SSH credentials only if operators still need helper bootstrap or diagnostics.

## Monitoring Machines

- View each machine's health status, detection state, and latest resource snapshot.
- Use the reachability and execution badges to distinguish topology from the active websocket-based runtime path.
- Distinguish `ws_listener` from `ws_reverse` when triaging network reachability versus daemon registration issues.
- Refresh machine health to update heartbeat, websocket session status, and resource telemetry.

## Tips

- Register at least one machine before creating tickets, otherwise agents have nowhere to execute.
- If agent execution fails, first check whether the machine is reachable through the expected topology and whether websocket transport or daemon health is degraded.
- Use SSH helper access sparingly for bootstrap or diagnostics; normal execution should continue to rely on websocket runtime paths.
