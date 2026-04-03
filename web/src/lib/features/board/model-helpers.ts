import type { ActivityEvent, Agent, Ticket } from '$lib/api/contracts'
import type { BoardTicket } from './types'

export type AgentRuntimeInfo = { runtimePhase: string; lastError: string }

export function buildTicketRuntimeById(agents: Agent[], activity: ActivityEvent[]) {
  const agentNameById = new Map(agents.map((agent) => [agent.id, agent.name]))
  const agentRuntimeByTicketId = buildAgentRuntimeByTicketId(agents)
  const runtimeByTicketId = new Map<
    string,
    { agentName: string; updatedAt: string; timestamp: number }
  >()

  for (const event of activity) {
    if (!event.ticket_id) continue

    const agentName = getActivityAgentName(event, agentNameById)
    if (!agentName) continue

    const timestamp = Date.parse(event.created_at)
    const current = runtimeByTicketId.get(event.ticket_id)
    if (current && !Number.isNaN(timestamp) && current.timestamp > timestamp) continue

    runtimeByTicketId.set(event.ticket_id, {
      agentName,
      updatedAt: event.created_at,
      timestamp: Number.isNaN(timestamp) ? 0 : timestamp,
    })
  }

  return { runtimeByTicketId, agentRuntimeByTicketId }
}

export function normalizePriority(priority: string): BoardTicket['priority'] {
  if (priority === 'urgent' || priority === 'high' || priority === 'medium' || priority === 'low') {
    return priority
  }
  return ''
}

export function inferAnomaly(
  ticket: Pick<Ticket, 'budget_usd' | 'cost_amount' | 'consecutive_errors' | 'retry_paused'>,
): BoardTicket['anomaly'] | undefined {
  if (ticket.retry_paused) return 'retry'
  if (ticket.budget_usd > 0 && ticket.cost_amount >= ticket.budget_usd) return 'budget_exhausted'
  return undefined
}

export function normalizeRuntimePhase(
  phase: string | undefined,
): BoardTicket['runtimePhase'] | undefined {
  if (phase === 'launching' || phase === 'ready' || phase === 'executing' || phase === 'failed') {
    return phase
  }
  return undefined
}

export function isDefined<T>(value: T | undefined): value is T {
  return value !== undefined
}

function buildAgentRuntimeByTicketId(agents: Agent[]): Map<string, AgentRuntimeInfo> {
  const map = new Map<string, AgentRuntimeInfo>()
  for (const agent of agents) {
    const rt = agent.runtime
    if (!rt?.current_ticket_id) continue
    map.set(rt.current_ticket_id, {
      runtimePhase: rt.runtime_phase ?? 'none',
      lastError: rt.last_error ?? '',
    })
  }
  return map
}

function getActivityAgentName(
  event: Pick<ActivityEvent, 'agent_id' | 'metadata'>,
  agentNameById: Map<string, string>,
) {
  const metadataAgentName = event.metadata.agent_name
  if (typeof metadataAgentName === 'string' && metadataAgentName.trim() !== '') {
    return metadataAgentName
  }

  return event.agent_id ? agentNameById.get(event.agent_id) : undefined
}
