<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { ApiError } from '$lib/api/client'
  import {
    listActivity,
    listAgents,
    listStatuses,
    listTickets,
    listWorkflows,
    updateTicket,
  } from '$lib/api/openase'
  import { statusSync } from '$lib/features/statuses/public'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { PageScaffold } from '$lib/components/layout'
  import { Button } from '$ui/button'
  import {
    BoardListView,
    BoardToolbar,
    BoardView,
    buildBoardData,
    filterBoardColumns,
    findTicketLocation,
    patchTicket,
    relocateTicket,
    type BoardColumnType,
    type BoardFilter,
    type BoardTicket,
    type PendingTicketMove,
  } from '$lib/features/board'

  let filter = $state<BoardFilter>({ search: '' })
  let loading = $state(false)
  let error = $state('')
  let allColumns = $state<BoardColumnType[]>([])
  let workflows = $state<string[]>([])
  let agentOptions = $state<string[]>([])
  let draggingTicketId = $state<string | null>(null)
  let dropColumnId = $state<string | null>(null)

  const pendingMoveByTicket = new Map<string, PendingTicketMove>()
  let activeProjectId: string | null = null
  let loadRequestVersion = 0
  let queuedReload = false
  let reloadInFlight = false
  let filteredColumns = $derived(filterBoardColumns(allColumns, filter))

  function isStaleLoad(projectId: string, requestVersion: number) {
    return activeProjectId !== projectId || requestVersion !== loadRequestVersion
  }

  function beginLoad(mode: 'initial' | 'background') {
    if (mode === 'initial') {
      loading = true
    }
    error = ''
  }

  function shouldDeferLoadedBoard(mode: 'initial' | 'background') {
    return mode === 'background' && pendingMoveByTicket.size > 0
  }

  function finishInitialLoad(
    projectId: string,
    requestVersion: number,
    mode: 'initial' | 'background',
  ) {
    if (!isStaleLoad(projectId, requestVersion) && mode === 'initial') {
      loading = false
    }
  }

  async function loadBoard(projectId: string, mode: 'initial' | 'background') {
    const requestVersion = ++loadRequestVersion
    beginLoad(mode)

    try {
      const [statusPayload, ticketPayload, workflowPayload, agentPayload, activityPayload] =
        await Promise.all([
          listStatuses(projectId),
          listTickets(projectId),
          listWorkflows(projectId),
          listAgents(projectId),
          listActivity(projectId, { limit: 200 }),
        ])
      if (isStaleLoad(projectId, requestVersion)) {
        return
      }
      if (shouldDeferLoadedBoard(mode)) {
        queuedReload = true
        return
      }

      const nextBoard = buildBoardData(
        statusPayload.statuses,
        ticketPayload.tickets,
        workflowPayload.workflows,
        agentPayload.agents,
        activityPayload.events,
      )

      workflows = nextBoard.workflowTypes
      agentOptions = nextBoard.agentOptions
      allColumns = nextBoard.columns
    } catch (caughtError) {
      if (isStaleLoad(projectId, requestVersion)) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load tickets.'
    } finally {
      finishInitialLoad(projectId, requestVersion, mode)
    }
  }

  function requestReload(projectId: string) {
    queuedReload = true
    void drainReloadQueue(projectId)
  }

  async function drainReloadQueue(projectId: string) {
    if (
      !queuedReload ||
      reloadInFlight ||
      pendingMoveByTicket.size > 0 ||
      activeProjectId !== projectId
    ) {
      return
    }

    reloadInFlight = true
    queuedReload = false
    try {
      await loadBoard(projectId, 'background')
    } finally {
      reloadInFlight = false
      if (queuedReload && pendingMoveByTicket.size === 0 && activeProjectId === projectId) {
        void drainReloadQueue(projectId)
      }
    }
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const statusVersion = statusSync.version
    activeProjectId = projectId ?? null
    pendingMoveByTicket.clear()
    queuedReload = false
    reloadInFlight = false
    draggingTicketId = null
    dropColumnId = null
    if (!projectId) {
      allColumns = []
      workflows = []
      agentOptions = []
      error = ''
      loading = false
      return
    }

    void statusVersion
    void loadBoard(projectId, 'initial')

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: () => {
        requestReload(projectId)
      },
      onError: (streamError) => {
        console.error('Tickets stream error:', streamError)
      },
    })

    return () => {
      if (activeProjectId === projectId) {
        activeProjectId = null
      }
      disconnect()
    }
  })

  function handleTicketClick(ticket: BoardTicket) {
    appStore.openRightPanel({ type: 'ticket', id: ticket.id })
  }

  function handleTicketDragStart(ticket: BoardTicket) {
    if (ticket.isMoving) return
    draggingTicketId = ticket.id
    dropColumnId = ticket.statusId
  }

  function handleTicketDragEnd() {
    draggingTicketId = null
    dropColumnId = null
  }

  function handleTicketDragOverColumn(columnId: string) {
    if (!draggingTicketId) return
    dropColumnId = columnId
  }

  async function handleTicketDrop(ticketId: string, targetColumnId: string) {
    const projectId = appStore.currentProject?.id
    const location = findTicketLocation(allColumns, ticketId)
    handleTicketDragEnd()

    if (!projectId || !location || location.ticket.isMoving || pendingMoveByTicket.has(ticketId)) {
      return
    }
    if (location.columnId === targetColumnId) {
      return
    }

    pendingMoveByTicket.set(ticketId, {
      fromColumnId: location.columnId,
      fromIndex: location.index,
    })

    allColumns = relocateTicket(allColumns, ticketId, targetColumnId, {
      isMoving: true,
      updatedAt: new Date().toISOString(),
    })

    try {
      await updateTicket(ticketId, { status_id: targetColumnId })
      allColumns = patchTicket(allColumns, ticketId, (ticket) => ({
        ...ticket,
        statusId: targetColumnId,
        isMoving: false,
      }))
    } catch (caughtError) {
      const pendingMove = pendingMoveByTicket.get(ticketId)
      if (pendingMove) {
        allColumns = relocateTicket(allColumns, ticketId, pendingMove.fromColumnId, {
          targetIndex: pendingMove.fromIndex,
          isMoving: false,
        })
      }
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to move ticket to the new status.',
      )
    } finally {
      pendingMoveByTicket.delete(ticketId)
      requestReload(projectId)
    }
  }
</script>

{#snippet actions()}
  <Button
    size="sm"
    disabled={!appStore.currentProject?.id}
    onclick={() => appStore.openNewTicketDialog()}
  >
    New Ticket
  </Button>
{/snippet}

<PageScaffold
  title="Tickets"
  description="Track tickets across workflow statuses."
  variant="workspace"
  {actions}
>
  <div class="flex min-h-0 flex-1 flex-col gap-4">
    <BoardToolbar bind:filter {workflows} agents={agentOptions} />
    {#if error}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {/if}
    {#if loading && allColumns.length === 0}
      <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
        Loading tickets…
      </div>
    {:else if ticketViewStore.mode === 'board'}
      <BoardView
        columns={filteredColumns}
        onticketclick={handleTicketClick}
        ondragstartticket={handleTicketDragStart}
        ondragendticket={handleTicketDragEnd}
        ondragovercolumn={handleTicketDragOverColumn}
        ondropticket={handleTicketDrop}
        {draggingTicketId}
        {dropColumnId}
      />
    {:else}
      <BoardListView columns={filteredColumns} onticketclick={handleTicketClick} />
    {/if}
  </div>
</PageScaffold>
