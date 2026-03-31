import type { StatusPayload, TicketStatus } from '$lib/api/contracts'
import type { BoardColumn, BoardGroup, BoardTicket } from './types'

export function buildBoardGroups(
  statusPayload: Pick<StatusPayload, 'stage_groups' | 'statuses'>,
  ticketsByStatusId: Map<string, BoardTicket[]>,
): BoardGroup[] {
  return normalizeBoardStatusGroups(statusPayload).map((group) => ({
    id: group.stage?.id ? `stage:${group.stage.id}` : 'ungrouped',
    kind: group.stage ? 'stage' : 'ungrouped',
    name: group.stage?.name ?? 'Ungrouped statuses',
    description: resolveGroupDescription(group),
    wipInfo: formatStageWipInfo(group.stage),
    columns: group.statuses.map((status) => ({
      id: status.id,
      name: status.name,
      color: status.color || '#94a3b8',
      icon: status.icon || undefined,
      tickets: ticketsByStatusId.get(status.id) ?? [],
    })),
  }))
}

export function projectBoardGroups(groups: BoardGroup[], columns: BoardColumn[]): BoardGroup[] {
  const columnsById = new Map(columns.map((column) => [column.id, column]))

  return groups.map((group) => ({
    ...group,
    columns: group.columns.map((column) => columnsById.get(column.id)).filter(isDefined),
  }))
}

function normalizeBoardStatusGroups(
  statusPayload: Pick<StatusPayload, 'stage_groups' | 'statuses'>,
) {
  const sortedStatuses = statusPayload.statuses.slice().sort(compareTicketStatuses)
  if (statusPayload.stage_groups.length === 0) {
    return sortedStatuses.length === 0 ? [] : [{ stage: null, statuses: sortedStatuses }]
  }

  const coveredStatusIDs = new Set<string>()
  const groups = statusPayload.stage_groups.map((group) => {
    const statuses = group.statuses.slice().sort(compareTicketStatuses)
    for (const status of statuses) {
      coveredStatusIDs.add(status.id)
    }
    return { stage: group.stage, statuses }
  })

  const missingStatuses = sortedStatuses.filter((status) => !coveredStatusIDs.has(status.id))
  if (missingStatuses.length === 0) {
    return groups
  }

  const ungroupedIndex = groups.findIndex((group) => group.stage === null)
  if (ungroupedIndex === -1) {
    return [...groups, { stage: null, statuses: missingStatuses }]
  }

  return groups.map((group, index) =>
    index === ungroupedIndex
      ? { ...group, statuses: [...group.statuses, ...missingStatuses].sort(compareTicketStatuses) }
      : group,
  )
}

function compareTicketStatuses(left: TicketStatus, right: TicketStatus) {
  if (left.position !== right.position) {
    return left.position - right.position
  }
  return left.name.localeCompare(right.name)
}

function resolveGroupDescription(group: StatusPayload['stage_groups'][number]) {
  const description = group.stage?.description?.trim()
  if (description) {
    return description
  }
  if (group.stage) {
    return undefined
  }
  return 'Statuses without a stage always render after staged groups.'
}

function formatStageWipInfo(stage: StatusPayload['stage_groups'][number]['stage']) {
  if (!stage?.max_active_runs) {
    return undefined
  }
  return `${stage.active_runs} / ${stage.max_active_runs} active`
}

function isDefined<T>(value: T | undefined): value is T {
  return value !== undefined
}
