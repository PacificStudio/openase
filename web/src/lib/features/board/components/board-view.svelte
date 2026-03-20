<script lang="ts">
  import { cn } from '$lib/utils'
  import type { BoardColumn, BoardTicket } from '../types'
  import BoardColumnComponent from './board-column.svelte'

  let {
    columns,
    class: className = '',
    onticketclick,
    draggingTicketId = null,
    dropColumnId = null,
    ondragstartticket,
    ondragendticket,
    ondragovercolumn,
    ondropticket,
  }: {
    columns: BoardColumn[]
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
    draggingTicketId?: string | null
    dropColumnId?: string | null
    ondragstartticket?: (ticket: BoardTicket) => void
    ondragendticket?: () => void
    ondragovercolumn?: (columnId: string) => void
    ondropticket?: (ticketId: string, columnId: string) => void
  } = $props()
</script>

<div class={cn('flex-1 overflow-x-auto', className)}>
  <div class="flex h-full gap-3 p-1">
    {#each columns as column (column.id)}
      <BoardColumnComponent
        {column}
        {onticketclick}
        {ondragstartticket}
        {ondragendticket}
        {ondragovercolumn}
        {ondropticket}
        isDropTarget={dropColumnId === column.id}
        {draggingTicketId}
      />
    {/each}
  </div>
</div>
