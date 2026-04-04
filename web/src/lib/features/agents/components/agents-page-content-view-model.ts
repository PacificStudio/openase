import type { AgentOutputEntry, AgentProvider, AgentStepEntry } from '$lib/api/contracts'
import type { StreamConnectionState } from '$lib/api/sse'
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
  onViewOutput: (agentId: string) => void
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
  selectedOutputAgent: AgentInstance | null
  outputEntries: AgentOutputEntry[]
  outputSteps: AgentStepEntry[]
  outputLoading: boolean
  outputError: string
  outputStreamState: StreamConnectionState
  onProviderDraftChange: (field: ProviderDraftField, value: string) => void
  onProviderSave: () => void
  onOutputOpenChange: (open: boolean) => void
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
  onViewOutput: (agentId: string) => void
  onConfigureProvider: (provider: ProviderConfig) => void
  onPauseAgent: (agent: AgentInstance) => void
  onResumeAgent: (agent: AgentInstance) => void
  selectedProvider: ProviderConfig | null
  providerDraft: ProviderDraft
  providerSaving: boolean
  providerFeedback: string
  providerError: string
  selectedOutputAgent: AgentInstance | null
  outputEntries: AgentOutputEntry[]
  outputSteps: AgentStepEntry[]
  outputLoading: boolean
  outputError: string
  outputStreamState: StreamConnectionState
  onProviderDraftChange: (field: ProviderDraftField, value: string) => void
  onProviderSave: () => void
  onOutputOpenChange: (open: boolean) => void
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
  onViewOutput,
  onConfigureProvider,
  onPauseAgent,
  onResumeAgent,
  selectedProvider,
  providerDraft,
  providerSaving,
  providerFeedback,
  providerError,
  selectedOutputAgent,
  outputEntries,
  outputSteps,
  outputLoading,
  outputError,
  outputStreamState,
  onProviderDraftChange,
  onProviderSave,
  onOutputOpenChange,
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
    onViewOutput,
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
    selectedOutputAgent,
    outputEntries,
    outputSteps,
    outputLoading,
    outputError,
    outputStreamState,
    onProviderDraftChange,
    onProviderSave,
    onOutputOpenChange,
  }
}
