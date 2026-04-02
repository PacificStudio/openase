import { appStore } from '$lib/stores/app.svelte'
import { ApiError } from '$lib/api/client'
import { listActivity, listAgents, listStatuses, listTickets, listWorkflows } from '$lib/api/openase'
import { subscribeProjectEvents } from '$lib/features/project-events'
import { statusSync } from '$lib/features/statuses/public'
import {
  markProjectBoardCacheDirty,
  readProjectBoardCache,
  syncProjectBoardCacheStatusVersion,
  writeProjectBoardCache,
} from '../board-cache'
import { ticketBoardToolbarStore } from '../board-toolbar-store.svelte'
import {
  buildBoardData,
  filterBoardColumns,
  projectBoardGroups,
  type BoardData,
  type BoardGroupType,
  type BoardTicket,
  type HiddenColumn,
  type PendingTicketMove,
} from '$lib/features/board'
import type { BoardColumnType, BoardStatusOption } from '$lib/features/board'
import {
  type TicketsPageControllerActionsState,
  handleColumnAction as runColumnAction,
  handlePriorityChange as runPriorityChange,
  handleTicketDragEnd as endTicketDrag,
  handleTicketDragOverColumn as dragOverTicketColumn,
  handleTicketDragStart as startTicketDrag,
  handleTicketDrop as dropTicket,
} from './tickets-page-controller-actions'

export function createTicketsPageController() {
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
  const controllerState: TicketsPageControllerActionsState = {
    get allColumns() { return allColumns },
    set allColumns(value) { allColumns = value },
    get allStatuses() { return allStatuses },
    get pendingMoveByTicket() { return pendingMoveByTicket },
    get draggingTicketId() { return draggingTicketId },
    set draggingTicketId(value) { draggingTicketId = value },
    get dropColumnId() { return dropColumnId },
    set dropColumnId(value) { dropColumnId = value },
    persistBoardSnapshot,
    requestReload,
  }

  const filteredColumns = $derived(filterBoardColumns(allColumns, ticketBoardToolbarStore.filter))
  const filteredGroups = $derived(
    projectBoardGroups(allGroups, filteredColumns, {
      hideEmpty: ticketBoardToolbarStore.hideEmpty,
    }),
  )
  const hiddenColumns = $derived.by((): HiddenColumn[] => {
    if (!ticketBoardToolbarStore.hideEmpty) return []
    const visibleIds = new Set(filteredGroups.flatMap((group) => group.columns.map((column) => column.id)))
    return filteredColumns
      .filter((column) => !visibleIds.has(column.id))
      .map((column) => ({
        id: column.id,
        name: column.name,
        color: column.color,
        ticketCount: column.tickets.length,
      }))
  })

  const isStaleLoad = (projectId: string, requestVersion: number) =>
    activeProjectId !== projectId || requestVersion !== loadRequestVersion

  function applyBoardSnapshot(nextBoard: BoardData) {
    workflows = nextBoard.workflowTypes
    agentOptions = nextBoard.agentOptions
    allStatuses = nextBoard.statusOptions
    allGroups = nextBoard.groups
    allColumns = nextBoard.columns
  }

  function persistBoardSnapshot(projectId: string) {
    writeProjectBoardCache(projectId, {
      workflowTypes: workflows,
      agentOptions,
      statusOptions: allStatuses,
      groups: allGroups,
      columns: allColumns,
    })
  }

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
      if (isStaleLoad(projectId, requestVersion)) return
      if (shouldDeferLoadedBoard(mode)) {
        queuedReload = true
        return
      }

      applyBoardSnapshot(
        buildBoardData(
          statusPayload,
          ticketPayload.tickets,
          workflowPayload.workflows,
          agentPayload.agents,
          activityPayload.events,
        ),
      )
      persistBoardSnapshot(projectId)
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
    ticketBoardToolbarStore.activateProject(appStore.currentProject?.id ?? null)
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const statusVersion = statusSync.version
    if (projectId) {
      syncProjectBoardCacheStatusVersion(projectId, statusVersion)
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
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

    const cachedBoard = readProjectBoardCache(projectId)
    if (cachedBoard) {
      applyBoardSnapshot(cachedBoard.snapshot)
      loading = false
      error = ''
      if (cachedBoard.dirty) {
        void loadBoard(projectId, 'background')
      }
    } else {
      void loadBoard(projectId, 'initial')
    }

    const disconnectProjectEvents = subscribeProjectEvents(projectId, () => {
      markProjectBoardCacheDirty(projectId)
      requestReload(projectId)
    })

    return () => {
      if (activeProjectId === projectId) {
        activeProjectId = null
      }
      disconnectProjectEvents()
    }
  })

  function handleTicketClick(ticket: BoardTicket) {
    appStore.openRightPanel({ type: 'ticket', id: ticket.id })
  }

  function handleTicketDragStart(ticket: BoardTicket) {
    startTicketDrag(controllerState, ticket)
  }

  function handleTicketDragEnd() {
    endTicketDrag(controllerState)
  }

  function handleTicketDragOverColumn(columnId: string) {
    dragOverTicketColumn(controllerState, columnId)
  }

  async function handlePriorityChange(ticketId: string, priority: string) {
    await runPriorityChange(controllerState, ticketId, priority)
  }

  async function handleTicketDrop(ticketId: string, targetColumnId: string) {
    await dropTicket(controllerState, ticketId, targetColumnId)
  }

  async function handleColumnAction(columnId: string, action: string) {
    await runColumnAction(controllerState, columnId, action)
  }

  return {
    get loading() { return loading },
    get error() { return error },
    get allColumns() { return allColumns },
    get allStatuses() { return allStatuses },
    get workflows() { return workflows },
    get agentOptions() { return agentOptions },
    get draggingTicketId() { return draggingTicketId },
    get dropColumnId() { return dropColumnId },
    get filteredColumns() { return filteredColumns },
    get filteredGroups() { return filteredGroups },
    get hiddenColumns() { return hiddenColumns },
    handleTicketClick,
    handleTicketDragStart,
    handleTicketDragEnd,
    handleTicketDragOverColumn,
    handleTicketDrop,
    handleStatusChange: handleTicketDrop,
    handlePriorityChange,
    handleColumnAction,
  }
}
