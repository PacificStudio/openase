import { listAgents, listMachines, listProviders, listTickets } from '$lib/api/openase'
import type { AgentProvider, Machine } from '$lib/api/contracts'
import { buildAgentRows, buildProviderCards } from './model'
import type { AgentInstance, ProviderConfig } from './types'

export type AgentsPageData = {
  agents: AgentInstance[]
  providers: ProviderConfig[]
  providerItems: AgentProvider[]
  machineItems: Machine[]
}

export async function loadAgentsPageData(
  projectId: string,
  orgId: string,
  defaultProviderId: string | null,
): Promise<AgentsPageData> {
  const [agentPayload, providerPayload, ticketPayload, machinePayload] = await Promise.all([
    listAgents(projectId),
    listProviders(orgId),
    listTickets(projectId),
    listMachines(orgId),
  ])

  return {
    providerItems: providerPayload.providers,
    machineItems: machinePayload.machines,
    providers: buildProviderCards(
      providerPayload.providers,
      agentPayload.agents,
      defaultProviderId,
    ),
    agents: buildAgentRows(providerPayload.providers, ticketPayload.tickets, agentPayload.agents),
  }
}
