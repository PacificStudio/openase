import type { Ticket, TicketStatus, Workflow } from '$lib/api/contracts'
import type { BoardColumn, BoardFilter, BoardTicket } from './types'

export type PendingTicketMove = {
  fromColumnId: string
  fromIndex: number
}

export function filterBoardColumns(columns: BoardColumn[], filter: BoardFilter): BoardColumn[] {
  return columns.map((column) => ({
    ...column,
    tickets: column.tickets.filter((ticket) => matchesBoardFilter(ticket, filter)),
  }))
}

function matchesBoardFilter(ticket: BoardTicket, filter: BoardFilter): boolean {
  return [
    matchesSearchFilter(ticket, filter.search),
    matchesValueFilter(ticket.workflowType, filter.workflow),
    matchesValueFilter(ticket.agentName, filter.agent),
    matchesValueFilter(ticket.priority, filter.priority),
    matchesAnomalyFilter(ticket, filter.anomalyOnly),
  ].every(Boolean)
}

function matchesSearchFilter(ticket: BoardTicket, rawSearch: string | undefined) {
  const normalizedSearch = rawSearch?.trim().toLowerCase()
  if (!normalizedSearch) return true

  return (
    ticket.title.toLowerCase().includes(normalizedSearch) ||
    ticket.identifier.toLowerCase().includes(normalizedSearch)
  )
}

function matchesValueFilter(value: string | undefined, expected: string | undefined) {
  return !expected || value === expected
}

function matchesAnomalyFilter(ticket: BoardTicket, anomalyOnly: boolean | undefined) {
  return !anomalyOnly || !!ticket.anomaly
}

export function buildBoardColumns(
  statuses: TicketStatus[],
  tickets: Ticket[],
  workflows: Workflow[],
) {
  const workflowTypeById = new Map(workflows.map((workflow) => [workflow.id, workflow.type]))

  return {
    workflowTypes: Array.from(new Set(workflows.map((workflow) => workflow.type))),
    columns: statuses
      .slice()
      .sort((left, right) => left.position - right.position)
      .map((status) => ({
        id: status.id,
        name: status.name,
        color: status.color || '#94a3b8',
        tickets: tickets
          .filter((ticket) => ticket.status_id === status.id)
          .map((ticket) => ({
            id: ticket.id,
            statusId: ticket.status_id,
            identifier: ticket.identifier,
            title: ticket.title,
            priority: normalizePriority(ticket.priority),
            workflowType: ticket.workflow_id
              ? (workflowTypeById.get(ticket.workflow_id) ?? undefined)
              : undefined,
            updatedAt: ticket.created_at,
            labels: [],
            anomaly: inferAnomaly(ticket),
          })),
      })),
  }
}

export function findTicketLocation(columns: BoardColumn[], ticketId: string) {
  for (const column of columns) {
    const index = column.tickets.findIndex((ticket) => ticket.id === ticketId)
    if (index !== -1) {
      return {
        columnId: column.id,
        index,
        ticket: column.tickets[index],
      }
    }
  }

  return null
}

export function patchTicket(
  columns: BoardColumn[],
  ticketId: string,
  update: (ticket: BoardTicket) => BoardTicket,
): BoardColumn[] {
  return columns.map((column) => ({
    ...column,
    tickets: column.tickets.map((ticket) => (ticket.id === ticketId ? update(ticket) : ticket)),
  }))
}

export function relocateTicket(
  columns: BoardColumn[],
  ticketId: string,
  targetColumnId: string,
  options: {
    targetIndex?: number
    isMoving?: boolean
    updatedAt?: string
  } = {},
): BoardColumn[] {
  const location = findTicketLocation(columns, ticketId)
  const targetColumn = columns.find((column) => column.id === targetColumnId)
  if (!location || !targetColumn) return columns

  if (location.columnId === targetColumnId && options.targetIndex === undefined) {
    return patchTicket(columns, ticketId, (ticket) => ({
      ...ticket,
      statusId: targetColumnId,
      isMoving: options.isMoving ?? ticket.isMoving,
      updatedAt: options.updatedAt ?? ticket.updatedAt,
    }))
  }

  const movedTicket: BoardTicket = {
    ...location.ticket,
    statusId: targetColumnId,
    isMoving: options.isMoving ?? false,
    updatedAt: options.updatedAt ?? location.ticket.updatedAt,
  }

  return columns.map((column) => {
    if (column.id === location.columnId && column.id === targetColumnId) {
      const nextTickets = column.tickets.slice()
      nextTickets.splice(location.index, 1)
      const targetIndex = clampIndex(options.targetIndex ?? nextTickets.length, nextTickets.length)
      nextTickets.splice(targetIndex, 0, movedTicket)
      return { ...column, tickets: nextTickets }
    }

    if (column.id === location.columnId) {
      return {
        ...column,
        tickets: column.tickets.filter((ticket) => ticket.id !== ticketId),
      }
    }

    if (column.id === targetColumnId) {
      const nextTickets = column.tickets.slice()
      const targetIndex = clampIndex(options.targetIndex ?? nextTickets.length, nextTickets.length)
      nextTickets.splice(targetIndex, 0, movedTicket)
      return { ...column, tickets: nextTickets }
    }

    return column
  })
}

function clampIndex(index: number, length: number) {
  return Math.max(0, Math.min(index, length))
}

function normalizePriority(priority: string): BoardTicket['priority'] {
  if (priority === 'urgent' || priority === 'high' || priority === 'medium' || priority === 'low') {
    return priority
  }

  return 'medium'
}

function inferAnomaly(
  ticket: Pick<Ticket, 'budget_usd' | 'cost_amount' | 'consecutive_errors' | 'retry_paused'>,
): BoardTicket['anomaly'] | undefined {
  if (ticket.retry_paused) return 'retry'
  if (ticket.consecutive_errors > 0) return 'hook_failed'
  if (ticket.budget_usd > 0 && ticket.cost_amount >= ticket.budget_usd) return 'budget_exhausted'
  return undefined
}
