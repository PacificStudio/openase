# Activity

## What Is This?

Activity is a **system-generated event stream**. Every event in the project — ticket creation, agent starting execution, status changes, execution completion — is recorded here. It's like the project's "monitoring dashboard", giving you a clear view of all activity at a glance.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Event** | An activity record containing timestamp, event type, and details |
| **Event Type** | Ticket events, agent events, workflow events, machine events, user actions, etc. |
| **Event Metadata** | Structured context information (e.g. related ticket ID, agent name) |

## Event Types

| Type | Examples |
|------|----------|
| **Ticket Events** | Ticket created, status changed, comment added, archived |
| **Agent Events** | Agent claimed ticket, started execution, completed, failed |
| **Workflow Events** | Workflow created, updated, retired |
| **Machine Events** | Machine online, offline, health check failed |
| **User Actions** | Manual operation records |

## Common Operations

### Viewing Activity

- Go to the Activity page to see a time-ordered event stream (newest first)
- Supports real-time updates (SSE push), no manual refresh needed

### Filtering Events

- **By type**: All / Tickets / Agents / Workflows etc.
- **By keyword**: Ticket identifier, agent name, event content

## Use Cases

- **Troubleshooting**: When agent execution fails, view the complete event chain in Activity
- **Lifecycle tracking**: Trace every step of a ticket from creation to completion
- **Monitoring activity**: Understand overall project health
- **Historical review**: See what happened during a specific time period

## Tips

- Activity is read-only — you cannot manually add or delete events
- If you need to publish subjective project progress, use [Updates](./updates.md)
- Events in Activity contain metadata links — click to jump to the related ticket or agent
