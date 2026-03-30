<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import Bot from '@lucide/svelte/icons/bot'
  import * as Select from '$ui/select'
  import { Switch } from '$ui/switch'
  import { Textarea } from '$ui/textarea'
  import {
    scheduledJobPriorityOptions,
    scheduledJobTypeOptions,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'
  import WorkflowScheduledJobCronHelperDialog from './workflow-scheduled-job-cron-helper-dialog.svelte'

  type WorkflowOption = {
    value: string
    label: string
  }

  let {
    draft,
    projectId,
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
    projectId: string
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

  const editorTitle = $derived(selectedJob ? 'Edit scheduled job' : 'New scheduled job')
  let assistantOpen = $state(false)

  const cronContextNote = $derived(
    draft.cronExpression.trim()
      ? `Current cron expression: ${draft.cronExpression.trim()}`
      : 'Describe the cadence in natural language and the assistant will help turn it into a cron expression.',
  )
  const cronMessagePrefix = $derived(
    draft.cronExpression.trim()
      ? [
          'The user is editing a scheduled job for this project.',
          `Current cron expression: ${draft.cronExpression.trim()}.`,
          'Help explain, validate, or revise the cron schedule.',
        ].join('\n')
      : 'The user is creating a scheduled job for this project and needs help drafting a cron expression.',
  )
</script>

<div class="flex min-w-0 flex-1 flex-col">
  <!-- Scrollable form area -->
  <div class="flex-1 overflow-y-auto px-5 py-4">
    <!-- Header -->
    <div class="mb-4">
      <h4 class="text-foreground text-sm font-medium">{editorTitle}</h4>
      {#if selectedJob}
        <div class="text-muted-foreground mt-1 flex flex-wrap gap-3 text-[11px]">
          <span>
            Last run {selectedJob.last_run_at
              ? formatRelativeTime(selectedJob.last_run_at)
              : 'never'}
          </span>
          <span>
            Next run {selectedJob.next_run_at
              ? formatRelativeTime(selectedJob.next_run_at)
              : 'not scheduled'}
          </span>
        </div>
      {/if}
    </div>

    <!-- Section: Job configuration -->
    <fieldset class="space-y-3">
      <legend class="text-muted-foreground mb-2 text-[11px] font-medium tracking-wider uppercase">
        Job configuration
      </legend>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="scheduled-job-name" class="text-xs">Job name</Label>
          <Input
            id="scheduled-job-name"
            value={draft.name}
            placeholder="Nightly regression sweep"
            oninput={(event) =>
              onFieldChange?.('name', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-1.5">
          <div class="flex items-center justify-between">
            <Label for="scheduled-job-cron" class="text-xs">Cron expression</Label>
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground flex items-center gap-1 text-[11px] transition-colors"
              onclick={() => (assistantOpen = true)}
            >
              <Bot class="size-3" />
              AI help
            </button>
          </div>
          <Input
            id="scheduled-job-cron"
            value={draft.cronExpression}
            placeholder="0 2 * * *"
            class="font-mono"
            oninput={(event) =>
              onFieldChange?.('cronExpression', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label class="text-xs">Workflow</Label>
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

        <div class="space-y-1.5">
          <Label for="scheduled-job-enabled" class="text-xs">Status</Label>
          <div class="flex h-9 items-center gap-3 px-1">
            <Switch
              id="scheduled-job-enabled"
              checked={draft.isEnabled}
              onCheckedChange={(checked) => onFieldChange?.('isEnabled', checked)}
            />
            <span class="text-foreground text-sm">
              {draft.isEnabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </div>
      </div>
    </fieldset>

    <!-- Divider -->
    <div class="border-border my-4 border-t"></div>

    <!-- Section: Ticket template -->
    <fieldset class="space-y-3">
      <legend class="text-muted-foreground mb-2 text-[11px] font-medium tracking-wider uppercase">
        Ticket template
      </legend>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="scheduled-job-ticket-title" class="text-xs">Title</Label>
          <Input
            id="scheduled-job-ticket-title"
            value={draft.ticketTitle}
            placeholder="Run scheduled validation"
            oninput={(event) =>
              onFieldChange?.('ticketTitle', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>

        <div class="space-y-1.5">
          <Label for="scheduled-job-ticket-created-by" class="text-xs">Created by</Label>
          <Input
            id="scheduled-job-ticket-created-by"
            value={draft.ticketCreatedBy}
            placeholder="scheduler"
            oninput={(event) =>
              onFieldChange?.('ticketCreatedBy', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
      </div>

      <div class="space-y-1.5">
        <Label for="scheduled-job-ticket-description" class="text-xs">Description</Label>
        <Textarea
          id="scheduled-job-ticket-description"
          rows={3}
          value={draft.ticketDescription}
          placeholder="Include runbook notes, escalation hints, and expected outcome."
          oninput={(event) =>
            onFieldChange?.(
              'ticketDescription',
              (event.currentTarget as HTMLTextAreaElement).value,
            )}
        />
      </div>

      <div class="grid gap-3 sm:grid-cols-3">
        <div class="space-y-1.5">
          <Label class="text-xs">Priority</Label>
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

        <div class="space-y-1.5">
          <Label class="text-xs">Type</Label>
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

        <div class="space-y-1.5">
          <Label for="scheduled-job-ticket-budget" class="text-xs">Budget USD</Label>
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
    </fieldset>
  </div>

  <!-- Sticky action bar -->
  <div class="border-border flex shrink-0 items-center justify-between border-t px-5 py-3">
    <div class="flex items-center gap-2">
      <Button size="sm" onclick={onSubmit} disabled={saving}>
        {saving ? 'Saving…' : selectedJob ? 'Save job' : 'Create job'}
      </Button>

      {#if selectedJob}
        <Button variant="outline" size="sm" onclick={onTrigger} disabled={triggering}>
          {triggering ? 'Triggering…' : 'Run once'}
        </Button>
      {/if}
    </div>

    {#if selectedJob}
      <Button
        variant="ghost"
        size="sm"
        class="text-destructive hover:text-destructive hover:bg-destructive/10"
        onclick={onDelete}
        disabled={deleting}
      >
        {deleting ? 'Deleting…' : 'Delete'}
      </Button>
    {/if}
  </div>
</div>

<WorkflowScheduledJobCronHelperDialog
  bind:open={assistantOpen}
  {projectId}
  {cronContextNote}
  {cronMessagePrefix}
/>
