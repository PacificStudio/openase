import type { BoardFilterPriority, BoardPriority } from './priority'

export type BoardExternalLink = {
  id: string
  type: string
  url: string
  externalId: string
  title?: string
  status?: string
  relation: string
}

export type BoardColumn = {
  id: string
  name: string
  color: string
  icon?: string
  tickets: BoardTicket[]
  wipInfo?: string
  pickupWorkflows?: { id: string; name: string; type: string }[]
}

export type BoardGroup = {
  id: string
  name: string
  wipInfo?: string
  columns: BoardColumn[]
}

export type BoardTicket = {
  id: string
  archived: boolean
  statusId: string
  statusName: string
  statusColor: string
  stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
  identifier: string
  title: string
  priority: BoardPriority
  workflowType?: string
  agentName?: string
  anomaly?: 'retry' | 'awaiting_approval' | 'budget_exhausted'
  runtimePhase?: 'none' | 'launching' | 'ready' | 'executing' | 'failed'
  lastError?: string
  updatedAt: string
  isMoving?: boolean
  labels?: string[]
  isBlocked?: boolean
  externalLinks?: BoardExternalLink[]
  pullRequestURLs?: string[]
}

export type BoardStatusOption = {
  id: string
  name: string
  color: string
  stage: BoardTicket['stage']
  position: number
  maxActiveRuns: number | null
}

export type HiddenColumn = {
  id: string
  name: string
  color: string
  ticketCount: number
}

export type BoardFilter = {
  workflow?: string
  agent?: string
  priority?: BoardFilterPriority
  anomalyOnly?: boolean
  search?: string
}
