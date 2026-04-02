<script lang="ts">
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import { Checkbox } from '$ui/checkbox'
  import * as Popover from '$ui/popover'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { ChevronDown, GitBranch } from '@lucide/svelte'
  import { PriorityIcon, StageIcon } from '$lib/features/board/public'
  import {
    ticketPriorityOptions,
    type NewTicketDraft,
    type TicketRepoOption,
    type TicketStatusOption,
  } from '../new-ticket'

  let {
    loading = false,
    saving = false,
    draft,
    statusOptions,
    repoOptions,
    priorityLabels,
    statusPopoverOpen = $bindable(false),
    priorityPopoverOpen = $bindable(false),
    repoPopoverOpen = $bindable(false),
    branchConfigOpen = $bindable(false),
    onSelectStatus,
    onSelectPriority,
    onToggleRepoScope,
    onUpdateRepoBranchOverride,
  }: {
    loading?: boolean
    saving?: boolean
    draft: NewTicketDraft
    statusOptions: TicketStatusOption[]
    repoOptions: TicketRepoOption[]
    priorityLabels: Record<string, string>
    statusPopoverOpen?: boolean
    priorityPopoverOpen?: boolean
    repoPopoverOpen?: boolean
    branchConfigOpen?: boolean
    onSelectStatus?: (statusId: string) => void
    onSelectPriority?: (priority: NewTicketDraft['priority']) => void
    onToggleRepoScope?: (repoId: string) => void
    onUpdateRepoBranchOverride?: (repoId: string, value: string) => void
  } = $props()

  const selectedStatus = $derived(
    statusOptions.find((status) => status.id === draft.statusId) ?? null,
  )
  const selectedRepoCount = $derived(draft.repoIds.length)
</script>

<div class="flex flex-wrap items-center gap-2">
  <Popover.Root bind:open={statusPopoverOpen}>
    <Popover.Trigger
      class={cn(
        'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
        (loading || saving) && 'pointer-events-none opacity-50',
      )}
      disabled={loading || saving || statusOptions.length === 0}
    >
      {#if selectedStatus}
        <StageIcon stage={selectedStatus.stage} color={selectedStatus.color} class="size-3.5" />
        <span class="text-foreground">{selectedStatus.label}</span>
      {:else}
        <span class="text-muted-foreground">Status</span>
      {/if}
      <ChevronDown class="text-muted-foreground size-3" />
    </Popover.Trigger>
    <Popover.Content align="start" class="w-48 gap-0 p-0.5">
      {#each statusOptions as status (status.id)}
        <button
          type="button"
          class={cn(
            'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
            status.id === draft.statusId && 'bg-muted',
          )}
          onclick={() => onSelectStatus?.(status.id)}
        >
          <StageIcon stage={status.stage} color={status.color} class="size-3.5" />
          <span class="text-foreground">{status.label}</span>
        </button>
      {/each}
    </Popover.Content>
  </Popover.Root>

  <Popover.Root bind:open={priorityPopoverOpen}>
    <Popover.Trigger
      class={cn(
        'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
        (loading || saving) && 'pointer-events-none opacity-50',
      )}
      disabled={loading || saving}
    >
      <PriorityIcon priority={draft.priority} class="size-3.5" />
      <span class="text-foreground">{priorityLabels[draft.priority]}</span>
      <ChevronDown class="text-muted-foreground size-3" />
    </Popover.Trigger>
    <Popover.Content align="start" class="w-36 gap-0 p-0.5">
      {#each ticketPriorityOptions as priority (priority)}
        <button
          type="button"
          class={cn(
            'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
            priority === draft.priority && 'bg-muted',
          )}
          onclick={() => onSelectPriority?.(priority)}
        >
          <PriorityIcon {priority} class="size-3.5" />
          <span class="text-foreground">{priorityLabels[priority]}</span>
        </button>
      {/each}
    </Popover.Content>
  </Popover.Root>

  {#if repoOptions.length > 0}
    <Popover.Root bind:open={repoPopoverOpen}>
      <Popover.Trigger
        class={cn(
          'border-border hover:bg-muted inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors',
          (loading || saving) && 'pointer-events-none opacity-50',
        )}
        disabled={loading || saving}
      >
        <GitBranch class="text-muted-foreground size-3.5" />
        {#if selectedRepoCount === 0}
          <span class="text-muted-foreground">Repos</span>
        {:else if selectedRepoCount === 1}
          <span class="text-foreground max-w-28 truncate">
            {repoOptions.find((repo) => draft.repoIds.includes(repo.id))?.label ?? '1 repo'}
          </span>
        {:else}
          <span class="text-foreground">{selectedRepoCount} repos</span>
        {/if}
        <ChevronDown class="text-muted-foreground size-3" />
      </Popover.Trigger>
      <Popover.Content align="start" class="max-h-56 w-64 gap-0 overflow-y-auto p-1">
        {#each repoOptions as option (option.id)}
          <label
            class="hover:bg-muted flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-1.5 text-xs transition-colors"
          >
            <Checkbox
              class="size-3.5"
              checked={draft.repoIds.includes(option.id)}
              disabled={loading || saving}
              onCheckedChange={() => onToggleRepoScope?.(option.id)}
            />
            <div class="min-w-0 flex-1">
              <span class="text-foreground truncate">{option.label}</span>
              <span class="text-muted-foreground ml-1">base: {option.defaultBranch}</span>
            </div>
          </label>
        {/each}
      </Popover.Content>
    </Popover.Root>
  {/if}
</div>

{#if selectedRepoCount > 0}
  <div class="space-y-3 rounded-lg border p-3">
    <div class="flex items-center justify-between gap-3">
      <div>
        <p class="text-sm font-medium">Branch configuration</p>
        <p class="text-muted-foreground text-xs">
          By default OpenASE will use the generated ticket branch for each selected repo.
        </p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={loading || saving}
        onclick={() => {
          branchConfigOpen = !branchConfigOpen
        }}
      >
        {branchConfigOpen ? 'Hide advanced' : 'Advanced'}
      </Button>
    </div>

    {#if branchConfigOpen}
      <div class="space-y-3">
        {#each repoOptions.filter( (option) => draft.repoIds.includes(option.id), ) as option (option.id)}
          <div class="space-y-2 rounded-md border p-3">
            <div>
              <p class="text-sm font-medium">{option.label}</p>
              <p class="text-muted-foreground text-xs">Base branch: {option.defaultBranch}</p>
            </div>
            <div class="space-y-1">
              <Label for={`repo-override-${option.id}`}>Work branch override</Label>
              <Input
                id={`repo-override-${option.id}`}
                value={draft.repoBranchOverrides[option.id] ?? ''}
                placeholder="Leave blank to use the generated ticket branch"
                disabled={loading || saving}
                oninput={(event) =>
                  onUpdateRepoBranchOverride?.(option.id, event.currentTarget.value)}
              />
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}
