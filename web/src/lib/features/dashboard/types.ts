export type ProjectHealth = 'healthy' | 'warning' | 'blocked'

export type ProjectSummary = {
  id: string
  name: string
  health: ProjectHealth
  activeAgents: number
  activeTickets: number
  lastActivity: string
}

export type DashboardStats = {
  runningAgents: number
  activeTickets: number
  pendingApprovals: number
  todayCost: number
  weekCost: number
  ticketsCreatedToday: number
  ticketsCompletedToday: number
  avgCycleMinutes: number
  prMergeRate: number
}

export type ExceptionItem = {
  id: string
  type: 'hook_failed' | 'budget_alert' | 'agent_stalled' | 'retry_paused'
  message: string
  ticketIdentifier?: string
  timestamp: string
}

export type ActivityItem = {
  id: string
  type: string
  message: string
  timestamp: string
  ticketIdentifier?: string
  agentName?: string
}
