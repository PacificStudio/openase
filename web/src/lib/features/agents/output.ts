import { ApiError } from '$lib/api/client'
import { getAgentOutput } from '$lib/api/openase'
import type { AgentOutputEntry } from '$lib/api/contracts'

export async function fetchAgentOutput(agentId: string, limit = '80'): Promise<AgentOutputEntry[]> {
  const payload = await getAgentOutput(agentId, { limit })
  return payload.entries
}

export function describeAgentOutputError(caughtError: unknown) {
  return caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agent output.'
}
