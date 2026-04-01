export type BoardColumn = {
  id: string
  name: string
  color: string
  icon?: string
  tickets: BoardTicket[]
  wipInfo?: string
}

export type BoardGroup = {
  id: string
  name: string
  wipInfo?: string
  columns: BoardColumn[]
}

export type BoardTicket = {
  id: string
  statusId: string
  statusName: string
  statusColor: string
  stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
  identifier: string
  title: string
  priority: 'urgent' | 'high' | 'medium' | 'low'
  workflowType?: string
  agentName?: string
  anomaly?: 'retry' | 'awaiting_approval' | 'budget_exhausted'
  runtimePhase?: 'none' | 'launching' | 'ready' | 'executing' | 'failed'
  lastError?: string
  updatedAt: string
  isMoving?: boolean
  labels?: string[]
  isBlocked?: boolean
}

export type BoardStatusOption = {
  id: string
  name: string
  color: string
  stage: BoardTicket['stage']
  position: number
}

export type BoardFilter = {
  workflow?: string
  agent?: string
  priority?: string
  anomalyOnly?: boolean
  search?: string
}
