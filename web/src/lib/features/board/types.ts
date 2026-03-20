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
  identifier: string
  title: string
  priority: 'urgent' | 'high' | 'medium' | 'low'
  workflowType?: string
  agentName?: string
  prCount?: number
  prStatus?: string
  anomaly?: 'retry' | 'hook_failed' | 'awaiting_approval' | 'budget_exhausted'
  updatedAt: string
  labels?: string[]
}

export type BoardFilter = {
  workflow?: string
  agent?: string
  priority?: string
  anomalyOnly?: boolean
  search?: string
}
