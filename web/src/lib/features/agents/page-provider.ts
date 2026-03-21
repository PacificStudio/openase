import { ApiError } from '$lib/api/client'
import { updateProvider } from '$lib/api/openase'
import type { AgentProvider } from '$lib/api/contracts'
import { applyUpdatedProviderState, parseProviderDraft, providerToDraft } from './model'
import type { AgentInstance, ProviderConfig, ProviderDraft } from './types'

type SaveProviderInput = {
  agents: AgentInstance[]
  providerDraft: ProviderDraft
  providerItems: AgentProvider[]
  providers: ProviderConfig[]
  selectedProvider: ProviderConfig
}

export async function saveProvider(input: SaveProviderInput) {
  const parsed = parseProviderDraft(input.providerDraft)
  if (!parsed.ok) {
    throw new Error(parsed.error)
  }

  const payload = await updateProvider(input.selectedProvider.id, parsed.value)
  const updatedProvider = payload.provider
  if (!updatedProvider) {
    throw new Error(
      'Provider updated, but the latest provider data could not be refreshed. Please reload the page.',
    )
  }

  const nextProviderItems = input.providerItems.map((provider) =>
    provider.id === updatedProvider.id ? updatedProvider : provider,
  )
  const nextState = applyUpdatedProviderState(input.providers, input.agents, updatedProvider)

  return {
    providerDraft: nextState.provider ? providerToDraft(nextState.provider) : input.providerDraft,
    providerItems: nextProviderItems,
    providers: nextState.providers,
    agents: nextState.agents,
  }
}

export function describeProviderSaveError(caughtError: unknown) {
  return caughtError instanceof ApiError ? caughtError.detail : 'Failed to save provider.'
}
