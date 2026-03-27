<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import {
    scheduledJobPriorityOptions,
    scheduledJobTypeOptions,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'

  type WorkflowOption = {
    value: string
    label: string
  }

  let {
    draft,
    selectedJob = null,
    workflowOptions,
    saving = false,
    deleting = false,
    triggering = false,
    onFieldChange,
    onSubmit,
    onDelete,
    onTrigger,
  }: {
    draft: ScheduledJobDraft
    selectedJob?: ScheduledJob | null
    workflowOptions: WorkflowOption[]
    saving?: boolean
    deleting?: boolean
    triggering?: boolean
    onFieldChange?: (field: keyof ScheduledJobDraft, value: string | boolean) => void
    onSubmit?: () => void
    onDelete?: () => void
    onTrigger?: () => void
  } = $props()

  const editorTitle = $derived(selectedJob ? 'Edit scheduled job' : 'Create scheduled job')
</script>

<div class="min-w-0 flex-1 px-5 py-5">
  <div class="space-y-4">
    <div>
      <h4 class="text-foreground text-sm font-medium">{editorTitle}</h4>
      <p class="text-muted-foreground mt-1 text-xs">
        Provide a cron expression, pick the workflow, and shape the ticket template emitted on each
        run.
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="scheduled-job-name">Job name</Label>
        <Input
          id="scheduled-job-name"
          value={draft.name}
          placeholder="Nightly regression sweep"
          oninput={(event) =>
            onFieldChange?.('name', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="scheduled-job-cron">Cron expression</Label>
        <Input
          id="scheduled-job-cron"
          value={draft.cronExpression}
          placeholder="0 2 * * *"
          oninput={(event) =>
            onFieldChange?.('cronExpression', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label>Workflow</Label>
        <Select.Root
          type="single"
          value={draft.workflowId}
          onValueChange={(value) => onFieldChange?.('workflowId', value || '')}
        >
          <Select.Trigger class="w-full">
            {workflowOptions.find((workflow) => workflow.value === draft.workflowId)?.label ??
              'Select workflow'}
          </Select.Trigger>
          <Select.Content>
            {#each workflowOptions as workflow (workflow.value)}
              <Select.Item value={workflow.value}>{workflow.label}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label>Delivery mode</Label>
        <label
          class="border-border bg-muted/20 flex min-h-10 items-center gap-3 rounded-md border px-3 text-sm"
        >
          <input
            type="checkbox"
            checked={draft.isEnabled}
            onchange={(event) =>
              onFieldChange?.('isEnabled', (event.currentTarget as HTMLInputElement).checked)}
          />
          <span>{draft.isEnabled ? 'Enabled' : 'Disabled'}</span>
        </label>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="scheduled-job-ticket-title">Ticket title</Label>
        <Input
          id="scheduled-job-ticket-title"
          value={draft.ticketTitle}
          placeholder="Run scheduled validation"
          oninput={(event) =>
            onFieldChange?.('ticketTitle', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="scheduled-job-ticket-created-by">Created by</Label>
        <Input
          id="scheduled-job-ticket-created-by"
          value={draft.ticketCreatedBy}
          placeholder="scheduler"
          oninput={(event) =>
            onFieldChange?.('ticketCreatedBy', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>
    </div>

    <div class="space-y-2">
      <Label for="scheduled-job-ticket-description">Ticket description</Label>
      <Textarea
        id="scheduled-job-ticket-description"
        rows={4}
        value={draft.ticketDescription}
        placeholder="Include runbook notes, escalation hints, and expected outcome."
        oninput={(event) =>
          onFieldChange?.('ticketDescription', (event.currentTarget as HTMLTextAreaElement).value)}
      />
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <div class="space-y-2">
        <Label>Priority</Label>
        <Select.Root
          type="single"
          value={draft.ticketPriority}
          onValueChange={(value) => onFieldChange?.('ticketPriority', value || 'medium')}
        >
          <Select.Trigger class="w-full capitalize">{draft.ticketPriority}</Select.Trigger>
          <Select.Content>
            {#each scheduledJobPriorityOptions as priority (priority)}
              <Select.Item value={priority} class="capitalize">{priority}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label>Type</Label>
        <Select.Root
          type="single"
          value={draft.ticketType}
          onValueChange={(value) => onFieldChange?.('ticketType', value || 'feature')}
        >
          <Select.Trigger class="w-full capitalize">{draft.ticketType}</Select.Trigger>
          <Select.Content>
            {#each scheduledJobTypeOptions as ticketType (ticketType)}
              <Select.Item value={ticketType} class="capitalize">{ticketType}</Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="space-y-2">
        <Label for="scheduled-job-ticket-budget">Budget USD</Label>
        <Input
          id="scheduled-job-ticket-budget"
          type="number"
          min="0"
          step="0.01"
          value={draft.ticketBudgetUsd}
          placeholder="Optional"
          oninput={(event) =>
            onFieldChange?.('ticketBudgetUsd', (event.currentTarget as HTMLInputElement).value)}
        />
      </div>
    </div>

    {#if selectedJob}
      <div class="text-muted-foreground flex flex-wrap gap-4 text-xs">
        <span
          >Last run {selectedJob.last_run_at
            ? formatRelativeTime(selectedJob.last_run_at)
            : 'never'}</span
        >
        <span
          >Next run {selectedJob.next_run_at
            ? formatRelativeTime(selectedJob.next_run_at)
            : 'not scheduled'}</span
        >
      </div>
    {/if}

    <div class="flex flex-wrap items-center gap-3">
      <Button onclick={onSubmit} disabled={saving}>
        {saving ? 'Saving…' : selectedJob ? 'Save job' : 'Create job'}
      </Button>

      {#if selectedJob}
        <Button variant="outline" onclick={onTrigger} disabled={triggering}>
          {triggering ? 'Triggering…' : 'Run once'}
        </Button>
        <Button variant="destructive" onclick={onDelete} disabled={deleting}>
          {deleting ? 'Deleting…' : 'Delete job'}
        </Button>
      {/if}
    </div>
  </div>
</div>
