import type { StatusPayload, TicketStatus } from '$lib/api/contracts'
import type { BoardColumn, BoardGroup, BoardTicket } from './types'

export function buildBoardGroups(
  statusPayload: Pick<StatusPayload, 'statuses'>,
  ticketsByStatusId: Map<string, BoardTicket[]>,
): BoardGroup[] {
  const columns = statusPayload.statuses
    .slice()
    .sort(compareTicketStatuses)
    .map((status) => ({
      id: status.id,
      name: status.name,
      color: status.color || '#94a3b8',
      icon: status.icon || undefined,
      wipInfo: formatStatusWipInfo(status),
      tickets: ticketsByStatusId.get(status.id) ?? [],
    }))

  if (columns.length === 0) {
    return []
  }

  return [
    {
      id: 'board',
      name: 'Board',
      columns,
    },
  ]
}

export function projectBoardGroups(
  groups: BoardGroup[],
  columns: BoardColumn[],
  options?: { hideEmpty?: boolean },
): BoardGroup[] {
  const columnsById = new Map(columns.map((column) => [column.id, column]))

  return groups.map((group) => ({
    ...group,
    columns: group.columns
      .map((column) => columnsById.get(column.id))
      .filter(isDefined)
      .filter((column) => !options?.hideEmpty || column.tickets.length > 0),
  }))
}

export function compareTicketStatuses(left: TicketStatus, right: TicketStatus) {
  const stageOrderDiff = stageOrder(left.stage) - stageOrder(right.stage)
  if (stageOrderDiff !== 0) {
    return stageOrderDiff
  }
  if (left.position !== right.position) {
    return left.position - right.position
  }
  return left.name.localeCompare(right.name)
}

function formatStatusWipInfo(status: TicketStatus) {
  if (!status.max_active_runs) {
    return undefined
  }
  return `${status.active_runs} / ${status.max_active_runs} active`
}

function isDefined<T>(value: T | undefined): value is T {
  return value !== undefined
}

function stageOrder(stage: string) {
  switch (stage) {
    case 'backlog':
      return 0
    case 'unstarted':
      return 1
    case 'started':
      return 2
    case 'completed':
      return 3
    case 'canceled':
      return 4
    default:
      return 5
  }
}
