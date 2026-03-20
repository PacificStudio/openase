import { api } from '$lib/features/workspace'
import type { AgentProvider } from './types'

export async function loadAgentProviders(organizationId: string) {
  const payload = await api<{ providers: AgentProvider[] }>(
    `/api/v1/orgs/${organizationId}/providers`,
  )
  return payload.providers
}
