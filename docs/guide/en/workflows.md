# Workflows

## What Is This?

A Workflow is an **agent behavior template**. If tickets are "what to do", workflows are "how to do it". They define the agent's role, execution instructions, trigger conditions, and lifecycle hooks. Think of it as a "job description" for AI agents.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Harness** | Markdown-format execution instructions telling the agent what role to play and how to complete tasks |
| **Role Type** | The workflow's role positioning, e.g. "Backend Engineer", "QA Tester", "Deployment" |
| **Status Binding** | Configure which ticket statuses trigger automatic agent pickup |
| **Skill Binding** | Attach reusable [Skills](./skills.md) to the workflow |
| **Hooks** | Lifecycle event handlers (on claim, on complete, on fail) |
| **Version** | Workflows support version control, each edit creates a new version |

## Common Operations

### Creating a Workflow

1. Go to the Workflows page
2. Click "New Workflow"
3. Fill in the name and role type
4. Write the Harness document

### Writing a Harness

The Harness is the core of a workflow. Written in Markdown format, it supports template variables for automatic substitution:

```markdown
# Role
You are a backend engineer responsible for API development tasks.

# Task
Please complete the following work based on the ticket description:
- Ticket title: {{ ticket.title }}
- Ticket description: {{ ticket.description }}

# Requirements
- Code must pass all existing tests
- New features need corresponding unit tests
- Run lint checks before submitting a PR
```

**Harness Writing Tips**:

| Practice | Effect |
|----------|--------|
| Define a clear role | Agent better understands its responsibility boundaries |
| Provide specific steps | Reduces agent improvisation, improves consistency |
| Set acceptance criteria | Agent knows when the task is "done" |
| Use template variables | Automatically injects ticket context information |

### Configuring Status Binding

- **Pickup Status**: When a ticket enters these statuses, the agent automatically picks it up
- **Finish Status**: After the agent completes work, the ticket moves to these statuses

This is the key to automation — status binding tells agents when to start working and what to do when done.

### Binding Skills

Select needed [Skills](./skills.md) from existing ones to attach to the workflow. Agents can invoke scripts and operational standards defined in these skills during execution.

### Advanced Configuration

| Parameter | Description |
|-----------|-------------|
| `MaxConcurrent` | Maximum concurrency, limits how many tickets an agent handles simultaneously |
| `MaxRetryAttempts` | Maximum retry count for automatic retries after failure |
| `TimeoutMinutes` | Timeout duration, prevents agents from running indefinitely |

### Managing Workflows

- **Impact Analysis**: Before retiring, see which tickets and scheduled jobs would be affected
- **Replace References**: Batch-switch old workflow references to a new workflow
- **Version History**: View the workflow's modification history

## Built-in Templates

The platform provides ready-to-use workflow templates:

| Template | Purpose |
|----------|---------|
| **Coding** | Code writing and feature development |
| **Testing** | Test execution and reporting |
| **QA** | Quality assurance and code review |
| **Deployment** | Deployment and release |

## Tips

- The more specific the Harness, the better the agent performs. Avoid vague instructions like "please complete this task"
- Set `MaxConcurrent` appropriately to prevent agents from handling too many tickets simultaneously
- Set `TimeoutMinutes` to prevent agents from running indefinitely
- Use version control to track Harness evolution for easy rollback
