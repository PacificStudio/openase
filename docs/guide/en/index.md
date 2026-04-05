# OpenASE User Guide

OpenASE is a **ticket-driven automated software engineering platform**. Think of it as a "project management hub for AI development teams" — you create tickets describing requirements, AI agents automatically pick them up and execute, with full traceability and control.

## Table of Contents

### Getting Started

- [Quick Start (Startup)](./startup.md) — 5 steps to run the full workflow, from zero to your first AI-executed ticket

### Feature Modules

| Module | Description | Docs |
|--------|-------------|------|
| Tickets | Ticket management, the core unit of the platform | [View](./tickets.md) |
| Agents | AI agent registration & monitoring | [View](./agents.md) |
| Machines | Execution environment management | [View](./machines.md) |
| Updates | Manually published project updates | [View](./updates.md) |
| Activity | System-generated activity logs | [View](./activity.md) |
| Workflows | Agent behavior template definitions | [View](./workflows.md) |
| Skills | Reusable automation skill packs | [View](./skills.md) |
| Scheduled Jobs | Automatic ticket creation on a schedule | [View](./scheduled-jobs.md) |
| Settings | Project configuration | [View](./settings.md) |

### Appendix

- [Module Architecture](./architecture.md) — How modules collaborate with each other
- [Remote Runtime v1 Rollout](../../en/remote-websocket-rollout.md) — Supported remote topologies, migration, validation, and operator rollout guidance
- [FAQ](./faq.md) — Common questions & troubleshooting guide

## How It Works

```
User creates ticket  →  Orchestrator detects pickup status
    →  Agent claims ticket  →  Executes workflow on Machine
    →  Process recorded in Activity  →  Ticket completes
```

Start with the [Quick Start](./startup.md), then read individual module docs as needed.
