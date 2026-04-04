# Machines

## What Is This?

A Machine is the **execution environment** for agents. The agent is the "brain", the machine is the "hands" — agents need a machine to execute code and run commands on. It can be your local development machine, a remote server, or a cloud VM.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Machine** | A registered execution environment with connection information (host, port, username) |
| **Health Status** | Current connection and resource status (Online / Offline / Unhealthy) |
| **Resource Metrics** | CPU, memory, and disk usage snapshots |
| **Probe** | Tests machine connectivity and collects diagnostic information |

## Common Operations

### Adding a Machine

1. Go to the Machines page
2. Click "Add Machine"
3. Fill in the following:
   - **Name**: Display name within the platform
   - **Host**: IP address or domain name
   - **SSH Port**: Default 22
   - **SSH Username**: Login user
4. Click "Test Connection" to verify

### Monitoring Machines

- View each machine's health status (Online / Offline / Unhealthy)
- View resource usage (CPU, memory, disk)
- Click "Refresh" to get the latest status
- Supports real-time streaming health updates (SSE)

### Deleting a Machine

Confirm no agents are currently using the machine before deleting.

## Tips

- Register at least one machine before creating tickets, otherwise agents have nowhere to execute
- If agent execution fails, first check the target machine's connection status and available resources
- Regularly monitor machine resource metrics to avoid failures due to full disk or insufficient memory
