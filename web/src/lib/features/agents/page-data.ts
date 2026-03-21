import { ApiError } from '$lib/api/client'
import { loadAgentsPageData, type AgentsPageData } from './data'

export async function loadAgentsPageResult(input: {
  projectId: string
  orgId: string
  defaultProviderId: string | null
}): Promise<{ ok: true; data: AgentsPageData } | { ok: false; error: string }> {
  try {
    return {
      ok: true,
      data: await loadAgentsPageData(input.projectId, input.orgId, input.defaultProviderId),
    }
  } catch (error) {
    return {
      ok: false,
      error: error instanceof ApiError ? error.detail : 'Failed to load agents.',
    }
  }
}
