# FAQ (Frequently Asked Questions)

## Agent Related

### Agent was created but doesn't execute tickets?

Check the following:

1. Is there an available [Machine](./machines.md) with a healthy status?
2. Does the [Workflow](./workflows.md) have pickup statuses correctly bound?
3. Is the ticket's current status in the Workflow's pickup status list?
4. Is the agent in "Active" status (not Paused or Retired)?

### What to do when an agent execution fails?

1. Check error events in [Activity](./activity.md)
2. Go to Agent Run details to view output logs
3. Verify the target [Machine](./machines.md) is online with sufficient resources
4. Check the [Workflow](./workflows.md) Harness instructions for correctness

## Ticket Related

### Ticket stuck in "In Progress" for a long time?

1. Check [Activity](./activity.md) logs for the latest events
2. Look at Agent Run details for any errors
3. Confirm the target Machine is online with sufficient resources
4. Check if the Workflow has `TimeoutMinutes` configured

### How to have different types of tickets handled by different agents?

1. Create multiple [Workflows](./workflows.md), each bound to a different agent
2. Select the appropriate Workflow when creating tickets
3. Example: `Coding` Workflow bound to Claude Code, `Testing` Workflow bound to another agent

### How to recover an accidentally archived ticket?

Go to [Settings](./settings.md) → Archived Tickets, find the target ticket and click Restore.

## Workflow Related

### How to write a good Harness?

Refer to the Harness writing tips in the [Workflows documentation](./workflows.md). Core principles:

- Define a clear role
- Provide specific steps
- Set acceptance criteria
- Use template variables to inject ticket context

### Will retiring a workflow affect existing tickets?

Yes. Before retiring, check the impact analysis to see which tickets and scheduled jobs are using that workflow. You can use the "Replace References" feature to batch-migrate to a new workflow.

## Machine Related

### Machine shows "Offline"?

1. Confirm the machine is powered on with normal network connectivity
2. Check if the SSH service is running
3. Verify SSH port and username are correct
4. Click "Test Connection" on the Machines page for diagnostic information

## Scheduled Job Related

### Scheduled job didn't create a ticket on time?

1. Confirm the job is in "Enabled" status
2. Check if the cron expression is correct (verify "Next Run Time")
3. Confirm the bound workflow still exists and hasn't been retired

### How to test a scheduled job?

Use "Manual Trigger" to execute immediately without waiting for the next scheduled time.
