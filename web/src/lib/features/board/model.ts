import type { ActivityEvent, Agent, StatusPayload, Ticket, Workflow } from '$lib/api/contracts'
import {
  buildTicketRuntimeById,
  inferAnomaly,
  isDefined,
  normalizePriority,
  normalizeRuntimePhase,
  type AgentRuntimeInfo,
} from './model-helpers'
import { buildBoardGroups, compareTicketStatuses } from './grouping'
import type { BoardColumn, BoardFilter, BoardGroup, BoardStatusOption, BoardTicket } from './types'

export { projectBoardGroups } from './grouping'

export type PendingTicketMove = {
  fromColumnId: string
  fromIndex: number
}

export type BoardData = {
  columns: BoardColumn[]
  groups: BoardGroup[]
  statusOptions: BoardStatusOption[]
  workflowTypes: string[]
  agentOptions: string[]
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

export function buildBoardData(
  statusPayload: Pick<StatusPayload, 'statuses'>,
  tickets: Ticket[],
  workflows: Workflow[],
  agents: Agent[],
  activity: ActivityEvent[],
): BoardData {
  const workflowTypeById = new Map(workflows.map((workflow) => [workflow.id, workflow.type]))
  const { runtimeByTicketId, agentRuntimeByTicketId } = buildTicketRuntimeById(agents, activity)
  const statusMap = new Map(
    statusPayload.statuses.map((s) => [
      s.id,
      { name: s.name, color: s.color || '#94a3b8', stage: s.stage },
    ]),
  )
  const terminalStatusIds = new Set(
    statusPayload.statuses
      .filter((s) => s.stage === 'completed' || s.stage === 'canceled')
      .map((s) => s.id),
  )
  const ticketsByStatusId = buildBoardTicketsByStatusId(
    tickets,
    workflowTypeById,
    runtimeByTicketId,
    agentRuntimeByTicketId,
    terminalStatusIds,
    statusMap,
  )
  const groups = buildBoardGroups(statusPayload, ticketsByStatusId)
  const columns = groups.flatMap((group) => group.columns)

  const statusOptions: BoardStatusOption[] = statusPayload.statuses
    .slice()
    .sort(compareTicketStatuses)
    .map((s) => ({
      id: s.id,
      name: s.name,
      color: s.color || '#94a3b8',
      stage: (s.stage || 'unstarted') as BoardStatusOption['stage'],
      position: s.position,
      maxActiveRuns: typeof s.max_active_runs === 'number' ? s.max_active_runs : null,
    }))

  return {
    groups,
    statusOptions,
    workflowTypes: Array.from(new Set(workflows.map((workflow) => workflow.type))),
    agentOptions: Array.from(
      new Set(
        columns.flatMap((column) =>
          column.tickets.map((ticket) => ticket.agentName).filter(isDefined),
        ),
      ),
    ).sort((left, right) => left.localeCompare(right)),
    columns,
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
      return { ...column, tickets: insertTicket(nextTickets, movedTicket, options.targetIndex) }
    }

    if (column.id === location.columnId) {
      return {
        ...column,
        tickets: column.tickets.filter((ticket) => ticket.id !== ticketId),
      }
    }

    if (column.id === targetColumnId) {
      return { ...column, tickets: insertTicket(column.tickets, movedTicket, options.targetIndex) }
    }

    return column
  })
}

function insertTicket(
  tickets: BoardTicket[],
  ticket: BoardTicket,
  targetIndex: number | undefined,
): BoardTicket[] {
  const nextTickets = tickets.slice()
  const insertIndex = clampIndex(targetIndex ?? nextTickets.length, nextTickets.length)
  nextTickets.splice(insertIndex, 0, ticket)
  return nextTickets
}

function clampIndex(index: number, length: number) {
  return Math.max(0, Math.min(index, length))
}

function buildBoardTicketsByStatusId(
  tickets: Ticket[],
  workflowTypeById: Map<string, string>,
  runtimeByTicketId: Map<string, { agentName: string; updatedAt: string; timestamp: number }>,
  agentRuntimeByTicketId: Map<string, AgentRuntimeInfo>,
  terminalStatusIds: Set<string>,
  statusMap: Map<string, { name: string; color: string; stage: string }>,
) {
  const ticketsByStatusId = new Map<string, BoardTicket[]>()

  for (const ticket of tickets) {
    if (ticket.archived) continue

    const runtime = runtimeByTicketId.get(ticket.id)
    const agentRuntime = agentRuntimeByTicketId.get(ticket.id)
    const isBlocked = ticket.dependencies.some(
      (dep) => dep.type === 'blocked_by' && !terminalStatusIds.has(dep.target.status_id),
    )
    const statusInfo = statusMap.get(ticket.status_id)
    const boardTicket: BoardTicket = {
      id: ticket.id,
      archived: ticket.archived,
      statusId: ticket.status_id,
      statusName: statusInfo?.name ?? 'Unknown',
      statusColor: statusInfo?.color ?? '#94a3b8',
      stage: (statusInfo?.stage || 'unstarted') as BoardTicket['stage'],
      identifier: ticket.identifier,
      title: ticket.title,
      priority: normalizePriority(ticket.priority),
      workflowType: ticket.workflow_id
        ? (workflowTypeById.get(ticket.workflow_id) ?? undefined)
        : undefined,
      agentName: runtime?.agentName,
      runtimePhase: normalizeRuntimePhase(agentRuntime?.runtimePhase),
      lastError: agentRuntime?.lastError || undefined,
      updatedAt: runtime?.updatedAt ?? ticket.created_at,
      labels: [],
      anomaly: inferAnomaly(ticket),
      isBlocked: isBlocked || undefined,
    }

    const current = ticketsByStatusId.get(ticket.status_id)
    if (current) {
      current.push(boardTicket)
      continue
    }

    ticketsByStatusId.set(ticket.status_id, [boardTicket])
  }

  return ticketsByStatusId
}
