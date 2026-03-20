<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { listStatuses, listTickets, listWorkflows, updateTicket } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import type { BoardColumn, BoardFilter, BoardTicket } from '../types'
  import {
    buildBoardColumns,
    filterBoardColumns,
    findTicketLocation,
    patchTicket,
    relocateTicket,
    type PendingTicketMove,
  } from '../model'
  import BoardToolbar from './board-toolbar.svelte'
  import BoardView from './board-view.svelte'

  let filter = $state<BoardFilter>({ search: '' })
  let view = $state<'board' | 'list'>('board')
  let loading = $state(false)
  let error = $state('')
  let mutationError = $state('')
  let allColumns = $state<BoardColumn[]>([])
  let workflows = $state<string[]>([])
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
      const [statusPayload, ticketPayload, workflowPayload] = await Promise.all([
        listStatuses(projectId),
        listTickets(projectId),
        listWorkflows(projectId),
      ])
      if (isStaleLoad(projectId, requestVersion)) {
        return
      }
      if (shouldDeferLoadedBoard(mode)) {
        queuedReload = true
        return
      }

      const nextBoard = buildBoardColumns(
        statusPayload.statuses,
        ticketPayload.tickets,
        workflowPayload.workflows,
      )

      workflows = nextBoard.workflowTypes
      allColumns = nextBoard.columns

      mutationError = ''
    } catch (caughtError) {
      if (isStaleLoad(projectId, requestVersion)) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load board data.'
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
    activeProjectId = projectId ?? null
    pendingMoveByTicket.clear()
    queuedReload = false
    reloadInFlight = false
    draggingTicketId = null
    dropColumnId = null
    mutationError = ''

    if (!projectId) {
      allColumns = []
      workflows = []
      error = ''
      loading = false
      return
    }

    void loadBoard(projectId, 'initial')

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: () => {
        requestReload(projectId)
      },
      onError: (streamError) => {
        console.error('Board tickets stream error:', streamError)
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
    mutationError = ''
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
    mutationError = ''

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
      mutationError =
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to move ticket to the new status.'
    } finally {
      pendingMoveByTicket.delete(ticketId)
      requestReload(projectId)
    }
  }
</script>

<div class="flex h-full flex-col gap-4">
  <BoardToolbar bind:filter bind:view {workflows} agents={[]} listEnabled={false} />
  {#if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {/if}
  {#if mutationError}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {mutationError}
    </div>
  {/if}
  {#if loading && allColumns.length === 0}
    <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
      Loading board…
    </div>
  {:else}
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
  {/if}
</div>
