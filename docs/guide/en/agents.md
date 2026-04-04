# Agents

## What Is This?

Agents are the platform's **AI executors**. They are the ones who actually "do the work" — reading code, writing code, running tests, submitting PRs. You don't need to manually instruct them at every step; just create a ticket, and the agent will execute automatically according to the [Workflow](./workflows.md).

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Agent Definition** | A project-level configuration binding an AI Provider (e.g. Claude Code) with project parameters |
| **Agent Run** | A complete execution instance of an agent working on a ticket |
| **Agent Output** | Logs and results produced during execution |
| **Agent Step** | Human-readable action stage descriptions |

## Supported AI Providers

| Provider | Source |
|----------|--------|
| **Claude Code** | Anthropic |
| **Codex** | OpenAI |
| **Gemini CLI** | Google |

## Common Operations

### Registering an Agent

1. Go to the Agents page
2. Select from available Providers
3. Name the agent and confirm creation

### Monitoring Agents

- The sidebar shows a badge with the count of active agents
- Click through to view all agent statuses (Active / Paused / Retired)
- Click a specific agent to see its current tickets and execution history

### Agent Lifecycle Management

| Action | Description |
|--------|-------------|
| **Pause** | Temporarily stop the agent from picking up new tickets |
| **Resume** | Resume from paused state |
| **Retire** | Permanently deactivate an agent |

### Real-time Execution Viewing

Agent execution supports real-time streaming output (SSE). In the Agent Run detail page you can see:

- Step-by-step operation descriptions
- Live log output
- Execution status changes

## Tips

- A single project can have multiple agents, each bound to different workflows (e.g. one for coding, one for testing)
- If an agent behaves abnormally, first check the [Machine](./machines.md) connection status and [Workflow](./workflows.md) Harness configuration
- Agent execution history helps you understand AI decision-making and optimize Harness instructions
