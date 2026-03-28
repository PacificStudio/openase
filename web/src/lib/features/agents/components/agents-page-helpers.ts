import { currentHashSelection } from '$lib/utils/hash-state'
import type { AgentProvider, Machine } from '$lib/api/contracts'
import type { AgentsPageData } from '../data'
import type { AgentInstance, AgentRunInstance, ProviderConfig } from '../types'

export const agentPageTabs = ['runtime', 'definitions', 'providers'] as const

export type AgentPageTab = (typeof agentPageTabs)[number]

export type AgentsPageStateSnapshot = {
  agents: AgentInstance[]
  agentRuns: AgentRunInstance[]
  providers: ProviderConfig[]
  providerItems: AgentProvider[]
  machineItems: Machine[]
}

export function resolveAgentPageTab(): AgentPageTab {
  return currentHashSelection(agentPageTabs, 'runtime')
}

export function mapAgentsPageData(data: AgentsPageData): AgentsPageStateSnapshot {
  return {
    providerItems: data.providerItems,
    machineItems: data.machineItems,
    providers: data.providers,
    agents: data.agents,
    agentRuns: data.agentRuns,
  }
}
