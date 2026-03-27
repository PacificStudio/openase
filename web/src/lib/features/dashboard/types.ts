import type {
  HRAdvisorRecommendation,
  HRAdvisorStaffing,
  HRAdvisorSummary,
  SystemMemorySnapshot,
} from '$lib/api/contracts'

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
  newTicketsTodayCost: number
  projectCost: number
  ticketsCreatedToday: number
  ticketsCompletedToday: number
  ticketInputTokens: number
  ticketOutputTokens: number
  totalAgentTokens: number
  avgCycleMinutes: number
  prMergeRate: number
}

export type DashboardUsageLeader = {
  name: string
  value: number
}

export type MemorySnapshot = SystemMemorySnapshot

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

export type HRAdvisorSnapshot = {
  summary: HRAdvisorSummary
  staffing: HRAdvisorStaffing
  recommendations: HRAdvisorRecommendation[]
}
