import { api } from '$lib/features/workspace'
import type { AgentProviderListPayload } from '$lib/features/workspace'

export async function loadAgentProviders(organizationId: string) {
  const payload = await api<AgentProviderListPayload>(`/api/v1/orgs/${organizationId}/providers`)
  return payload.providers
}
