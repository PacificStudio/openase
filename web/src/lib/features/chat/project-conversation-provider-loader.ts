import { ApiError } from '$lib/api/client'
import type { AgentProvider } from '$lib/api/contracts'
import { listProviders } from '$lib/api/openase'

export async function loadProjectConversationProviders(organizationId: string): Promise<{
  providers: AgentProvider[]
  error: string
}> {
  try {
    const payload = await listProviders(organizationId)
    return {
      providers: payload.providers,
      error: '',
    }
  } catch (caughtError) {
    return {
      providers: [],
      error:
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to load chat providers.',
    }
  }
}
