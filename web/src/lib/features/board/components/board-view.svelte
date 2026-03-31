<script lang="ts">
  import { cn } from '$lib/utils'
  import type { BoardGroup, BoardTicket } from '../types'
  import BoardColumnComponent from './board-column.svelte'

  let {
    groups,
    class: className = '',
    onticketclick,
    draggingTicketId = null,
    dropColumnId = null,
    ondragstartticket,
    ondragendticket,
    ondragovercolumn,
    ondropticket,
  }: {
    groups: BoardGroup[]
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
  {#if groups.length === 0}
    <div
      class="text-muted-foreground flex h-full items-center justify-center rounded-lg border border-dashed p-6 text-sm"
    >
      No board statuses configured yet.
    </div>
  {:else}
    <div class="flex h-full gap-4 p-1">
      {#each groups as group (group.id)}
        <section class="bg-muted/10 flex shrink-0 flex-col gap-3 rounded-xl border px-3 py-2">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="text-foreground text-sm font-semibold">{group.name}</h2>
            </div>
            {#if group.wipInfo}
              <span
                class="bg-background text-muted-foreground rounded-full border px-2 py-1 text-[10px] font-medium whitespace-nowrap"
              >
                {group.wipInfo}
              </span>
            {/if}
          </div>

          {#if group.columns.length === 0}
            <div
              class="text-muted-foreground flex min-h-36 min-w-[280px] items-center justify-center rounded-lg border border-dashed px-4 text-sm"
            >
              No statuses in this group.
            </div>
          {:else}
            <div class="flex gap-3">
              {#each group.columns as column (column.id)}
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
          {/if}
        </section>
      {/each}
    </div>
  {/if}
</div>
