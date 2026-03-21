import type { AgentProvider } from '$lib/api/contracts'
import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
import type { AgentInstance, ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'

export type ContentViewModel = {
  agents: AgentInstance[]
  providers: ProviderConfig[]
  loading: boolean
  error: string
  pageError: string
  pageFeedback: string
  runtimeControlPendingAgentId: string | null
  canRegister: boolean
  registerButtonTitle?: string
  onOpenRegister: () => void
  onSelectTicket: (ticketId: string) => void
  onConfigureProvider: (provider: ProviderConfig) => void
  onPauseAgent: (agent: AgentInstance) => void
  onResumeAgent: (agent: AgentInstance) => void
  providerItems: AgentProvider[]
  registrationDraft: AgentRegistrationDraft
  registerSaving: boolean
  registerError: string
  registerFeedback: string
  onRegistrationDraftChange: (field: AgentRegistrationDraftField, value: string) => void
  onRegisterAgent: () => void
  onRegisterOpenChange: (open: boolean) => void
  selectedProvider: ProviderConfig | null
  providerDraft: ProviderDraft
  providerSaving: boolean
  providerFeedback: string
  providerError: string
  onProviderDraftChange: (field: ProviderDraftField, value: string) => void
  onProviderSave: () => void
}

type ContentViewModelOptions = {
  agents: AgentInstance[]
  providers: ProviderConfig[]
  loading: boolean
  error: string
  pageError: string
  pageFeedback: string
  runtimeControlPendingAgentId: string | null
  projectId?: string
  providerItems: AgentProvider[]
  registrationDraft: AgentRegistrationDraft
  registerSaving: boolean
  registerError: string
  registerFeedback: string
  onRegistrationDraftChange: (field: AgentRegistrationDraftField, value: string) => void
  onRegisterAgent: () => void
  onRegisterOpenChange: (open: boolean) => void
  onOpenTicket: (ticketId: string) => void
  onConfigureProvider: (provider: ProviderConfig) => void
  onPauseAgent: (agent: AgentInstance) => void
  onResumeAgent: (agent: AgentInstance) => void
  selectedProvider: ProviderConfig | null
  providerDraft: ProviderDraft
  providerSaving: boolean
  providerFeedback: string
  providerError: string
  onProviderDraftChange: (field: ProviderDraftField, value: string) => void
  onProviderSave: () => void
}

export function createContentViewModel({
  agents,
  providers,
  loading,
  error,
  pageError,
  pageFeedback,
  runtimeControlPendingAgentId,
  projectId,
  providerItems,
  registrationDraft,
  registerSaving,
  registerError,
  registerFeedback,
  onRegistrationDraftChange,
  onRegisterAgent,
  onRegisterOpenChange,
  onOpenTicket,
  onConfigureProvider,
  onPauseAgent,
  onResumeAgent,
  selectedProvider,
  providerDraft,
  providerSaving,
  providerFeedback,
  providerError,
  onProviderDraftChange,
  onProviderSave,
}: ContentViewModelOptions): ContentViewModel {
  return {
    agents,
    providers,
    loading,
    error,
    pageError,
    pageFeedback,
    runtimeControlPendingAgentId,
    canRegister: !!projectId && providerItems.length > 0,
    registerButtonTitle:
      providerItems.length === 0
        ? 'Register a provider before creating agents.'
        : projectId
          ? undefined
          : 'Project context is unavailable.',
    onOpenRegister: () => onRegisterOpenChange(true),
    onSelectTicket: onOpenTicket,
    onConfigureProvider,
    onPauseAgent,
    onResumeAgent,
    providerItems,
    registrationDraft,
    registerSaving,
    registerError,
    registerFeedback,
    onRegistrationDraftChange,
    onRegisterAgent,
    onRegisterOpenChange,
    selectedProvider,
    providerDraft,
    providerSaving,
    providerFeedback,
    providerError,
    onProviderDraftChange,
    onProviderSave,
  }
}
