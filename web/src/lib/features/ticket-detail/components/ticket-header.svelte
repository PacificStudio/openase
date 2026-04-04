<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import Archive from '@lucide/svelte/icons/archive'
  import Copy from '@lucide/svelte/icons/copy'
  import Check from '@lucide/svelte/icons/check'
  import Pencil from '@lucide/svelte/icons/pencil'
  import Save from '@lucide/svelte/icons/save'
  import X from '@lucide/svelte/icons/x'
  import * as Popover from '$ui/popover'
  import { cn } from '$lib/utils'
  import { formatBoardPriorityLabel, PriorityIcon } from '$lib/features/board/public'
  import type { BoardPriority } from '$lib/features/board/public'
  import type { TicketDetail, TicketStatusOption } from '../types'

  let {
    ticket,
    statuses,
    savingFields = false,
    archiving = false,
    onClose,
    onArchive,
    onSaveFields,
    onPriorityChange,
  }: {
    ticket: TicketDetail
    statuses: TicketStatusOption[]
    savingFields?: boolean
    archiving?: boolean
    onClose?: () => void
    onArchive?: () => void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onPriorityChange?: (priority: TicketDetail['priority']) => void
  } = $props()

  let copied = $state(false)
  let titleEditOpen = $state(false)
  let titleDraft = $state('')
  let statusOpen = $state(false)
  let priorityOpen = $state(false)

  const priorityOptions: Array<{ value: BoardPriority; label: string }> = [
    { value: 'urgent', label: formatBoardPriorityLabel('urgent') },
    { value: 'high', label: formatBoardPriorityLabel('high') },
    { value: 'medium', label: formatBoardPriorityLabel('medium') },
    { value: 'low', label: formatBoardPriorityLabel('low') },
  ]

  function handlePrioritySelect(value: TicketDetail['priority']) {
    priorityOpen = false
    if (value !== ticket.priority) {
      onPriorityChange?.(value)
    }
  }

  const priorityColors: Record<string, string> = {
    '': 'bg-muted text-muted-foreground border-border',
    urgent: 'bg-red-500/15 text-red-400 border-red-500/20',
    high: 'bg-orange-500/15 text-orange-400 border-orange-500/20',
    medium: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/20',
    low: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  }

  const typeLabels: Record<string, string> = {
    feature: 'Feature',
    bugfix: 'Bug Fix',
    refactor: 'Refactor',
    chore: 'Chore',
  }

  function copyIdentifier() {
    navigator.clipboard.writeText(ticket.identifier)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  const titleDirty = $derived(titleDraft.trim() !== ticket.title)

  $effect(() => {
    if (!titleEditOpen) {
      titleDraft = ticket.title
    }
  })

  function toggleTitleEdit() {
    if (titleEditOpen) {
      handleTitleSave()
      return
    }
    titleDraft = ticket.title
    titleEditOpen = true
  }

  function handleTitleSave() {
    const nextTitle = titleDraft.trim()
    if (!nextTitle) {
      titleDraft = ticket.title
      titleEditOpen = false
      return
    }
    if (nextTitle === ticket.title) {
      titleEditOpen = false
      return
    }
    onSaveFields?.({
      title: nextTitle,
      description: ticket.description,
      statusId: ticket.status.id,
    })
    titleEditOpen = false
  }

  function handleStatusSelect(nextStatusId: string) {
    statusOpen = false
    if (!nextStatusId || nextStatusId === ticket.status.id) {
      return
    }
    onSaveFields?.({
      title: ticket.title,
      description: ticket.description,
      statusId: nextStatusId,
    })
  }

  function cancelTitleEdit() {
    titleDraft = ticket.title
    titleEditOpen = false
  }

  function handleTitleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault()
      handleTitleSave()
    }
    if (event.key === 'Escape') {
      cancelTitleEdit()
    }
  }
</script>

{#if titleEditOpen}
  <div class="border-border flex items-center gap-2 border-b px-4 py-1.5">
    <Input
      bind:value={titleDraft}
      class="h-7 flex-1 text-xs font-medium"
      disabled={savingFields}
      onkeydown={handleTitleKeydown}
    />
    <Button
      variant="outline"
      size="sm"
      class="h-6 px-2 text-[11px]"
      onclick={cancelTitleEdit}
      disabled={savingFields}
    >
      Cancel
    </Button>
    <Button
      size="sm"
      class="h-6 px-2 text-[11px]"
      onclick={handleTitleSave}
      disabled={savingFields || !titleDirty}
    >
      <Save class="size-3" />
      {savingFields ? 'Saving…' : 'Save'}
    </Button>
  </div>
{:else}
  <div class="border-border flex items-center gap-2 border-b px-4 py-1.5">
    <button
      onclick={copyIdentifier}
      class="text-muted-foreground hover:bg-muted flex shrink-0 items-center gap-1 rounded px-1 py-0.5 font-mono text-[11px] transition-colors"
    >
      {ticket.identifier}
      {#if copied}
        <Check class="size-3 text-green-400" />
      {:else}
        <Copy class="size-2.5" />
      {/if}
    </button>
    <Popover.Root bind:open={statusOpen}>
      <Popover.Trigger
        class="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-full border px-1.5 py-0 text-[10px] font-medium transition-opacity hover:opacity-80"
        disabled={savingFields}
        style="background-color: {ticket.status.color}20; color: {ticket.status
          .color}; border-color: {ticket.status.color}30"
      >
        {savingFields ? 'Saving…' : ticket.status.name}
      </Popover.Trigger>
      <Popover.Content align="start" class="w-40 gap-0 p-0.5">
        {#each statuses as status (status.id)}
          <button
            type="button"
            class={cn(
              'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
              status.id === ticket.status.id && 'bg-muted',
            )}
            onclick={() => handleStatusSelect(status.id)}
          >
            <span class="size-2 shrink-0 rounded-full" style="background-color: {status.color}"
            ></span>
            <span class="text-foreground">{status.name}</span>
          </button>
        {/each}
      </Popover.Content>
    </Popover.Root>
    <Popover.Root bind:open={priorityOpen}>
      <Popover.Trigger
        class={cn(
          'inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-full border px-1.5 py-0 text-[10px] font-medium transition-opacity hover:opacity-80',
          priorityColors[ticket.priority],
        )}
        disabled={savingFields}
      >
        <PriorityIcon priority={ticket.priority} class="size-3" />
        {formatBoardPriorityLabel(ticket.priority)}
      </Popover.Trigger>
      <Popover.Content align="start" class="w-36 gap-0 p-0.5">
        {#each priorityOptions as option (option.value)}
          <button
            type="button"
            class={cn(
              'hover:bg-muted flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
              option.value === ticket.priority && 'bg-muted',
            )}
            onclick={() => handlePrioritySelect(option.value)}
          >
            <PriorityIcon priority={option.value} />
            <span class="text-foreground">{option.label}</span>
          </button>
        {/each}
      </Popover.Content>
    </Popover.Root>
    <Badge variant="outline" class="shrink-0 px-1.5 py-0 text-[10px]">
      {typeLabels[ticket.type] ?? ticket.type}
    </Badge>
    <h2 class="min-w-0 flex-1 truncate text-xs leading-snug font-medium">{ticket.title}</h2>
    <div class="ml-auto flex shrink-0 items-center">
      <Button
        variant="ghost"
        size="icon-sm"
        class="size-6"
        onclick={toggleTitleEdit}
        aria-label="Edit title"
      >
        <Pencil class="size-3" />
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        class="text-muted-foreground hover:text-destructive size-6"
        disabled={archiving || ticket.archived}
        onclick={() => {
          if (confirm('Archive this ticket?')) {
            onArchive?.()
          }
        }}
        aria-label="Archive ticket"
        title="Archive ticket"
      >
        <Archive class="size-3" />
      </Button>
      <Button variant="ghost" size="icon-sm" class="size-6" onclick={onClose}>
        <X class="size-3" />
      </Button>
    </div>
  </div>
{/if}
