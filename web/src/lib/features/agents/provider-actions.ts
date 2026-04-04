import { ApiError } from '$lib/api/client'
import { updateProvider } from '$lib/api/openase'
import type { AgentProvider } from '$lib/api/contracts'
import { applyUpdatedProviderState } from './model'
import { parseProviderDraft, providerToDraft } from './provider-draft'
import type { AgentInstance, ProviderConfig, ProviderDraft, ProviderMutation } from './types'

export async function saveProviderAndApply(input: {
  selectedProviderId: string
  mutation: ProviderMutation
  providerItems: AgentProvider[]
  providers: ProviderConfig[]
  agents: AgentInstance[]
}): Promise<
  | {
      ok: true
      providerItems: AgentProvider[]
      providers: ProviderConfig[]
      agents: AgentInstance[]
      providerDraft: ProviderDraft | null
    }
  | { ok: false; error: string }
> {
  const payload = await updateProvider(input.selectedProviderId, input.mutation)
  const updatedProvider = payload.provider
  if (!updatedProvider) {
    return {
      ok: false,
      error:
        'Provider updated, but the latest provider data could not be refreshed. Please reload the page.',
    }
  }

  const providerItems = input.providerItems.map((provider) =>
    provider.id === updatedProvider.id ? updatedProvider : provider,
  )
  const nextState = applyUpdatedProviderState(input.providers, input.agents, updatedProvider)

  return {
    ok: true,
    providerItems,
    providers: nextState.providers,
    agents: nextState.agents,
    providerDraft: nextState.provider ? providerToDraft(nextState.provider) : null,
  }
}

export function providerSaveError(error: unknown) {
  return error instanceof ApiError ? error.detail : 'Failed to save provider.'
}

export async function saveProviderDraft(input: {
  selectedProviderId: string | null
  draft: ProviderDraft
  providerItems: AgentProvider[]
  providers: ProviderConfig[]
  agents: AgentInstance[]
}): Promise<
  | {
      ok: true
      providerItems: AgentProvider[]
      providers: ProviderConfig[]
      agents: AgentInstance[]
      providerDraft: ProviderDraft | null
    }
  | { ok: false; error: string }
> {
  if (!input.selectedProviderId) {
    return { ok: false, error: 'Select a provider to configure.' }
  }

  const parsed = parseProviderDraft(input.draft)
  if (!parsed.ok) {
    return { ok: false, error: parsed.error }
  }

  try {
    return await saveProviderAndApply({
      selectedProviderId: input.selectedProviderId,
      mutation: parsed.value,
      providerItems: input.providerItems,
      providers: input.providers,
      agents: input.agents,
    })
  } catch (error) {
    return { ok: false, error: providerSaveError(error) }
  }
}
