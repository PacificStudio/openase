# Machines

## What Is This?

A Machine is the execution environment for agents. OpenASE now separates two concerns that used to be mixed together:

- **Reachability mode**: how the control plane and the machine reach each other.
- **Execution mode**: how runtime commands actually execute once the machine is reachable.

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
| **Execution Mode** | `local_process` or `websocket`. Older records may still read as `ssh_compat` until they are resaved. |
| **SSH Helper** | Optional SSH credentials used for bootstrap, diagnostics, or emergency repair while remote runtime stays on websocket. |
| **Health Status** | Current reachability and resource status (Online / Offline / Degraded / Maintenance). |
| **Probe** | Tests the currently implemented access path and collects diagnostic information. |

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
5. Fill in the required machine identity, workspace, and endpoint or helper fields.
6. Run `Connection test` to verify the currently configured path.

## Operational Guidance

- Prefer `direct_connect + websocket` when the control plane can reach the machine endpoint.
- Prefer `reverse_connect + websocket` when the machine can dial out but should not expose an inbound listener.
- Keep SSH helper credentials only when you need bootstrap access, diagnostics, or emergency repair.
- Existing older records may still surface as `execution_mode=ssh_compat`; resave them onto websocket before treating the helper as optional cleanup work.

## Monitoring Machines

- View each machine's health status, detection state, and latest resource snapshot.
- Use the reachability and execution badges to distinguish topology from the active websocket-based runtime path.
- Refresh machine health to update heartbeat, websocket session status, and resource telemetry.

## Tips

- Register at least one machine before creating tickets, otherwise agents have nowhere to execute.
- If agent execution fails, first check whether the machine is reachable through the expected topology and whether websocket transport or daemon health is degraded.
- Use SSH helper access sparingly for bootstrap or diagnostics; normal execution should continue to rely on websocket runtime paths.
