export type BoardColumn = {
  id: string
  name: string
  color: string
  icon?: string
  tickets: BoardTicket[]
  wipInfo?: string
}

export type BoardTicket = {
  id: string
  statusId: string
  identifier: string
  title: string
  priority: 'urgent' | 'high' | 'medium' | 'low'
  workflowType?: string
  agentName?: string
  anomaly?: 'retry' | 'hook_failed' | 'awaiting_approval' | 'budget_exhausted'
  updatedAt: string
  isMoving?: boolean
  labels?: string[]
}

export type BoardFilter = {
  workflow?: string
  agent?: string
  priority?: string
  anomalyOnly?: boolean
  search?: string
}
