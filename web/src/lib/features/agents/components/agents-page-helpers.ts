import type { AgentProvider } from '$lib/api/contracts'
import type { AgentsPageData } from '../data'
import type { AgentInstance, AgentRunInstance } from '../types'

export type AgentsPageStateSnapshot = {
  agents: AgentInstance[]
  agentRuns: AgentRunInstance[]
  providerItems: AgentProvider[]
}

export function mapAgentsPageData(data: AgentsPageData): AgentsPageStateSnapshot {
  return {
    providerItems: data.providerItems,
    agents: data.agents,
    agentRuns: data.agentRuns,
  }
}
