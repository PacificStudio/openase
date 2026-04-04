# Tickets

## What Is This?

Tickets are the **core unit** of the entire platform. Every task that needs to be completed — whether it's fixing a bug, building a new feature, or running tests — exists as a ticket. Think of it as a Jira Issue or GitHub Issue, but it's not just a record — it's also a "task instruction" for AI agents.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Status** | The current stage of a ticket, e.g. "To Do", "In Progress", "Done". Customizable in [Settings](./settings.md) |
| **Workflow** | The execution template linked to the ticket, determining how agents handle it. See [Workflows](./workflows.md) |
| **Priority** | The urgency level of the ticket |
| **Dependencies** | Tickets can have "blocks" and "blocked by" relationships |
| **Parent/Child** | Large tickets can be broken down into sub-tickets |

## View Modes

- **Board View (Kanban)**: Displays tickets as columns by status, drag-and-drop to change status
- **List View**: Table format showing all tickets, good for batch viewing

## Common Operations

### Creating a Ticket

1. Click "New Ticket"
2. Fill in the title and description (Markdown supported)
3. Select a workflow and initial status
4. (Optional) Set priority, dependencies, repository scope

### Managing Tickets

- **Comments**: Add comments below the ticket for discussion, with edit history support
- **External Links**: Link to GitHub PRs, design docs, and other external resources
- **Repository Scope**: Restrict agent access to specific files or directories
- **Archive**: Archive completed or no-longer-needed tickets (recoverable in [Settings](./settings.md))

### Filtering Tickets

- Filter by workflow type
- Filter by assigned agent
- Filter by status

## Tips

- The clearer the ticket description, the better the agent performs. Include: context, expected results, relevant code locations
- Use parent/child tickets to break complex tasks into independently executable sub-tasks
- Setting repository scope limits agent access, improving both security and execution efficiency
