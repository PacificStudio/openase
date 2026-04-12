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
  import { i18nStore } from '$lib/i18n/store.svelte'

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
    <Label for="scheduled-job-ticket-title" class="text-xs">
      {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.title')}
    </Label>
    <Input
      id="scheduled-job-ticket-title"
      value={draft.ticketTitle}
      placeholder={i18nStore.t('settings.workflowScheduledJobTemplateFields.placeholders.title')}
      oninput={(event) =>
        onFieldChange?.('ticketTitle', (event.currentTarget as HTMLInputElement).value)}
    />
  </div>

  <div class="space-y-1.5">
    <Label for="scheduled-job-ticket-description" class="text-xs">
      {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.description')}
    </Label>
    <Textarea
      id="scheduled-job-ticket-description"
      rows={3}
      value={draft.ticketDescription}
      placeholder={i18nStore.t('settings.workflowScheduledJobTemplateFields.placeholders.description')}
      oninput={(event) =>
        onFieldChange?.('ticketDescription', (event.currentTarget as HTMLTextAreaElement).value)}
    />
  </div>

  <div class="grid gap-3 sm:grid-cols-2">
    <div class="space-y-1.5">
      <Label class="text-xs">
        {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.priority')}
      </Label>
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
      <Label class="text-xs">
        {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.type')}
      </Label>
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
      <Label for="scheduled-job-ticket-budget" class="text-xs">
        {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.budget')}
      </Label>
      <Input
        id="scheduled-job-ticket-budget"
        type="number"
        min="0"
        step="0.01"
        value={draft.ticketBudgetUsd}
        placeholder={i18nStore.t('settings.workflowScheduledJobTemplateFields.placeholders.optional')}
        oninput={(event) =>
          onFieldChange?.('ticketBudgetUsd', (event.currentTarget as HTMLInputElement).value)}
      />
    </div>

    <div class="space-y-1.5">
      <Label for="scheduled-job-ticket-created-by" class="text-xs">
        {i18nStore.t('settings.workflowScheduledJobTemplateFields.labels.createdBy')}
      </Label>
      <Input
        id="scheduled-job-ticket-created-by"
        value={draft.ticketCreatedBy}
        placeholder={i18nStore.t('settings.workflowScheduledJobTemplateFields.placeholders.createdBy')}
        oninput={(event) =>
          onFieldChange?.('ticketCreatedBy', (event.currentTarget as HTMLInputElement).value)}
      />
    </div>
  </div>
</div>
