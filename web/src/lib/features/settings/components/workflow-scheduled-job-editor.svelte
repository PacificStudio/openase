<script lang="ts">
  import type { ScheduledJob } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import Bot from '@lucide/svelte/icons/bot'
  import { ChevronDown, ChevronRight } from '@lucide/svelte'
  import * as Select from '$ui/select'
  import { Switch } from '$ui/switch'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { type ScheduledJobDraft } from './workflow-scheduled-jobs'
  import WorkflowScheduledJobCronHelperDialog from './workflow-scheduled-job-cron-helper-dialog.svelte'
  import WorkflowScheduledJobTemplateFields from './workflow-scheduled-job-template-fields.svelte'

  type WorkflowOption = {
    value: string
    label: string
  }

  let {
    open = $bindable(false),
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
    onOpenChange,
  }: {
    open?: boolean
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
    onOpenChange?: (open: boolean) => void
  } = $props()

  const editorTitle = $derived(selectedJob ? 'Edit scheduled job' : 'New scheduled job')
  let assistantOpen = $state(false)
  let templateExpanded = $state(false)

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

  const hasTemplateValues = $derived(
    !!(
      draft.ticketTitle ||
      draft.ticketDescription ||
      draft.ticketCreatedBy ||
      draft.ticketBudgetUsd ||
      draft.ticketPriority !== 'medium' ||
      draft.ticketType !== 'feature'
    ),
  )

  let lastReportedOpen = $state(open)

  $effect(() => {
    if (open === lastReportedOpen) return
    lastReportedOpen = open
    onOpenChange?.(open)
  })

  $effect(() => {
    if (open) {
      templateExpanded = hasTemplateValues
    }
  })
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-lg">
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <SheetTitle>{editorTitle}</SheetTitle>
      <SheetDescription>
        {#if selectedJob}
          Last run {selectedJob.last_run_at ? formatRelativeTime(selectedJob.last_run_at) : 'never'}
          · Next run {selectedJob.next_run_at
            ? formatRelativeTime(selectedJob.next_run_at)
            : 'not scheduled'}
        {:else}
          Configure when and how this job creates tickets.
        {/if}
      </SheetDescription>
    </SheetHeader>

    <!-- Scrollable form -->
    <div class="flex-1 overflow-y-auto px-6 py-5">
      <div class="space-y-4">
        <!-- Job name -->
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

        <!-- Cron expression -->
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

        <!-- Workflow -->
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

        <!-- Enabled switch -->
        <div class="flex items-center justify-between">
          <Label for="scheduled-job-enabled" class="text-xs">Enabled</Label>
          <Switch
            id="scheduled-job-enabled"
            checked={draft.isEnabled}
            onCheckedChange={(checked) => onFieldChange?.('isEnabled', checked)}
          />
        </div>

        <!-- Divider -->
        <div class="border-border border-t"></div>

        <!-- Ticket template (collapsible) -->
        <div>
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground flex w-full items-center gap-1.5 py-1 text-xs font-medium transition-colors"
            onclick={() => (templateExpanded = !templateExpanded)}
          >
            {#if templateExpanded}
              <ChevronDown class="size-3.5" />
            {:else}
              <ChevronRight class="size-3.5" />
            {/if}
            Ticket template
            {#if !templateExpanded && hasTemplateValues}
              <span class="text-muted-foreground/60 ml-1 font-normal"> (configured) </span>
            {/if}
          </button>

          {#if templateExpanded}
            <WorkflowScheduledJobTemplateFields {draft} {onFieldChange} />
          {/if}
        </div>
      </div>
    </div>

    <!-- Sticky action bar -->
    <div class="border-border flex shrink-0 items-center justify-between border-t px-6 py-3">
      <div class="flex items-center gap-2">
        <Button size="sm" onclick={onSubmit} disabled={saving}>
          {saving ? 'Saving…' : selectedJob ? 'Save' : 'Create'}
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
  </SheetContent>
</Sheet>

<WorkflowScheduledJobCronHelperDialog
  bind:open={assistantOpen}
  {projectId}
  {cronContextNote}
  {cronMessagePrefix}
/>
