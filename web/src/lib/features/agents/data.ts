import {
  listAgentRuns,
  listAgents,
  listMachines,
  listProviders,
  listTickets,
  listWorkflows,
} from '$lib/api/openase'
import type { AgentProvider, Machine } from '$lib/api/contracts'
import { buildAgentRows, buildAgentRunRows, buildProviderCards } from './model'
import type { AgentInstance, AgentRunInstance, ProviderConfig } from './types'

export type AgentsPageData = {
  agents: AgentInstance[]
  agentRuns: AgentRunInstance[]
  providers: ProviderConfig[]
  providerItems: AgentProvider[]
  machineItems: Machine[]
}

export async function loadAgentsPageData(
  projectId: string,
  orgId: string,
  defaultProviderId: string | null,
): Promise<AgentsPageData> {
  const [
    agentPayload,
    agentRunPayload,
    providerPayload,
    ticketPayload,
    workflowPayload,
    machinePayload,
  ] = await Promise.all([
    listAgents(projectId),
    listAgentRuns(projectId),
    listProviders(orgId),
    listTickets(projectId),
    listWorkflows(projectId),
    listMachines(orgId),
  ])

  return {
    agentRuns: buildAgentRunRows(
      providerPayload.providers,
      ticketPayload.tickets,
      workflowPayload.workflows,
      agentPayload.agents,
      agentRunPayload.agent_runs,
    ),
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
