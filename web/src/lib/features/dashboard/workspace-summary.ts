import { getWorkspaceSummary } from '$lib/api/openase'

export type WorkspaceOrgMetrics = {
  projectCount: number
  providerCount: number
  runningAgents: number
  activeTickets: number
  todayCost: number
}

export type WorkspaceStats = {
  runningAgents: number
  activeTickets: number
  todayCost: number
  totalTokens: number
}

export const emptyWorkspaceStats: WorkspaceStats = {
  runningAgents: 0,
  activeTickets: 0,
  todayCost: 0,
  totalTokens: 0,
}

export async function loadWorkspaceDashboardSummary(opts?: { signal?: AbortSignal }): Promise<{
  orgMetrics: Record<string, WorkspaceOrgMetrics>
  totalProjects: number
  workspaceStats: WorkspaceStats
}> {
  const payload = await getWorkspaceSummary(opts)

  return {
    orgMetrics: Object.fromEntries(
      (payload.organizations ?? []).map((organization) => [
        organization.organization_id ?? '',
        {
          projectCount: organization.project_count ?? 0,
          providerCount: organization.provider_count ?? 0,
          runningAgents: organization.running_agents ?? 0,
          activeTickets: organization.active_tickets ?? 0,
          todayCost: organization.today_cost ?? 0,
        },
      ]),
    ),
    totalProjects: payload.workspace?.project_count ?? 0,
    workspaceStats: {
      runningAgents: payload.workspace?.running_agents ?? 0,
      activeTickets: payload.workspace?.active_tickets ?? 0,
      todayCost: payload.workspace?.today_cost ?? 0,
      totalTokens: payload.workspace?.total_tokens ?? 0,
    },
  }
}
