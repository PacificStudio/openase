import type { ActivityEvent, Agent, Project, Ticket } from '$lib/api/contracts'
import { isActivityExceptionEvent } from '$lib/features/activity'
import type {
  ActivityItem,
  DashboardStats,
  DashboardUsageLeader,
  ExceptionItem,
  ProjectSummary,
} from './types'

export function buildDashboardStats(
  agents: Agent[],
  tickets: Ticket[],
  options?: { ticketSpendToday?: number },
  now = new Date(),
): DashboardStats {
  const activeTickets = tickets.filter((ticket) => !isTerminalStatus(ticket.status_name))
  const runningAgents = agents.filter((agent) => agent.runtime?.status === 'running').length
  const ticketSpendTotal = tickets.reduce((sum, ticket) => sum + ticket.cost_amount, 0)
  const ticketInputTokens = tickets.reduce((sum, ticket) => sum + ticket.cost_tokens_input, 0)
  const ticketOutputTokens = tickets.reduce((sum, ticket) => sum + ticket.cost_tokens_output, 0)
  const agentLifetimeTokens = agents.reduce((sum, agent) => sum + agent.total_tokens_used, 0)
  const todayStart = new Date(now)
  todayStart.setHours(0, 0, 0, 0)
  const todayTickets = tickets.filter((ticket) => new Date(ticket.created_at) >= todayStart)

  return {
    runningAgents,
    activeTickets: activeTickets.length,
    totalTickets: tickets.length,
    pendingApprovals: 0,
    ticketSpendToday: options?.ticketSpendToday ?? 0,
    ticketSpendTotal,
    ticketsCreatedToday: todayTickets.length,
    ticketsCompletedToday: todayTickets.filter((ticket) => isTerminalStatus(ticket.status_name))
      .length,
    ticketInputTokens,
    ticketOutputTokens,
    agentLifetimeTokens,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  }
}

export function buildProjectSummary(
  project: Project,
  stats: Pick<DashboardStats, 'runningAgents' | 'activeTickets'>,
  lastActivity: string | null,
): ProjectSummary {
  return {
    id: project.id,
    name: project.name,
    description: project.description,
    status: project.status,
    activeAgents: stats.runningAgents,
    activeTickets: stats.activeTickets,
    lastActivity,
  }
}

export function shouldShowProjectOnboarding(params: {
  dismissed: boolean
  loading: boolean
  stats: Pick<DashboardStats, 'totalTickets' | 'runningAgents'>
  projectId?: string | null
  orgId?: string | null
}) {
  return (
    !params.dismissed &&
    !params.loading &&
    params.stats.totalTickets === 0 &&
    params.stats.runningAgents === 0 &&
    Boolean(params.projectId) &&
    Boolean(params.orgId)
  )
}

export function buildActivityItems(events: ActivityEvent[]): ActivityItem[] {
  return events.slice(0, 6).map((event) => ({
    id: event.id,
    type: event.event_type,
    message: event.message,
    timestamp: event.created_at,
    ticketIdentifier: undefined,
    agentName: agentNameFromMetadata(event.metadata),
  }))
}

export function buildExceptionItems(events: ActivityEvent[]): ExceptionItem[] {
  return events
    .filter((event) => isExceptionEvent(event.event_type))
    .slice(0, 4)
    .map((event) => ({
      id: event.id,
      type: normalizeExceptionType(event.event_type),
      message: event.message,
      timestamp: event.created_at,
    }))
}

export function findTopCostTicket(tickets: Ticket[]): DashboardUsageLeader | null {
  const leader = tickets.reduce<Ticket | null>((current, ticket) => {
    if (ticket.cost_amount <= 0) return current
    if (!current || ticket.cost_amount > current.cost_amount) {
      return ticket
    }
    return current
  }, null)

  return leader ? { name: leader.identifier, value: leader.cost_amount } : null
}

export function findTopTokenAgent(agents: Agent[]): DashboardUsageLeader | null {
  const leader = agents.reduce<Agent | null>((current, agent) => {
    if (agent.total_tokens_used <= 0) return current
    if (!current || agent.total_tokens_used > current.total_tokens_used) {
      return agent
    }
    return current
  }, null)

  return leader ? { name: leader.name, value: leader.total_tokens_used } : null
}

function isTerminalStatus(statusName: string) {
  const value = statusName.toLowerCase()
  return value === 'done' || value === 'cancelled' || value === 'archived'
}

function isExceptionEvent(eventType: string) {
  return isActivityExceptionEvent(eventType)
}

function normalizeExceptionType(eventType: string): ExceptionItem['type'] {
  if (
    eventType === 'hook.failed' ||
    eventType === 'ticket.budget_exhausted' ||
    eventType === 'agent.failed'
  ) {
    return eventType
  }

  return 'ticket.retry_paused'
}

function agentNameFromMetadata(metadata: Record<string, unknown>) {
  const value = metadata.agent_name
  return typeof value === 'string' ? value : undefined
}
