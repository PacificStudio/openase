<script lang="ts">
  import { cn } from '$lib/utils'
  import { Inbox, Plus, Ellipsis, ArrowLeft, ArrowRight, Archive, Trash2 } from '@lucide/svelte'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import type { BoardColumn, BoardStatusOption, BoardTicket } from '../types'
  import TicketCard from './ticket-card.svelte'

  let {
    column,
    statuses = [],
    class: className = '',
    onticketclick,
    ondragstartticket,
    ondragendticket,
    ondragovercolumn,
    ondropticket,
    onStatusChange,
    onPriorityChange,
    onCreateTicket,
    onColumnAction,
    isDropTarget = false,
    draggingTicketId = null,
  }: {
    column: BoardColumn
    statuses?: BoardStatusOption[]
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
    ondragstartticket?: (ticket: BoardTicket) => void
    ondragendticket?: () => void
    ondragovercolumn?: (columnId: string) => void
    ondropticket?: (ticketId: string, columnId: string) => void
    onStatusChange?: (ticketId: string, statusId: string) => void
    onPriorityChange?: (ticketId: string, priority: BoardTicket['priority']) => void
    onCreateTicket?: (statusId: string) => void
    onColumnAction?: (columnId: string, action: string) => void
    isDropTarget?: boolean
    draggingTicketId?: string | null
  } = $props()

  function handleDragOver(event: DragEvent) {
    event.preventDefault()
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move'
    }
    ondragovercolumn?.(column.id)
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault()
    const ticketId =
      event.dataTransfer?.getData('application/x-openase-ticket-id') ||
      event.dataTransfer?.getData('text/plain')
    if (!ticketId) return
    ondropticket?.(ticketId, column.id)
  }

  const showDropPlaceholder = $derived(
    isDropTarget &&
      !!draggingTicketId &&
      !column.tickets.some((ticket) => ticket.id === draggingTicketId),
  )
  const currentStatus = $derived(statuses.find((status) => status.id === column.id) ?? null)
  const stageStatuses = $derived(
    currentStatus ? statuses.filter((status) => status.stage === currentStatus.stage) : [],
  )
  const stageStatusIndex = $derived(
    currentStatus ? stageStatuses.findIndex((status) => status.id === currentStatus.id) : -1,
  )
  const canMoveLeft = $derived(stageStatusIndex > 0)
  const canMoveRight = $derived(
    stageStatusIndex >= 0 && stageStatusIndex < stageStatuses.length - 1,
  )
  const hasConcurrencyLimit = $derived(currentStatus?.maxActiveRuns != null)
  const showInlineCreateButton = $derived(!showDropPlaceholder)
</script>

<div class={cn('flex h-full min-h-0 max-w-[320px] min-w-[280px] shrink-0 flex-col', className)}>
  <div class="mb-2 flex items-center gap-1.5 px-1">
    <span class="size-2.5 rounded-full" style="background-color: {column.color}"></span>
    <span class="text-foreground text-sm font-medium">{column.name}</span>
    <span class="text-muted-foreground text-xs">{column.tickets.length}</span>

    <div class="ml-auto flex items-center">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          class="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex size-6 items-center justify-center rounded transition-colors"
          aria-label="Column actions"
        >
          <Ellipsis class="size-3.5" />
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="end" class="w-40">
          <DropdownMenu.Item
            class="text-xs"
            onclick={() => onColumnAction?.(column.id, 'set_concurrency')}
          >
            Set concurrency
          </DropdownMenu.Item>
          <DropdownMenu.Item
            class="text-xs"
            onclick={() => onColumnAction?.(column.id, 'clear_concurrency')}
            disabled={!hasConcurrencyLimit}
          >
            Clear concurrency
          </DropdownMenu.Item>
          <DropdownMenu.Separator />
          <DropdownMenu.Item
            class="gap-2 text-xs"
            onclick={() => onColumnAction?.(column.id, 'archive_all')}
            disabled={column.tickets.length === 0}
          >
            <Archive class="size-3.5" />
            Archive all
          </DropdownMenu.Item>
          <DropdownMenu.Separator />
          <DropdownMenu.Item
            class="gap-2 text-xs"
            onclick={() => onColumnAction?.(column.id, 'move_left')}
            disabled={!canMoveLeft}
          >
            <ArrowLeft class="size-3.5" />
            Move left
          </DropdownMenu.Item>
          <DropdownMenu.Item
            class="gap-2 text-xs"
            onclick={() => onColumnAction?.(column.id, 'move_right')}
            disabled={!canMoveRight}
          >
            <ArrowRight class="size-3.5" />
            Move right
          </DropdownMenu.Item>
          <DropdownMenu.Separator />
          <DropdownMenu.Item
            class="text-destructive gap-2 text-xs"
            onclick={() => onColumnAction?.(column.id, 'delete')}
          >
            <Trash2 class="size-3.5" />
            Delete
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Root>

      <button
        type="button"
        class="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex size-6 items-center justify-center rounded transition-colors"
        aria-label="Create ticket in {column.name}"
        onclick={() => onCreateTicket?.(column.id)}
      >
        <Plus class="size-3.5" />
      </button>
    </div>
  </div>

  <div
    class={cn(
      'bg-muted/30 flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto rounded-lg border p-1.5 transition-colors',
      showDropPlaceholder ? 'border-primary/60 bg-primary/5' : 'border-transparent',
    )}
    role="list"
    aria-label={`${column.name} tickets`}
    data-column-id={column.id}
    ondragover={handleDragOver}
    ondrop={handleDrop}
  >
    {#if column.tickets.length === 0 && showInlineCreateButton}
      <button
        type="button"
        class="border-border/60 text-muted-foreground hover:border-border hover:text-foreground hover:bg-muted/50 flex w-full shrink-0 items-center justify-center rounded-md border border-dashed py-1.5 transition-colors"
        aria-label="Add ticket to {column.name}"
        onclick={() => onCreateTicket?.(column.id)}
      >
        <Plus class="size-3.5" />
      </button>
    {/if}

    {#if column.tickets.length === 0 && !showDropPlaceholder}
      <div class="text-muted-foreground flex flex-1 flex-col items-center justify-center py-8">
        <Inbox class="mb-2 size-5" />
        <span class="text-xs">No tickets</span>
      </div>
    {/if}

    {#each column.tickets as ticket (ticket.id)}
      <TicketCard
        {ticket}
        {statuses}
        onclick={onticketclick}
        {ondragstartticket}
        {ondragendticket}
        {onStatusChange}
        {onPriorityChange}
        isDragging={draggingTicketId === ticket.id}
        isPendingMove={ticket.isMoving === true}
      />
    {/each}

    {#if showDropPlaceholder}
      <div
        class="border-primary/50 bg-primary/5 text-primary rounded-md border border-dashed px-3 py-2 text-xs font-medium"
      >
        Drop ticket here
      </div>
    {/if}

    {#if column.tickets.length > 0 && showInlineCreateButton}
      <button
        type="button"
        class="border-border/60 text-muted-foreground hover:border-border hover:text-foreground hover:bg-muted/50 flex w-full shrink-0 items-center justify-center rounded-md border border-dashed py-1.5 transition-colors"
        aria-label="Add ticket to {column.name}"
        onclick={() => onCreateTicket?.(column.id)}
      >
        <Plus class="size-3.5" />
      </button>
    {/if}
  </div>
</div>
