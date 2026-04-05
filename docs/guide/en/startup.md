# Quick Start (Startup)

If this is your first time using OpenASE, follow these 5 steps to get the full workflow running.

## Step 1: Create a Project & Configure Basic Settings

Go to **[Settings](./settings.md)** and complete the following:

- Fill in the project name and description
- Configure ticket statuses (e.g. `To Do`, `In Progress`, `Done`) — these become columns on the board
- Connect a code repository (GitHub / GitLab) so agents can access code

## Step 2: Register a Machine

Go to **[Machines](./machines.md)** and add an execution environment for agents:

- Can be the reserved local machine, a direct-connect listener, or a reverse-connect daemon host
- Remote machines should use websocket runtime; keep SSH only if you want helper bootstrap or diagnostics access
- Click "Test Connection" after adding to verify connectivity

## Step 3: Register an Agent

Go to **[Agents](./agents.md)** and register an agent from an available AI Provider (e.g. Claude Code, Codex, Gemini CLI).

## Step 4: Create a Workflow

Go to **[Workflows](./workflows.md)** and create a workflow template:

- Select the agent to bind
- Write a Harness (instruction document) telling the agent how to execute tasks
- Configure which ticket statuses trigger automatic agent pickup

## Step 5: Create a Ticket

Go to **[Tickets](./tickets.md)** and create your first ticket:

- Fill in the title and description
- Select the workflow you just created
- Set the ticket status to "To Do"

The agent will automatically detect this ticket, pick it up, and begin execution. You can watch the process in real-time in **[Activity](./activity.md)**.

## How It Works

```
User creates ticket  →  Orchestrator detects pickup status
    →  Agent claims ticket  →  Executes workflow on Machine
    →  Process recorded in Activity  →  Ticket completes
```

## Next Steps

After completing the flow above, you've experienced OpenASE's core capabilities. Next you can:

- Dive deeper into [Workflows](./workflows.md) Harness writing to improve agent execution quality
- Explore [Skills](./skills.md) to add reusable capabilities to workflows
- Set up [Scheduled Jobs](./scheduled-jobs.md) to fully automate repetitive work
- Use [Updates](./updates.md) to record project progress for team collaboration
