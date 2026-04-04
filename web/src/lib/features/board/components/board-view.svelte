<script lang="ts">
  import { cn } from '$lib/utils'
  import { EyeOff, Plus } from '@lucide/svelte'
  import type { BoardGroup, BoardStatusOption, BoardTicket, HiddenColumn } from '../types'
  import BoardColumnComponent from './board-column.svelte'

  let {
    groups,
    statuses = [],
    hiddenColumns = [],
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
    hiddenColumns?: HiddenColumn[]
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

  const EDGE_ZONE = 80
  const MAX_SCROLL_SPEED = 18

  let scrollContainer: HTMLDivElement | undefined = $state()
  let autoScrollRaf = 0
  let scrollDirection = 0

  function startAutoScroll() {
    if (autoScrollRaf) return
    function tick() {
      if (scrollContainer && scrollDirection !== 0) {
        scrollContainer.scrollLeft += scrollDirection * MAX_SCROLL_SPEED
      }
      autoScrollRaf = requestAnimationFrame(tick)
    }
    autoScrollRaf = requestAnimationFrame(tick)
  }

  function stopAutoScroll() {
    if (autoScrollRaf) {
      cancelAnimationFrame(autoScrollRaf)
      autoScrollRaf = 0
    }
    scrollDirection = 0
  }

  function handleContainerDragOver(event: DragEvent) {
    if (!scrollContainer || !draggingTicketId) return
    const rect = scrollContainer.getBoundingClientRect()
    const x = event.clientX - rect.left
    const rightEdge = rect.width - x

    if (x < EDGE_ZONE) {
      scrollDirection = -(1 - x / EDGE_ZONE)
      startAutoScroll()
    } else if (rightEdge < EDGE_ZONE) {
      scrollDirection = 1 - rightEdge / EDGE_ZONE
      startAutoScroll()
    } else {
      scrollDirection = 0
    }
  }

  $effect(() => {
    if (!draggingTicketId) stopAutoScroll()
  })
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={scrollContainer}
  class={cn('flex min-h-0 flex-1 overflow-x-auto overflow-y-hidden', className)}
  ondragover={handleContainerDragOver}
  ondragleave={stopAutoScroll}
  ondrop={stopAutoScroll}
>
  {#if groups.length === 0}
    <div
      class="text-muted-foreground flex h-full items-center justify-center rounded-lg border border-dashed p-6 text-sm"
    >
      No board statuses configured yet.
    </div>
  {:else}
    <div class="flex h-full min-h-0 gap-4 p-1">
      {#each groups as group (group.id)}
        <section
          class="bg-muted/10 flex h-full min-h-0 shrink-0 flex-col gap-3 rounded-xl border px-3 py-2"
        >
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
            <div class="flex min-h-0 flex-1 gap-3">
              {#each group.columns as column (column.id)}
                <BoardColumnComponent
                  {column}
                  {statuses}
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

      {#if hiddenColumns.length > 0}
        <div class="flex h-full min-h-0 w-48 shrink-0 flex-col rounded-xl border px-3 py-2">
          <div class="text-muted-foreground mb-2 flex items-center gap-1.5">
            <EyeOff class="size-3" />
            <span class="text-xs font-medium">Hidden ({hiddenColumns.length})</span>
          </div>
          <div class="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
            {#each hiddenColumns as col (col.id)}
              <div class="text-muted-foreground flex items-center gap-1.5 rounded-md px-1.5 py-1">
                <span class="size-2 shrink-0 rounded-full" style="background-color: {col.color}"
                ></span>
                <span class="min-w-0 flex-1 truncate text-xs">{col.name}</span>
                <button
                  type="button"
                  class="text-muted-foreground/50 hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 transition-colors"
                  aria-label="Create ticket in {col.name}"
                  onclick={() => onCreateTicket?.(col.id)}
                >
                  <Plus class="size-3" />
                </button>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>
