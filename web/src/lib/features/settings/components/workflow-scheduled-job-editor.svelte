<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { EphemeralChatPanel } from '$lib/features/chat'
  import * as Dialog from '$ui/dialog'
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

  const editorTitle = $derived(selectedJob ? 'Edit scheduled job' : 'Create scheduled job')
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
        <div class="flex items-center justify-between gap-3">
          <Label for="scheduled-job-cron">Cron expression</Label>
          <Button variant="outline" size="sm" onclick={() => (assistantOpen = true)}>
            <Bot class="size-4" />
            AI
          </Button>
        </div>
        <div class="flex items-center gap-2">
          <Input
            id="scheduled-job-cron"
            value={draft.cronExpression}
            placeholder="0 2 * * *"
            oninput={(event) =>
              onFieldChange?.('cronExpression', (event.currentTarget as HTMLInputElement).value)}
          />
        </div>
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
        <Label for="scheduled-job-enabled">Delivery mode</Label>
        <div
          class="border-border bg-muted/20 flex min-h-10 items-center justify-between gap-3 rounded-md border px-3 text-sm"
        >
          <div class="space-y-0.5">
            <Label for="scheduled-job-enabled" class="text-sm font-medium">
              {draft.isEnabled ? 'Enabled' : 'Disabled'}
            </Label>
            <p id="scheduled-job-enabled-description" class="text-muted-foreground text-xs">
              Disabled jobs stay saved but will not schedule new tickets.
            </p>
          </div>
          <Switch
            id="scheduled-job-enabled"
            checked={draft.isEnabled}
            aria-describedby="scheduled-job-enabled-description"
            onCheckedChange={(checked) => onFieldChange?.('isEnabled', checked)}
          />
        </div>
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

<Dialog.Root bind:open={assistantOpen}>
  <Dialog.Content class="flex h-[80vh] max-h-[48rem] max-w-3xl flex-col overflow-hidden p-0">
    <Dialog.Header class="border-border border-b px-6 py-5">
      <Dialog.Title>Cron helper</Dialog.Title>
      <Dialog.Description>
        Ask AI to translate natural-language schedules, review an existing cron expression, or
        suggest safer timing.
      </Dialog.Description>
    </Dialog.Header>

    {#if assistantOpen}
      <EphemeralChatPanel
        source="project_sidebar"
        organizationId={appStore.currentOrg?.id ?? ''}
        defaultProviderId={appStore.currentProject?.default_agent_provider_id ?? null}
        context={{ projectId }}
        title="Scheduled Job AI"
        description="Project context plus cron-specific guidance."
        placeholder="Describe the schedule you want, or ask what the current cron expression means."
        emptyStateTitle="Cron context is ready"
        emptyStateDescription="Ask for cron translation, validation, or safer scheduling guidance for this recurring job."
        contextNote={cronContextNote}
        messagePrefix={cronMessagePrefix}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>
