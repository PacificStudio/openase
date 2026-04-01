<script lang="ts">
  import { cn } from '$lib/utils'
  import type { BoardGroup, BoardStatusOption, BoardTicket } from '../types'
  import BoardColumnComponent from './board-column.svelte'

  let {
    groups,
    statuses = [],
    class: className = '',
    onticketclick,
    draggingTicketId = null,
    dropColumnId = null,
    ondragstartticket,
    ondragendticket,
    ondragovercolumn,
    ondropticket,
    onStatusChange,
    onPriorityChange,
    onCreateTicket,
    onColumnAction,
  }: {
    groups: BoardGroup[]
    statuses?: BoardStatusOption[]
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
    draggingTicketId?: string | null
    dropColumnId?: string | null
    ondragstartticket?: (ticket: BoardTicket) => void
    ondragendticket?: () => void
    ondragovercolumn?: (columnId: string) => void
    ondropticket?: (ticketId: string, columnId: string) => void
    onStatusChange?: (ticketId: string, statusId: string) => void
    onPriorityChange?: (ticketId: string, priority: BoardTicket['priority']) => void
    onCreateTicket?: (statusId: string) => void
    onColumnAction?: (columnId: string, action: string) => void
  } = $props()

  const flatColumns = $derived(groups.flatMap((g) => g.columns))
  const columnIndexById = $derived(new Map(flatColumns.map((col, i) => [col.id, i])))
  const totalColumns = $derived(flatColumns.length)
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
                  {statuses}
                  columnIndex={columnIndexById.get(column.id) ?? 0}
                  columnCount={totalColumns}
                  {onticketclick}
                  {ondragstartticket}
                  {ondragendticket}
                  {ondragovercolumn}
                  {ondropticket}
                  {onStatusChange}
                  {onPriorityChange}
                  {onCreateTicket}
                  {onColumnAction}
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
