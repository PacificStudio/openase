import { ApiError } from '$lib/api/client'
import type { AgentProvider } from '$lib/api/contracts'
import { listProviders } from '$lib/api/openase'
import {
  retainOrganizationEventBus,
  subscribeOrganizationProviderEvents,
} from '$lib/features/org-events'

type WatchProjectConversationProvidersInput = {
  organizationId: string
  hasInlineProviders: boolean
  setLoading: (value: boolean) => void
  setError: (value: string) => void
  setProviders: (value: AgentProvider[]) => void
}

export function watchProjectConversationProviders(input: WatchProjectConversationProvidersInput) {
  if (input.hasInlineProviders || !input.organizationId) {
    input.setLoading(false)
    input.setError('')
    input.setProviders([])
    return () => {}
  }

  let cancelled = false
  let loadingInFlight = false
  let queuedReload = false

  const load = async () => {
    if (loadingInFlight) {
      queuedReload = true
      return
    }

    loadingInFlight = true
    input.setLoading(true)
    do {
      queuedReload = false
      input.setError('')

      try {
        const payload = await listProviders(input.organizationId)
        if (!cancelled) {
          input.setProviders(payload.providers)
        }
      } catch (caughtError) {
        if (!cancelled) {
          input.setError(
            caughtError instanceof ApiError ? caughtError.detail : 'Failed to load chat providers.',
          )
        }
      }
    } while (!cancelled && queuedReload)

    loadingInFlight = false
    if (!cancelled) {
      input.setLoading(false)
    }
  }

  const releaseEventBus = retainOrganizationEventBus(input.organizationId, 'providers')
  const unsubscribe = subscribeOrganizationProviderEvents(input.organizationId, () => {
    void load()
  })

  void load()

  return () => {
    cancelled = true
    unsubscribe()
    releaseEventBus()
  }
}
