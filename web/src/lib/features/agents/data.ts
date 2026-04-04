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
    providersPayload,
    ticketsPayload,
    workflowsPayload,
    agentsPayload,
    agentRunsPayload,
    machinesPayload,
  ] = await Promise.all([
    listProviders(orgId),
    listTickets(projectId),
    listWorkflows(projectId),
    listAgents(projectId),
    listAgentRuns(projectId),
    listMachines(orgId),
  ])

  return {
    agentRuns: buildAgentRunRows(
      providersPayload.providers,
      ticketsPayload.tickets,
      workflowsPayload.workflows,
      agentsPayload.agents,
      agentRunsPayload.agent_runs,
    ),
    providerItems: providersPayload.providers,
    machineItems: machinesPayload.machines,
    providers: buildProviderCards(
      providersPayload.providers,
      agentsPayload.agents,
      defaultProviderId,
    ),
    agents: buildAgentRows(
      providersPayload.providers,
      ticketsPayload.tickets,
      agentsPayload.agents,
    ),
  }
}
