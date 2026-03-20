<script lang="ts">
  import { cn } from '$lib/utils'
  import { Inbox } from '@lucide/svelte'
  import type { BoardColumn, BoardTicket } from '../types'
  import TicketCard from './ticket-card.svelte'

  let {
    column,
    class: className = '',
    onticketclick,
    ondragstartticket,
    ondragendticket,
    ondragovercolumn,
    ondropticket,
    isDropTarget = false,
    draggingTicketId = null,
  }: {
    column: BoardColumn
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
    ondragstartticket?: (ticket: BoardTicket) => void
    ondragendticket?: () => void
    ondragovercolumn?: (columnId: string) => void
    ondropticket?: (ticketId: string, columnId: string) => void
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
</script>

<div class={cn('flex max-w-[320px] min-w-[280px] shrink-0 flex-col', className)}>
  <div class="mb-2 flex items-center gap-2 px-1">
    <span class="size-2.5 rounded-full" style="background-color: {column.color}"></span>
    <span class="text-foreground text-sm font-medium">{column.name}</span>
    <span class="text-muted-foreground text-xs">{column.tickets.length}</span>
    {#if column.wipInfo}
      <span class="text-muted-foreground ml-auto text-[10px]">{column.wipInfo}</span>
    {/if}
  </div>

  <div
    class={cn(
      'bg-muted/30 flex flex-1 flex-col gap-1.5 overflow-y-auto rounded-lg border p-1.5 transition-colors',
      showDropPlaceholder ? 'border-primary/60 bg-primary/5' : 'border-transparent',
    )}
    role="list"
    aria-label={`${column.name} tickets`}
    ondragover={handleDragOver}
    ondrop={handleDrop}
  >
    {#if column.tickets.length === 0 && !showDropPlaceholder}
      <div class="text-muted-foreground flex flex-col items-center justify-center py-8">
        <Inbox class="mb-2 size-5" />
        <span class="text-xs">No tickets</span>
      </div>
    {/if}

    {#each column.tickets as ticket (ticket.id)}
      <TicketCard
        {ticket}
        onclick={onticketclick}
        {ondragstartticket}
        {ondragendticket}
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
  </div>
</div>
