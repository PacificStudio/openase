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
    updateStatus,
    updateTicket,
  } from '$lib/api/openase'
  import { statusSync } from '$lib/features/statuses/public'
  import { toastStore } from '$lib/stores/toast.svelte'
  import {
    BoardListView,
    BoardToolbar,
    BoardView,
    buildBoardData,
    filterBoardColumns,
    findTicketLocation,
    patchTicket,
    projectBoardGroups,
    relocateTicket,
    type BoardColumnType,
    type BoardFilter,
    type BoardGroupType,
    type BoardStatusOption,
    type BoardTicket,
    type PendingTicketMove,
  } from '$lib/features/board'

  let filter = $state<BoardFilter>({ search: '' })
  let loading = $state(false)
  let error = $state('')
  let allColumns = $state<BoardColumnType[]>([])
  let allGroups = $state<BoardGroupType[]>([])
  let allStatuses = $state<BoardStatusOption[]>([])
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
  let filteredGroups = $derived(projectBoardGroups(allGroups, filteredColumns))

  const isStaleLoad = (projectId: string, requestVersion: number) =>
    activeProjectId !== projectId || requestVersion !== loadRequestVersion

  function beginLoad(mode: 'initial' | 'background') {
    if (mode === 'initial') loading = true
    error = ''
  }

  const shouldDeferLoadedBoard = (mode: 'initial' | 'background') =>
    mode === 'background' && pendingMoveByTicket.size > 0

  const finishInitialLoad = (
    projectId: string,
    requestVersion: number,
    mode: 'initial' | 'background',
  ) => {
    if (!isStaleLoad(projectId, requestVersion) && mode === 'initial') loading = false
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
        statusPayload,
        ticketPayload.tickets,
        workflowPayload.workflows,
        agentPayload.agents,
        activityPayload.events,
      )

      workflows = nextBoard.workflowTypes
      agentOptions = nextBoard.agentOptions
      allStatuses = nextBoard.statusOptions
      allGroups = nextBoard.groups
      allColumns = nextBoard.columns
    } catch (caughtError) {
      if (isStaleLoad(projectId, requestVersion)) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load tickets.'
    } finally {
      finishInitialLoad(projectId, requestVersion, mode)
    }
  }

  const requestReload = (projectId: string) => {
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
      allGroups = []
      allStatuses = []
      workflows = []
      agentOptions = []
      error = ''
      loading = false
      return
    }

    void statusVersion
    void loadBoard(projectId, 'initial')

    const disconnectTickets = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: () => {
        requestReload(projectId)
      },
      onError: (streamError) => {
        console.error('Tickets stream error:', streamError)
      },
    })
    const disconnectAgents = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
      onEvent: () => {
        requestReload(projectId)
      },
      onError: (streamError) => {
        console.error('Agents stream error:', streamError)
      },
    })

    return () => {
      if (activeProjectId === projectId) {
        activeProjectId = null
      }
      disconnectTickets()
      disconnectAgents()
    }
  })

  const handleTicketClick = (ticket: BoardTicket) =>
    appStore.openRightPanel({ type: 'ticket', id: ticket.id })

  function handleTicketDragStart(ticket: BoardTicket) {
    if (ticket.isMoving) return
    draggingTicketId = ticket.id
    dropColumnId = ticket.statusId
  }

  const handleTicketDragEnd = () => {
    draggingTicketId = null
    dropColumnId = null
  }

  const handleTicketDragOverColumn = (columnId: string) => {
    if (!draggingTicketId) return
    dropColumnId = columnId
  }

  const handleStatusChange = async (ticketId: string, statusId: string) =>
    handleTicketDrop(ticketId, statusId)

  async function handlePriorityChange(ticketId: string, priority: string) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    allColumns = patchTicket(allColumns, ticketId, (ticket) => ({
      ...ticket,
      priority: priority as BoardTicket['priority'],
    }))

    try {
      await updateTicket(ticketId, { priority })
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update priority.',
      )
      requestReload(projectId)
    }
  }

  async function handleTicketDrop(ticketId: string, targetColumnId: string) {
    const projectId = appStore.currentProject?.id
    const location = findTicketLocation(allColumns, ticketId)
    handleTicketDragEnd()

    if (!projectId || !location || location.ticket.isMoving || pendingMoveByTicket.has(ticketId))
      return
    if (location.columnId === targetColumnId) return

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

  async function handleColumnAction(columnId: string, action: string) {
    if (action === 'move_left') {
      await handleColumnMove(columnId, 'left')
    } else if (action === 'move_right') {
      await handleColumnMove(columnId, 'right')
    } else if (action === 'set_concurrency') {
      await handleColumnConcurrency(columnId)
    } else if (action === 'clear_concurrency') {
      await handleColumnConcurrency(columnId, null)
    }
  }

  async function handleColumnMove(statusId: string, direction: 'left' | 'right') {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const currentStatus = allStatuses.find((status) => status.id === statusId)
    if (!currentStatus) return

    const stageStatuses = allStatuses.filter((status) => status.stage === currentStatus.stage)
    const currentIndex = stageStatuses.findIndex((status) => status.id === statusId)
    if (currentIndex === -1) return

    const targetIndex = direction === 'left' ? currentIndex - 1 : currentIndex + 1
    if (targetIndex < 0 || targetIndex >= stageStatuses.length) return

    const targetStatus = stageStatuses[targetIndex]
    if (!targetStatus) return

    try {
      await Promise.all([
        updateStatus(currentStatus.id, { position: targetStatus.position }),
        updateStatus(targetStatus.id, { position: currentStatus.position }),
      ])
      requestReload(projectId)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to reorder statuses.',
      )
    }
  }

  async function handleColumnConcurrency(statusId: string, nextMaxActiveRuns?: number | null) {
    const projectId = appStore.currentProject?.id
    if (!projectId) return

    const currentStatus = allStatuses.find((status) => status.id === statusId)
    if (!currentStatus) return

    let parsedMaxActiveRuns = nextMaxActiveRuns
    if (parsedMaxActiveRuns === undefined) {
      const rawValue = window.prompt(
        `Set concurrency limit for "${currentStatus.name}". Leave blank for Unlimited.`,
        currentStatus.maxActiveRuns?.toString() ?? '',
      )
      if (rawValue === null) return

      const normalized = rawValue.trim()
      if (!normalized) {
        parsedMaxActiveRuns = null
      } else if (!/^\d+$/.test(normalized) || Number.parseInt(normalized, 10) < 1) {
        toastStore.error('Status concurrency must be a positive integer or left blank.')
        return
      } else {
        parsedMaxActiveRuns = Number.parseInt(normalized, 10)
      }
    }

    try {
      await updateStatus(statusId, { max_active_runs: parsedMaxActiveRuns })
      requestReload(projectId)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update concurrency.',
      )
    }
  }
</script>

<div class="flex h-full min-h-0 flex-col gap-2 px-4 py-3">
  <BoardToolbar bind:filter {workflows} agents={agentOptions} />
  {#if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {/if}
  {#if loading && allColumns.length === 0}
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
      groups={filteredGroups}
      statuses={allStatuses}
      onticketclick={handleTicketClick}
      ondragstartticket={handleTicketDragStart}
      ondragendticket={handleTicketDragEnd}
      ondragovercolumn={handleTicketDragOverColumn}
      ondropticket={handleTicketDrop}
      onStatusChange={handleStatusChange}
      onPriorityChange={handlePriorityChange}
      onCreateTicket={(statusId) => appStore.openNewTicketDialog(statusId)}
      onColumnAction={handleColumnAction}
      {draggingTicketId}
      {dropColumnId}
    />
  {:else}
    <BoardListView columns={filteredColumns} onticketclick={handleTicketClick} />
  {/if}
</div>
