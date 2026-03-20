<script lang="ts">
  import { cn } from '$lib/utils'
  import { Inbox } from '@lucide/svelte'
  import type { BoardColumn, BoardTicket } from '../types'
  import TicketCard from './ticket-card.svelte'

  let {
    column,
    class: className = '',
    onticketclick,
  }: {
    column: BoardColumn
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
  } = $props()
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

  <div class="bg-muted/30 flex flex-1 flex-col gap-1.5 overflow-y-auto rounded-lg p-1.5">
    {#if column.tickets.length === 0}
      <div class="text-muted-foreground flex flex-col items-center justify-center py-8">
        <Inbox class="mb-2 size-5" />
        <span class="text-xs">No tickets</span>
      </div>
    {:else}
      {#each column.tickets as ticket (ticket.id)}
        <TicketCard {ticket} onclick={onticketclick} />
      {/each}
    {/if}
  </div>
</div>
