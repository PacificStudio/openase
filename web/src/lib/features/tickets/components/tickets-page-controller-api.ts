import type { BoardColumnType, BoardStatusOption } from '$lib/features/board'
import type { BoardGroupType, BoardTicket, HiddenColumn } from '$lib/features/board'

type TicketsPageControllerApiInput = {
  getLoading: () => boolean
  getError: () => string
  getAllColumns: () => BoardColumnType[]
  getAllStatuses: () => BoardStatusOption[]
  getWorkflows: () => string[]
  getAgentOptions: () => string[]
  getDraggingTicketId: () => string | null
  getDropColumnId: () => string | null
  getFilteredColumns: () => BoardColumnType[]
  getFilteredGroups: () => BoardGroupType[]
  getHiddenColumns: () => HiddenColumn[]
  handleTicketClick: (ticket: BoardTicket) => void
  handleTicketDragStart: (ticket: BoardTicket) => void
  handleTicketDragEnd: () => void
  handleTicketDragOverColumn: (columnId: string) => void
  handleTicketDrop: (ticketId: string, targetColumnId: string) => Promise<void>
  handlePriorityChange: (ticketId: string, priority: string) => Promise<void>
  handleColumnAction: (columnId: string, action: string) => Promise<void>
}

export function createTicketsPageControllerApi(input: TicketsPageControllerApiInput) {
  return {
    get loading() {
      return input.getLoading()
    },
    get error() {
      return input.getError()
    },
    get allColumns() {
      return input.getAllColumns()
    },
    get allStatuses() {
      return input.getAllStatuses()
    },
    get workflows() {
      return input.getWorkflows()
    },
    get agentOptions() {
      return input.getAgentOptions()
    },
    get draggingTicketId() {
      return input.getDraggingTicketId()
    },
    get dropColumnId() {
      return input.getDropColumnId()
    },
    get filteredColumns() {
      return input.getFilteredColumns()
    },
    get filteredGroups() {
      return input.getFilteredGroups()
    },
    get hiddenColumns() {
      return input.getHiddenColumns()
    },
    handleTicketClick: input.handleTicketClick,
    handleTicketDragStart: input.handleTicketDragStart,
    handleTicketDragEnd: input.handleTicketDragEnd,
    handleTicketDragOverColumn: input.handleTicketDragOverColumn,
    handleTicketDrop: input.handleTicketDrop,
    handleStatusChange: input.handleTicketDrop,
    handlePriorityChange: input.handlePriorityChange,
    handleColumnAction: input.handleColumnAction,
  }
}
