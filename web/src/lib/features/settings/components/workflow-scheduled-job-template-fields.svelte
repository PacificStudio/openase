<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import {
    scheduledJobPriorityOptions,
    scheduledJobTypeOptions,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'

  let {
    draft,
    onFieldChange,
  }: {
    draft: ScheduledJobDraft
    onFieldChange?: (field: keyof ScheduledJobDraft, value: string | boolean) => void
  } = $props()
</script>

<div class="mt-3 space-y-3">
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
    <Label for="scheduled-job-ticket-description" class="text-xs">Description</Label>
    <Textarea
      id="scheduled-job-ticket-description"
      rows={3}
      value={draft.ticketDescription}
      placeholder="Include runbook notes, escalation hints, and expected outcome."
      oninput={(event) =>
        onFieldChange?.('ticketDescription', (event.currentTarget as HTMLTextAreaElement).value)}
    />
  </div>

  <div class="grid gap-3 sm:grid-cols-2">
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
  </div>

  <div class="grid gap-3 sm:grid-cols-2">
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
</div>
