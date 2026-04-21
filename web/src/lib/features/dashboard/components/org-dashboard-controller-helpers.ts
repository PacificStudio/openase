import { getHRAdvisor } from '$lib/api/openase'
import type { ProjectDashboardRefreshSection } from '$lib/features/project-events'
import type { DashboardStats, HRAdvisorSnapshot } from '../types'

export const systemDashboardRefreshIntervalMs = 10_000

export const emptyDashboardStats: DashboardStats = {
  runningAgents: 0,
  activeTickets: 0,
  totalTickets: 0,
  pendingApprovals: 0,
  ticketSpendToday: 0,
  ticketSpendTotal: 0,
  ticketsCreatedToday: 0,
  ticketsCompletedToday: 0,
  ticketInputTokens: 0,
  ticketOutputTokens: 0,
  agentLifetimeTokens: 0,
  avgCycleMinutes: 0,
  prMergeRate: 0,
}

export type DashboardSection = ProjectDashboardRefreshSection | 'memory'

export function mergeDashboardSections(
  current: DashboardSection[],
  incoming: Iterable<DashboardSection>,
): DashboardSection[] {
  const merged = [...current]
  for (const section of incoming) {
    if (!merged.includes(section)) merged.push(section)
  }
  return merged
}

export const toAdvisorSnapshot = (
  payload: Awaited<ReturnType<typeof getHRAdvisor>> | null,
): HRAdvisorSnapshot | null =>
  payload
    ? {
        summary: payload.summary,
        staffing: payload.staffing,
        recommendations: payload.recommendations,
      }
    : null
