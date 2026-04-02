import type { TicketsPageControllerActionsState } from './tickets-page-controller-actions'
import type { BoardColumnType, BoardStatusOption, PendingTicketMove } from '$lib/features/board'

type TicketsPageControllerStateInput = {
  getAllColumns: () => BoardColumnType[]
  setAllColumns: (value: BoardColumnType[]) => void
  getAllStatuses: () => BoardStatusOption[]
  pendingMoveByTicket: Map<string, PendingTicketMove>
  getDraggingTicketId: () => string | null
  setDraggingTicketId: (value: string | null) => void
  getDropColumnId: () => string | null
  setDropColumnId: (value: string | null) => void
  persistBoardSnapshot: (projectId: string) => void
  requestReload: (projectId: string) => void
}

export function createTicketsPageControllerState(
  input: TicketsPageControllerStateInput,
): TicketsPageControllerActionsState {
  return {
    get allColumns() {
      return input.getAllColumns()
    },
    set allColumns(value) {
      input.setAllColumns(value)
    },
    get allStatuses() {
      return input.getAllStatuses()
    },
    get pendingMoveByTicket() {
      return input.pendingMoveByTicket
    },
    get draggingTicketId() {
      return input.getDraggingTicketId()
    },
    set draggingTicketId(value) {
      input.setDraggingTicketId(value)
    },
    get dropColumnId() {
      return input.getDropColumnId()
    },
    set dropColumnId(value) {
      input.setDropColumnId(value)
    },
    persistBoardSnapshot: input.persistBoardSnapshot,
    requestReload: input.requestReload,
  }
}
