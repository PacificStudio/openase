<script lang="ts">
  import { Checkbox } from '$ui/checkbox'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import type { RepoScopeOption as TicketRepoOption } from '$lib/features/repo-scope-selection'
  import {
    scheduledJobPriorityOptions,
    scheduledJobTypeOptions,
    type ScheduledJobDraft,
  } from './workflow-scheduled-jobs'

  let {
    draft,
    repoOptions,
    onFieldChange,
    onToggleRepoScope,
    onUpdateRepoBranchOverride,
  }: {
    draft: ScheduledJobDraft
    repoOptions: TicketRepoOption[]
    onFieldChange?: (field: keyof ScheduledJobDraft, value: string | boolean) => void
    onToggleRepoScope?: (repoId: string) => void
    onUpdateRepoBranchOverride?: (repoId: string, value: string) => void
  } = $props()

  const selectedRepos = $derived(
    repoOptions.filter((repo) => draft.ticketRepoIds.includes(repo.id)),
  )
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

  {#if repoOptions.length > 0}
    <div class="space-y-2 rounded-lg border border-dashed px-3 py-3">
      <div class="space-y-1">
        <Label class="text-xs">Repository scopes</Label>
        <p class="text-muted-foreground text-[11px]">
          {#if repoOptions.length === 1}
            Tickets from this job use the only project repository automatically.
          {:else}
            Select the repositories this job should attach to new tickets in multi-repo projects.
          {/if}
        </p>
      </div>

      {#if repoOptions.length === 1}
        <div class="flex items-center justify-between rounded-md border px-3 py-2 text-xs">
          <div>
            <p class="text-foreground font-medium">{repoOptions[0].label}</p>
            <p class="text-muted-foreground">base: {repoOptions[0].defaultBranch}</p>
          </div>
        </div>
      {:else}
        <div class="space-y-2">
          {#each repoOptions as option (option.id)}
            <label class="flex items-start gap-2 rounded-md border px-3 py-2 text-xs">
              <Checkbox
                class="mt-0.5 size-3.5"
                checked={draft.ticketRepoIds.includes(option.id)}
                onCheckedChange={() => onToggleRepoScope?.(option.id)}
              />
              <div class="min-w-0">
                <p class="text-foreground font-medium">{option.label}</p>
                <p class="text-muted-foreground">base: {option.defaultBranch}</p>
              </div>
            </label>
          {/each}
        </div>
      {/if}

      {#if selectedRepos.length > 0}
        <div class="space-y-2 pt-1">
          <p class="text-muted-foreground text-[11px]">Optional branch overrides</p>
          {#each selectedRepos as option (option.id)}
            <div class="flex items-center gap-2 text-xs">
              <span class="text-muted-foreground w-24 shrink-0 truncate" title={option.label}>
                {option.label}
              </span>
              <Input
                class="h-8 flex-1 text-xs"
                value={draft.ticketRepoBranchOverrides[option.id] ?? ''}
                placeholder={`default: project repo branch (${option.defaultBranch})`}
                oninput={(event) =>
                  onUpdateRepoBranchOverride?.(option.id, event.currentTarget.value)}
              />
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
