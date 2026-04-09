<script lang="ts">
  import { cn } from '$lib/utils'
  import { ContextMenu as ContextMenuPrimitive } from 'bits-ui'
  import ChevronRightIcon from '@lucide/svelte/icons/chevron-right'
  import { ExternalLink, Archive, Copy, CircleDot, Signal } from '@lucide/svelte'
  import { formatBoardPriorityLabel, type BoardPriority } from '../priority'
  import type { BoardStatusOption, BoardTicket } from '../types'
  import StageIcon from './stage-icon.svelte'
  import PriorityIcon from './priority-icon.svelte'

  let {
    ticket,
    statuses = [],
    open = $bindable(false),
    disabled = false,
    isPendingMove = false,
    onOpenDetails,
    onStatusChange,
    onPriorityChange,
    onArchive,
    children,
  }: {
    ticket: BoardTicket
    statuses?: BoardStatusOption[]
    open?: boolean
    disabled?: boolean
    isPendingMove?: boolean
    onOpenDetails?: (ticket: BoardTicket) => void
    onStatusChange?: (ticketId: string, statusId: string) => void
    onPriorityChange?: (ticketId: string, priority: BoardPriority) => void
    onArchive?: (ticketId: string) => void
    children?: import('svelte').Snippet
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

<ContextMenuPrimitive.Root bind:open>
  <ContextMenuPrimitive.Trigger class="block w-full" {disabled}>
    {@render children?.()}
  </ContextMenuPrimitive.Trigger>

  <ContextMenuPrimitive.Portal>
    <ContextMenuPrimitive.Content
      class={cn(
        'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 bg-popover text-popover-foreground z-50 min-w-48 overflow-x-hidden overflow-y-auto rounded-md p-1 shadow-md ring-1 duration-100 outline-none data-closed:overflow-hidden',
      )}
    >
      <ContextMenuPrimitive.Item
        class={cn(
          'focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
          'gap-2 text-xs',
        )}
        onclick={() => onOpenDetails?.(ticket)}
      >
        <ExternalLink class="size-3.5" />
        Open details
      </ContextMenuPrimitive.Item>

      <ContextMenuPrimitive.Separator class="bg-border -mx-1 my-1 h-px" />

      <ContextMenuPrimitive.Sub>
        <ContextMenuPrimitive.SubTrigger
          class={cn(
            'focus:bg-accent focus:text-accent-foreground data-open:bg-accent data-open:text-accent-foreground flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
            'gap-2 text-xs',
          )}
        >
          <CircleDot class="size-3.5" />
          Change status
          <ChevronRightIcon class="ml-auto size-4" />
        </ContextMenuPrimitive.SubTrigger>
        <ContextMenuPrimitive.SubContent
          class={cn(
            'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 bg-popover text-popover-foreground w-48 rounded-md p-1 shadow-lg ring-1 duration-100',
          )}
        >
          {#each statuses as status (status.id)}
            <ContextMenuPrimitive.Item
              class={cn(
                'focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
                'gap-2 text-xs',
              )}
              disabled={isPendingMove}
              onclick={() => handleStatusSelect(status.id)}
            >
              <StageIcon stage={status.stage} color={status.color} />
              <span class={cn('truncate', status.id === ticket.statusId && 'font-semibold')}>
                {status.name}
              </span>
            </ContextMenuPrimitive.Item>
          {/each}
        </ContextMenuPrimitive.SubContent>
      </ContextMenuPrimitive.Sub>

      <ContextMenuPrimitive.Sub>
        <ContextMenuPrimitive.SubTrigger
          class={cn(
            'focus:bg-accent focus:text-accent-foreground data-open:bg-accent data-open:text-accent-foreground flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
            'gap-2 text-xs',
          )}
        >
          <Signal class="size-3.5" />
          Change priority
          <ChevronRightIcon class="ml-auto size-4" />
        </ContextMenuPrimitive.SubTrigger>
        <ContextMenuPrimitive.SubContent
          class={cn(
            'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 bg-popover text-popover-foreground w-40 rounded-md p-1 shadow-lg ring-1 duration-100',
          )}
        >
          {#each priorityOptions as option (option.value)}
            <ContextMenuPrimitive.Item
              class={cn(
                'focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
                'gap-2 text-xs',
              )}
              disabled={isPendingMove}
              onclick={() => handlePrioritySelect(option.value)}
            >
              <PriorityIcon priority={option.value} />
              <span class={option.value === ticket.priority ? 'font-semibold' : ''}>
                {option.label}
              </span>
            </ContextMenuPrimitive.Item>
          {/each}
        </ContextMenuPrimitive.SubContent>
      </ContextMenuPrimitive.Sub>

      <ContextMenuPrimitive.Separator class="bg-border -mx-1 my-1 h-px" />

      <ContextMenuPrimitive.Item
        class={cn(
          'focus:bg-accent focus:text-accent-foreground relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
          'gap-2 text-xs',
        )}
        onclick={handleCopyTicketId}
      >
        <Copy class="size-3.5" />
        Copy ticket ID
      </ContextMenuPrimitive.Item>

      <ContextMenuPrimitive.Separator class="bg-border -mx-1 my-1 h-px" />

      <ContextMenuPrimitive.Item
        class={cn(
          'focus:bg-destructive/10 focus:text-destructive text-destructive relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
          'gap-2 text-xs',
        )}
        disabled={isPendingMove || isArchiving || ticket.archived}
        onclick={handleArchive}
      >
        <Archive class="size-3.5" />
        Archive
      </ContextMenuPrimitive.Item>
    </ContextMenuPrimitive.Content>
  </ContextMenuPrimitive.Portal>
</ContextMenuPrimitive.Root>
