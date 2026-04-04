# Scheduled Jobs

## What Is This?

Scheduled Jobs let you **automatically create tickets on a schedule**. For example, automatically create a "run regression tests" ticket every night, or a "code review" ticket every Monday. It's like a built-in cron scheduler that fully automates repetitive work.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Cron Expression** | Standard 5-field format: `minute hour day month weekday` |
| **Ticket Template** | Pre-filled ticket data (title, description, workflow binding) |
| **Execution History** | Last run time and next run time |

## Cron Expression Quick Reference

A cron expression consists of 5 fields separated by spaces:

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, 0=Sunday)
│ │ │ │ │
* * * * *
```

**Common examples**:

| Expression | Meaning |
|------------|---------|
| `0 8 * * *` | Every day at 8:00 AM |
| `0 8 * * 1` | Every Monday at 8:00 AM |
| `0 0 1 * *` | First day of every month at midnight |
| `*/30 * * * *` | Every 30 minutes |
| `0 9-17 * * 1-5` | Weekdays every hour (9:00 AM - 5:00 PM) |

## Common Operations

### Creating a Scheduled Job

1. Go to the Scheduled Jobs page
2. Click "New Scheduled Job"
3. Fill in the name and cron expression
4. Configure the ticket template:
   - **Title**: Title for auto-created tickets
   - **Description**: Ticket description content
   - **Workflow**: Linked [Workflow](./workflows.md)
5. Save and enable

### Managing Scheduled Jobs

| Action | Description |
|--------|-------------|
| **Enable / Disable** | Temporarily turn off without deleting |
| **Manual Trigger** | Execute immediately without waiting for the next scheduled time |
| **Edit** | Modify cron expression or ticket template |
| **Delete** | Permanently remove the scheduled job |

## Use Cases

| Scenario | Cron Expression | Description |
|----------|-----------------|-------------|
| **Daily Regression Tests** | `0 2 * * *` | Create test ticket every day at 2:00 AM |
| **Weekly Code Review** | `0 9 * * 1` | Create review ticket every Monday at 9:00 AM |
| **Dependency Security Check** | `0 10 * * 3` | Check dependency vulnerabilities every Wednesday at 10:00 AM |
| **Documentation Sync** | `0 0 1 * *` | Update docs on the first of every month |

## Tips

- Ensure the workflow bound to the scheduled job has agents and status bindings configured, otherwise auto-created tickets won't be executed
- Set execution frequency appropriately to avoid generating a large backlog of unprocessed tickets
- Use "Manual Trigger" to test whether the scheduled job works as expected
- Check "Next Run Time" to confirm your cron expression is correct
