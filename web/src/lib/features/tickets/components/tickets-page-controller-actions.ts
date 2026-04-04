import { ApiError } from '$lib/api/client'
import { updateStatus, updateTicket } from '$lib/api/openase'
import {
  findTicketLocation,
  patchTicket,
  relocateTicket,
  type BoardColumnType,
  type BoardStatusOption,
  type BoardTicket,
  type PendingTicketMove,
} from '$lib/features/board'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'

export type TicketsPageControllerActionsState = {
  get allColumns(): BoardColumnType[]
  set allColumns(value: BoardColumnType[])
  get allStatuses(): BoardStatusOption[]
  get pendingMoveByTicket(): Map<string, PendingTicketMove>
  get draggingTicketId(): string | null
  set draggingTicketId(value: string | null)
  get dropColumnId(): string | null
  set dropColumnId(value: string | null)
  persistBoardSnapshot(projectId: string): void
  requestReload(projectId: string): void
}

export function handleTicketDragStart(
  state: Pick<TicketsPageControllerActionsState, 'draggingTicketId' | 'dropColumnId'>,
  ticket: BoardTicket,
) {
  if (ticket.isMoving) return
  state.draggingTicketId = ticket.id
  state.dropColumnId = ticket.statusId
}

export function handleTicketDragEnd(
  state: Pick<TicketsPageControllerActionsState, 'draggingTicketId' | 'dropColumnId'>,
) {
  state.draggingTicketId = null
  state.dropColumnId = null
}

export function handleTicketDragOverColumn(
  state: Pick<TicketsPageControllerActionsState, 'draggingTicketId' | 'dropColumnId'>,
  columnId: string,
) {
  if (state.draggingTicketId) {
    state.dropColumnId = columnId
  }
}

export async function handlePriorityChange(
  state: Pick<
    TicketsPageControllerActionsState,
    'allColumns' | 'persistBoardSnapshot' | 'requestReload'
  >,
  ticketId: string,
  priority: string,
) {
  const projectId = appStore.currentProject?.id
  if (!projectId) return

  state.allColumns = patchTicket(state.allColumns, ticketId, (ticket) => ({
    ...ticket,
    priority: priority as BoardTicket['priority'],
  }))
  state.persistBoardSnapshot(projectId)

  try {
    await updateTicket(ticketId, { priority })
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to update priority.',
    )
    state.requestReload(projectId)
  }
}

export async function handleTicketDrop(
  state: TicketsPageControllerActionsState,
  ticketId: string,
  targetColumnId: string,
) {
  const projectId = appStore.currentProject?.id
  const location = findTicketLocation(state.allColumns, ticketId)
  handleTicketDragEnd(state)

  if (
    !projectId ||
    !location ||
    location.ticket.isMoving ||
    state.pendingMoveByTicket.has(ticketId) ||
    location.columnId === targetColumnId
  ) {
    return
  }

  state.pendingMoveByTicket.set(ticketId, {
    fromColumnId: location.columnId,
    fromIndex: location.index,
  })

  state.allColumns = relocateTicket(state.allColumns, ticketId, targetColumnId, {
    isMoving: true,
    updatedAt: new Date().toISOString(),
  })
  state.persistBoardSnapshot(projectId)

  try {
    await updateTicket(ticketId, { status_id: targetColumnId })
    state.allColumns = patchTicket(state.allColumns, ticketId, (ticket) => ({
      ...ticket,
      statusId: targetColumnId,
      isMoving: false,
    }))
    state.persistBoardSnapshot(projectId)
  } catch (caughtError) {
    const pendingMove = state.pendingMoveByTicket.get(ticketId)
    if (pendingMove) {
      state.allColumns = relocateTicket(state.allColumns, ticketId, pendingMove.fromColumnId, {
        targetIndex: pendingMove.fromIndex,
        isMoving: false,
      })
    }
    state.persistBoardSnapshot(projectId)
    toastStore.error(
      caughtError instanceof ApiError
        ? caughtError.detail
        : 'Failed to move ticket to the new status.',
    )
  } finally {
    state.pendingMoveByTicket.delete(ticketId)
    state.requestReload(projectId)
  }
}

async function handleColumnMove(
  state: Pick<TicketsPageControllerActionsState, 'allStatuses' | 'requestReload'>,
  statusId: string,
  direction: 'left' | 'right',
) {
  const projectId = appStore.currentProject?.id
  if (!projectId) return
  const currentStatus = state.allStatuses.find((status) => status.id === statusId)
  if (!currentStatus) return

  const stageStatuses = state.allStatuses.filter((status) => status.stage === currentStatus.stage)
  const currentIndex = stageStatuses.findIndex((status) => status.id === statusId)
  const targetIndex = direction === 'left' ? currentIndex - 1 : currentIndex + 1
  if (currentIndex === -1 || targetIndex < 0 || targetIndex >= stageStatuses.length) return

  const targetStatus = stageStatuses[targetIndex]
  if (!targetStatus) return

  try {
    await Promise.all([
      updateStatus(currentStatus.id, { position: targetStatus.position }),
      updateStatus(targetStatus.id, { position: currentStatus.position }),
    ])
    state.requestReload(projectId)
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to reorder statuses.',
    )
  }
}

async function handleColumnConcurrency(
  state: Pick<TicketsPageControllerActionsState, 'allStatuses' | 'requestReload'>,
  statusId: string,
  nextMaxActiveRuns?: number | null,
) {
  const projectId = appStore.currentProject?.id
  if (!projectId) return
  const currentStatus = state.allStatuses.find((status) => status.id === statusId)
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
    state.requestReload(projectId)
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to update concurrency.',
    )
  }
}

async function handleColumnArchiveAll(
  state: Pick<TicketsPageControllerActionsState, 'allColumns' | 'allStatuses' | 'requestReload'>,
  statusId: string,
) {
  const projectId = appStore.currentProject?.id
  if (!projectId) return

  const sourceColumn = state.allColumns.find((column) => column.id === statusId)
  if (!sourceColumn) return

  const ticketIDs = sourceColumn.tickets
    .filter((ticket) => !ticket.archived)
    .map((ticket) => ticket.id)
  if (ticketIDs.length === 0) return

  try {
    await Promise.all(ticketIDs.map((ticketID) => updateTicket(ticketID, { archived: true })))
    toastStore.success(`${ticketIDs.length} ticket${ticketIDs.length > 1 ? 's' : ''} archived.`)
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to archive tickets.',
    )
  } finally {
    state.requestReload(projectId)
  }
}

export async function handleColumnAction(
  state: Pick<TicketsPageControllerActionsState, 'allColumns' | 'allStatuses' | 'requestReload'>,
  columnId: string,
  action: string,
) {
  if (action === 'move_left') return handleColumnMove(state, columnId, 'left')
  if (action === 'move_right') return handleColumnMove(state, columnId, 'right')
  if (action === 'set_concurrency') return handleColumnConcurrency(state, columnId)
  if (action === 'clear_concurrency') return handleColumnConcurrency(state, columnId, null)
  if (action === 'archive_all') return handleColumnArchiveAll(state, columnId)
}
