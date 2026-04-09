import { getOrganizationSummary } from '$lib/api/openase'
import type { OrganizationSummaryResponse } from '$lib/api/contracts'
import type { DashboardStats } from './types'

export type ProjectMetrics = {
  runningAgents: number
  activeTickets: number
  todayCost: number
  lastActivity: string | null
}

export const emptyOrganizationDashboardStats: DashboardStats = {
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

export function mapOrganizationDashboardSummary(payload: OrganizationSummaryResponse): {
  activeProjectCount: number
  orgStats: DashboardStats
  projectMetrics: Record<string, ProjectMetrics>
} {
  return {
    activeProjectCount: payload.organization?.active_project_count ?? 0,
    orgStats: {
      ...emptyOrganizationDashboardStats,
      runningAgents: payload.organization?.running_agents ?? 0,
      activeTickets: payload.organization?.active_tickets ?? 0,
      ticketSpendToday: payload.organization?.today_cost ?? 0,
      agentLifetimeTokens: payload.organization?.total_tokens ?? 0,
    },
    projectMetrics: Object.fromEntries(
      (payload.projects ?? []).map((project) => [
        project.project_id ?? '',
        {
          runningAgents: project.running_agents ?? 0,
          activeTickets: project.active_tickets ?? 0,
          todayCost: project.today_cost ?? 0,
          lastActivity: project.last_activity_at ?? null,
        },
      ]),
    ),
  }
}

export async function loadOrganizationDashboardSummary(
  orgId: string,
  opts?: { signal?: AbortSignal },
) {
  return mapOrganizationDashboardSummary(await getOrganizationSummary(orgId, opts))
}
