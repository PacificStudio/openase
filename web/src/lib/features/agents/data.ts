import type { AgentPayload, AgentProvider, Ticket } from '$lib/api/contracts'
import type { AgentInstance, ProviderConfig } from './types'

export function buildProviderCards(
  providerItems: AgentProvider[],
  agentItems: AgentPayload['agents'],
  defaultProviderId?: string | null,
): ProviderConfig[] {
  return providerItems.map((provider) => ({
    id: provider.id,
    name: provider.name,
    adapterType: provider.adapter_type,
    modelName: provider.model_name,
    agentCount: agentItems.filter((agent) => agent.provider_id === provider.id).length,
    isDefault: defaultProviderId === provider.id,
  }))
}

export function buildAgentRows(
  providerItems: AgentProvider[],
  ticketItems: Ticket[],
  agentItems: AgentPayload['agents'],
): AgentInstance[] {
  const ticketMap = new Map(ticketItems.map((ticket) => [ticket.id, ticket]))
  const providerMap = new Map(providerItems.map((provider) => [provider.id, provider]))

  return agentItems.map((agent) => {
    const provider = providerMap.get(agent.provider_id)
    const currentTicket = agent.current_ticket_id ? ticketMap.get(agent.current_ticket_id) : null

    return {
      id: agent.id,
      name: agent.name,
      providerName: provider?.name ?? 'Unknown provider',
      modelName: provider?.model_name ?? 'Unknown model',
      status: normalizeAgentStatus(agent.status),
      runtimePhase: normalizeRuntimePhase(agent.runtime_phase),
      currentTicket: currentTicket
        ? {
            id: currentTicket.id,
            identifier: currentTicket.identifier,
            title: currentTicket.title,
          }
        : undefined,
      lastHeartbeat: agent.last_heartbeat_at,
      todayCompleted: agent.total_tickets_completed,
      todayCost: 0,
      capabilities: agent.capabilities,
    }
  })
}

function normalizeAgentStatus(status: string): AgentInstance['status'] {
  if (
    status === 'idle' ||
    status === 'claimed' ||
    status === 'running' ||
    status === 'failed' ||
    status === 'terminated'
  ) {
    return status
  }

  return status === 'active' ? 'running' : 'idle'
}

function normalizeRuntimePhase(runtimePhase: string): AgentInstance['runtimePhase'] {
  if (
    runtimePhase === 'none' ||
    runtimePhase === 'launching' ||
    runtimePhase === 'ready' ||
    runtimePhase === 'failed'
  ) {
    return runtimePhase
  }

  return 'none'
}
