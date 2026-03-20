import { listAgents, listProviders, listTickets } from '$lib/api/openase'
import type { AgentProvider } from '$lib/api/contracts'
import { buildAgentRows, buildProviderCards } from './model'
import type { AgentInstance, ProviderConfig } from './types'

export type AgentsPageData = {
  agents: AgentInstance[]
  providers: ProviderConfig[]
  providerItems: AgentProvider[]
}

export async function loadAgentsPageData(
  projectId: string,
  orgId: string,
  defaultProviderId: string | null,
): Promise<AgentsPageData> {
  const [agentPayload, providerPayload, ticketPayload] = await Promise.all([
    listAgents(projectId),
    listProviders(orgId),
    listTickets(projectId),
  ])

  return {
    providerItems: providerPayload.providers,
    providers: buildProviderCards(
      providerPayload.providers,
      agentPayload.agents,
      defaultProviderId,
    ),
    agents: buildAgentRows(providerPayload.providers, ticketPayload.tickets, agentPayload.agents),
  }
}
