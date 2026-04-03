import type { AgentProvider } from '$lib/api/contracts'
import type { AgentInstance, AgentRunInstance } from './types'

export type AgentsPageSnapshot = {
  agents: AgentInstance[]
  agentRuns: AgentRunInstance[]
  providerItems: AgentProvider[]
  cachedAt: number
}

type AgentsPageCacheEntry = {
  snapshot: AgentsPageSnapshot | null
  dirty: boolean
}

const cache = new Map<string, AgentsPageCacheEntry>()

export function readAgentsPageCache(projectId: string, orgId: string) {
  const entry = cache.get(buildCacheKey(projectId, orgId))
  if (!entry?.snapshot) {
    return null
  }

  return {
    snapshot: entry.snapshot,
    dirty: entry.dirty,
  }
}

export function writeAgentsPageCache(
  projectId: string,
  orgId: string,
  value: {
    agents: AgentInstance[]
    agentRuns: AgentRunInstance[]
    providerItems: AgentProvider[]
  },
) {
  cache.set(buildCacheKey(projectId, orgId), {
    snapshot: {
      ...value,
      cachedAt: Date.now(),
    },
    dirty: false,
  })
}

export function markAgentsPageCacheDirty(projectId: string, orgId: string) {
  const entry = cache.get(buildCacheKey(projectId, orgId))
  if (!entry?.snapshot) {
    return
  }

  entry.dirty = true
}

export function resetAgentsPageCacheForTests() {
  cache.clear()
}

function buildCacheKey(projectId: string, orgId: string) {
  return `${projectId}:${orgId}`
}
