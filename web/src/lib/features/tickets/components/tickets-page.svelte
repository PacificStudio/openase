<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
  import { ticketBoardToolbarStore } from '../board-toolbar-store.svelte'
  import { BoardListView, BoardToolbar, BoardView } from '$lib/features/board'
  import { createTicketsPageController } from './tickets-page-controller.svelte'

  const controller = createTicketsPageController()
</script>

<div class="flex h-full min-h-0 flex-col gap-2 px-4 py-3">
  <BoardToolbar
    filter={ticketBoardToolbarStore.filter}
    hideEmpty={ticketBoardToolbarStore.hideEmpty}
    workflows={controller.workflows}
    agents={controller.agentOptions}
    onFilterChange={(next) => ticketBoardToolbarStore.setFilter(next)}
    onHideEmptyChange={(next) => ticketBoardToolbarStore.setHideEmpty(next)}
  />
  {#if controller.error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {controller.error}
    </div>
  {/if}
  {#if controller.loading && controller.allColumns.length === 0}
    <!-- Skeleton board columns -->
    {#if ticketViewStore.mode === 'board'}
      <div class="flex min-h-0 flex-1 gap-4 overflow-x-auto p-1">
        {#each { length: 4 } as _}
          <div class="flex max-w-[320px] min-w-[280px] shrink-0 flex-col">
            <div class="mb-2 flex items-center gap-1.5 px-1">
              <div class="bg-muted size-2.5 animate-pulse rounded-full"></div>
              <div class="bg-muted h-4 w-20 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-5 animate-pulse rounded"></div>
            </div>
            <div
              class="bg-muted/30 flex min-h-0 flex-1 flex-col gap-1.5 rounded-lg border border-transparent p-1.5"
            >
              {#each { length: 3 } as __}
                <div class="border-border/60 bg-card rounded-lg border p-3">
                  <div class="flex items-center gap-2">
                    <div class="bg-muted h-3 w-14 animate-pulse rounded"></div>
                    <div class="bg-muted h-3.5 w-32 animate-pulse rounded"></div>
                  </div>
                  <div class="mt-2 flex items-center gap-2">
                    <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
                    <div class="bg-muted h-3 w-12 animate-pulse rounded"></div>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <!-- Skeleton list view -->
      <div class="border-border flex-1 rounded-md border">
        <div class="border-border flex border-b px-4 py-2.5">
          {#each ['w-24', 'w-16', 'w-16', 'w-20', 'w-16', 'w-16'] as w}
            <div class="flex-1 px-2">
              <div class="bg-muted h-3 {w} animate-pulse rounded"></div>
            </div>
          {/each}
        </div>
        {#each { length: 6 } as _}
          <div class="border-border flex items-center border-b px-4 py-3 last:border-0">
            <div class="flex flex-1 items-center gap-2 px-2">
              <div class="bg-muted size-3.5 animate-pulse rounded-full"></div>
              <div class="bg-muted h-3 w-14 animate-pulse rounded"></div>
              <div class="bg-muted h-3.5 w-40 animate-pulse rounded"></div>
            </div>
            <div class="flex-1 px-2">
              <div class="bg-muted h-5 w-16 animate-pulse rounded-full"></div>
            </div>
            <div class="flex-1 px-2">
              <div class="bg-muted h-3 w-12 animate-pulse rounded"></div>
            </div>
            <div class="flex-1 px-2">
              <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
            </div>
            <div class="flex-1 px-2">
              <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
            </div>
            <div class="flex-1 px-2 text-right">
              <div class="bg-muted ml-auto h-3 w-12 animate-pulse rounded"></div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {:else if ticketViewStore.mode === 'board'}
    <BoardView
      groups={controller.filteredGroups}
      statuses={controller.allStatuses}
      hiddenColumns={controller.hiddenColumns}
      onticketclick={controller.handleTicketClick}
      ondragstartticket={controller.handleTicketDragStart}
      ondragendticket={controller.handleTicketDragEnd}
      ondragovercolumn={controller.handleTicketDragOverColumn}
      ondropticket={controller.handleTicketDrop}
      onStatusChange={controller.handleStatusChange}
      onPriorityChange={controller.handlePriorityChange}
      onCreateTicket={(statusId) => appStore.openNewTicketDialog(statusId)}
      onColumnAction={controller.handleColumnAction}
      draggingTicketId={controller.draggingTicketId}
      dropColumnId={controller.dropColumnId}
    />
  {:else}
    <BoardListView
      columns={controller.filteredColumns}
      onticketclick={controller.handleTicketClick}
    />
  {/if}
</div>
