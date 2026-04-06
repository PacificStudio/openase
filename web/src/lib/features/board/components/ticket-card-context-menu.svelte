<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { ExternalLink, Archive, Copy, CircleDot, Signal } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import { formatBoardPriorityLabel, type BoardPriority } from '../priority'
  import type { BoardStatusOption, BoardTicket } from '../types'
  import StageIcon from './stage-icon.svelte'
  import PriorityIcon from './priority-icon.svelte'

  let {
    ticket,
    statuses = [],
    open = $bindable(false),
    isPendingMove = false,
    onOpenDetails,
    onStatusChange,
    onPriorityChange,
    onArchive,
  }: {
    ticket: BoardTicket
    statuses?: BoardStatusOption[]
    open?: boolean
    isPendingMove?: boolean
    onOpenDetails?: (ticket: BoardTicket) => void
    onStatusChange?: (ticketId: string, statusId: string) => void
    onPriorityChange?: (ticketId: string, priority: BoardPriority) => void
    onArchive?: (ticketId: string) => void
  } = $props()

  const priorityOptions: Array<{ value: BoardPriority; label: string }> = [
    { value: '', label: formatBoardPriorityLabel('') },
    { value: 'urgent', label: formatBoardPriorityLabel('urgent') },
    { value: 'high', label: formatBoardPriorityLabel('high') },
    { value: 'medium', label: formatBoardPriorityLabel('medium') },
    { value: 'low', label: formatBoardPriorityLabel('low') },
  ]

  let isArchiving = $state(false)

  function handleCopyTicketId() {
    navigator.clipboard.writeText(ticket.identifier)
  }

  function handleArchive() {
    if (isArchiving || isPendingMove) return
    isArchiving = true
    onArchive?.(ticket.id)
  }

  function handleStatusSelect(statusId: string) {
    if (statusId === ticket.statusId) return
    onStatusChange?.(ticket.id, statusId)
  }

  function handlePrioritySelect(priority: BoardPriority) {
    if (priority === ticket.priority) return
    onPriorityChange?.(ticket.id, priority)
  }
</script>

<DropdownMenu.Root bind:open>
  <DropdownMenuPrimitive.Trigger
    style="display: none; position: absolute; pointer-events: none;"
    tabindex={-1}
    aria-hidden="true"
  />
  <DropdownMenu.Content class="w-48">
    <DropdownMenu.Item class="gap-2 text-xs" onclick={() => onOpenDetails?.(ticket)}>
      <ExternalLink class="size-3.5" />
      Open details
    </DropdownMenu.Item>

    <DropdownMenu.Separator />

    <DropdownMenu.Sub>
      <DropdownMenu.SubTrigger class="gap-2 text-xs">
        <CircleDot class="size-3.5" />
        Change status
      </DropdownMenu.SubTrigger>
      <DropdownMenu.SubContent class="w-48">
        {#each statuses as status (status.id)}
          <DropdownMenu.Item
            class="gap-2 text-xs"
            disabled={isPendingMove}
            onclick={() => handleStatusSelect(status.id)}
          >
            <StageIcon stage={status.stage} color={status.color} />
            <span class="truncate {status.id === ticket.statusId ? 'font-semibold' : ''}">
              {status.name}
            </span>
          </DropdownMenu.Item>
        {/each}
      </DropdownMenu.SubContent>
    </DropdownMenu.Sub>

    <DropdownMenu.Sub>
      <DropdownMenu.SubTrigger class="gap-2 text-xs">
        <Signal class="size-3.5" />
        Change priority
      </DropdownMenu.SubTrigger>
      <DropdownMenu.SubContent class="w-40">
        {#each priorityOptions as option (option.value)}
          <DropdownMenu.Item
            class="gap-2 text-xs"
            disabled={isPendingMove}
            onclick={() => handlePrioritySelect(option.value)}
          >
            <PriorityIcon priority={option.value} />
            <span class={option.value === ticket.priority ? 'font-semibold' : ''}>
              {option.label}
            </span>
          </DropdownMenu.Item>
        {/each}
      </DropdownMenu.SubContent>
    </DropdownMenu.Sub>

    <DropdownMenu.Separator />

    <DropdownMenu.Item class="gap-2 text-xs" onclick={handleCopyTicketId}>
      <Copy class="size-3.5" />
      Copy ticket ID
    </DropdownMenu.Item>

    <DropdownMenu.Separator />

    <DropdownMenu.Item
      class="text-destructive gap-2 text-xs"
      disabled={isPendingMove || isArchiving || ticket.archived}
      onclick={handleArchive}
    >
      <Archive class="size-3.5" />
      Archive
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>
