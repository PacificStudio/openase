import type { Agent, Project, Ticket } from '$lib/api/contracts'
import { listAgents, listTickets } from '$lib/api/openase'
import { buildDashboardStats } from './model'
import type { DashboardStats } from './types'

export type ProjectMetrics = {
  runningAgents: number
  activeTickets: number
  todayCost: number
  lastActivity: string | null
}

export const emptyDashboardStats: DashboardStats = {
  runningAgents: 0,
  activeTickets: 0,
  pendingApprovals: 0,
  newTicketsTodayCost: 0,
  projectCost: 0,
  ticketsCreatedToday: 0,
  ticketsCompletedToday: 0,
  ticketInputTokens: 0,
  ticketOutputTokens: 0,
  totalAgentTokens: 0,
  avgCycleMinutes: 0,
  prMergeRate: 0,
}

export async function loadOrganizationMetrics(projects: Project[]) {
  const results = await Promise.all(
    projects.map(async (project) => {
      const [agentPayload, ticketPayload] = await Promise.all([
        listAgents(project.id),
        listTickets(project.id),
      ])

      return {
        projectId: project.id,
        agents: agentPayload.agents,
        tickets: ticketPayload.tickets,
      }
    }),
  )

  const allAgents: Agent[] = []
  const allTickets: Ticket[] = []
  const projectMetrics: Record<string, ProjectMetrics> = {}

  for (const { projectId, agents, tickets } of results) {
    allAgents.push(...agents)
    allTickets.push(...tickets)

    const stats = buildDashboardStats(agents, tickets)
    const latestTicket = tickets.reduce<Ticket | null>((latest, ticket) => {
      if (!latest || ticket.created_at > latest.created_at) {
        return ticket
      }

      return latest
    }, null)

    projectMetrics[projectId] = {
      runningAgents: stats.runningAgents,
      activeTickets: stats.activeTickets,
      todayCost: stats.newTicketsTodayCost,
      lastActivity: latestTicket?.created_at ?? null,
    }
  }

  return {
    projectMetrics,
    orgStats: buildDashboardStats(allAgents, allTickets),
  }
}
