import type {
  HRAdvisorRecommendation,
  HRAdvisorStaffing,
  HRAdvisorSummary,
  Project,
  SystemMemorySnapshot,
} from '$lib/api/contracts'

export type ProjectStatus = Project['status']

export type ProjectSummary = {
  id: string
  name: string
  description: string
  status: ProjectStatus
  activeAgents: number
  activeTickets: number
  lastActivity: string | null
}

export type DashboardStats = {
  runningAgents: number
  activeTickets: number
  pendingApprovals: number
  ticketSpendToday: number
  ticketSpendTotal: number
  ticketsCreatedToday: number
  ticketsCompletedToday: number
  ticketInputTokens: number
  ticketOutputTokens: number
  agentLifetimeTokens: number
  avgCycleMinutes: number
  prMergeRate: number
}

export type DashboardUsageLeader = {
  name: string
  value: number
}

export type OrganizationTokenUsageRange = 7 | 30 | 90 | 365

export type OrganizationTokenUsageDayPoint = {
  date: string
  dayLabel: string
  shortLabel: string
  inputTokens: number
  outputTokens: number
  cachedInputTokens: number
  reasoningTokens: number
  totalTokens: number
  finalizedRunCount: number
  intensity: 0 | 1 | 2 | 3 | 4
}

export type OrganizationTokenUsagePeak = {
  date: string
  dayLabel: string
  totalTokens: number
}

export type OrganizationTokenUsageAnalytics = {
  rangeDays: OrganizationTokenUsageRange
  days: OrganizationTokenUsageDayPoint[]
  calendarCells: Array<OrganizationTokenUsageDayPoint | null>
  totalTokens: number
  avgDailyTokens: number
  totalRuns: number
  peakDay: OrganizationTokenUsagePeak | null
  maxDailyTokens: number
}

export type MemorySnapshot = SystemMemorySnapshot

export type ExceptionItem = {
  id: string
  type: 'hook.failed' | 'ticket.budget_exhausted' | 'agent.failed' | 'ticket.retry_paused'
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
