# Settings

## What Is This?

Settings is the project's **control center**, managing all foundational configurations. Most settings only need to be configured once during project initialization, then adjusted as needed.

## Configuration Modules

### General

- Project name and description
- Default Agent Provider selection

This is the most basic configuration — set it up first after creating a project.

### Ticket Statuses

Manage the list of ticket statuses. These directly affect:
- [Tickets](./tickets.md) board view columns
- [Workflows](./workflows.md) trigger conditions

Each status has a **Stage** attribute:

| Stage | Description | Examples |
|-------|-------------|----------|
| `todo` | To Do | Pending, Planned |
| `in-progress` | In Progress | Developing, Coding |
| `reviewing` | Reviewing | Code Review, Pending Approval |
| `testing` | Testing | Testing, QA Verification |
| `done` | Done | Completed, Released |

**Operations**:
- Create custom statuses
- Adjust display order (drag to sort)
- Edit and delete statuses

> **Important**: The stage attribute of statuses affects workflow auto-pickup logic — make sure they are set correctly.

### Repositories

Connect code repositories to the project so agents can access code:

- Connect GitHub / GitLab repositories
- Configure Outbound Credentials
- Agents can clone code, create branches, submit PRs

After configuration, click "Test Connection" to ensure agents can access code properly.

### Agent Configuration

- Configure the default AI Provider
- Manage agent availability

### Notifications

Set up notification rules for project events:

- Configure notification channels (email, Slack, etc.)
- Choose which events trigger notifications (ticket completion, execution failure, etc.)

### Security

Manage project security credentials:

- GitHub credential management
- SSH key management
- Outbound credential testing

### Archived Tickets

- View archived tickets
- Recover accidentally archived tickets
- Permanently delete unwanted tickets

## Initialization Checklist

When setting up a project for the first time, configure in this order:

1. **General** — Fill in project name and description
2. **Ticket Statuses** — Create a status list suitable for your team (at minimum: "To Do" and "Done")
3. **Repositories** — Connect code repositories and test the connection
4. **Security** — Configure necessary credentials
5. **Notifications** — (Optional) Set up notification rules

## Tips

- Prioritize configuring "Ticket Statuses" and "Repositories" after project creation — they are the foundation for all other features
- Don't create too many statuses; 5-7 is usually enough to cover a complete workflow
- Always test the connection after configuring repository credentials
