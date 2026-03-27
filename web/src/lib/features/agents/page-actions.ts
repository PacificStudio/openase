import { ApiError } from '$lib/api/client'
import { createAgent, updateProvider } from '$lib/api/openase'
import type { AgentProvider } from '$lib/api/contracts'
import { loadAgentsPageData } from './data'
import { applyUpdatedProviderState } from './model'
import { parseProviderDraft, providerToDraft } from './provider-draft'
import { parseAgentRegistrationDraft, type AgentRegistrationDraft } from './registration'
import type { ProviderDraft, ProviderConfig } from './types'

type LoadData = (params: {
  projectId: string
  orgId: string
  showLoading: boolean
}) => Promise<void>

type RegisterAgentOptions = {
  projectId?: string
  orgId?: string
  getDraft: () => AgentRegistrationDraft
  getProviderItems: () => AgentProvider[]
  loadData: LoadData
  resetDraft: () => void
  setSheetOpen: (open: boolean) => void
  setSaving: (saving: boolean) => void
  setError: (message: string) => void
  setFeedback: (message: string) => void
  setPageFeedback: (message: string) => void
}

type SaveProviderOptions = {
  selectedProvider: ProviderConfig | null
  draft: ProviderDraft
  applyUpdatedProvider: (provider: AgentProvider) => void
  setSaving: (saving: boolean) => void
  setFeedback: (message: string) => void
  setError: (message: string) => void
}

export async function runAgentRegistration({
  projectId,
  orgId,
  getDraft,
  getProviderItems,
  loadData,
  resetDraft,
  setSheetOpen,
  setSaving,
  setError,
  setFeedback,
  setPageFeedback,
}: RegisterAgentOptions) {
  if (!projectId || !orgId) {
    setError('Project context is unavailable.')
    return
  }

  const parsed = parseAgentRegistrationDraft(getDraft(), getProviderItems())
  if (!parsed.ok) {
    setError(parsed.error)
    return
  }

  setSaving(true)
  setError('')
  setFeedback('')

  try {
    await createAgent(projectId, {
      provider_id: parsed.value.providerId,
      name: parsed.value.name,
    })

    setFeedback('Agent created. Refreshing list...')
    await loadData({ projectId, orgId, showLoading: false })
    setPageFeedback(`Registered ${parsed.value.name}.`)
    setSheetOpen(false)
    resetDraft()
  } catch (caughtError) {
    setError(caughtError instanceof ApiError ? caughtError.detail : 'Failed to register agent.')
  } finally {
    setSaving(false)
  }
}

type ApplyUpdatedProviderOptions = {
  providerItems: AgentProvider[]
  providers: ProviderConfig[]
  agents: import('./types').AgentInstance[]
  updatedProvider: AgentProvider
  setProviderItems: (items: AgentProvider[]) => void
  setProviders: (providers: ProviderConfig[]) => void
  setAgents: (agents: import('./types').AgentInstance[]) => void
  setProviderDraft: (draft: ProviderDraft) => void
}

export function applyUpdatedProviderResult({
  providerItems,
  providers,
  agents,
  updatedProvider,
  setProviderItems,
  setProviders,
  setAgents,
  setProviderDraft,
}: ApplyUpdatedProviderOptions) {
  setProviderItems(
    providerItems.map((provider) =>
      provider.id === updatedProvider.id ? updatedProvider : provider,
    ),
  )

  const nextState = applyUpdatedProviderState(providers, agents, updatedProvider)
  setProviders(nextState.providers)
  setAgents(nextState.agents)
  if (nextState.provider) {
    setProviderDraft(providerToDraft(nextState.provider))
  }
}

export async function runProviderSave({
  selectedProvider,
  draft,
  applyUpdatedProvider,
  setSaving,
  setFeedback,
  setError,
}: SaveProviderOptions) {
  if (!selectedProvider) {
    setError('Select a provider to configure.')
    return
  }

  const parsed = parseProviderDraft(draft)
  if (!parsed.ok) {
    setError(parsed.error)
    setFeedback('')
    return
  }

  setSaving(true)
  setFeedback('')
  setError('')

  try {
    const payload = await updateProvider(selectedProvider.id, parsed.value)
    if (payload.provider) {
      applyUpdatedProvider(payload.provider)
      setFeedback('Provider updated.')
      return
    }

    setError(
      'Provider updated, but the latest provider data could not be refreshed. Please reload the page.',
    )
  } catch (caughtError) {
    setError(caughtError instanceof ApiError ? caughtError.detail : 'Failed to save provider.')
  } finally {
    setSaving(false)
  }
}

type LoadAgentsPageOptions = {
  projectId: string
  orgId: string
  showLoading: boolean
  requestVersion: number
  defaultProviderId?: string | null
  setLoading: (loading: boolean) => void
  setError: (message: string) => void
  setProviderItems: (items: AgentProvider[]) => void
  setProviders: (providers: ProviderConfig[]) => void
  setAgents: (agents: import('./types').AgentInstance[]) => void
}

export async function runAgentsPageLoad({
  projectId,
  orgId,
  showLoading,
  requestVersion,
  defaultProviderId,
  setLoading,
  setError,
  setProviderItems,
  setProviders,
  setAgents,
}: LoadAgentsPageOptions) {
  if (showLoading) {
    setLoading(true)
  }
  setError('')

  try {
    const nextData = await loadAgentsPageData(projectId, orgId, defaultProviderId ?? null)
    return {
      requestVersion,
      apply() {
        setProviderItems(nextData.providerItems)
        setProviders(nextData.providers)
        setAgents(nextData.agents)
        if (showLoading) {
          setLoading(false)
        }
      },
    }
  } catch (caughtError) {
    return {
      requestVersion,
      apply() {
        setError(caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agents.')
        if (showLoading) {
          setLoading(false)
        }
      },
    }
  }
}
