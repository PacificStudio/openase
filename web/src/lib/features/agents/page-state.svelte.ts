import { ApiError } from '$lib/api/client'
import type { AgentOutputEntry, AgentProvider } from '$lib/api/contracts'
import { loadAgentsPageData } from './data'
import { createEmptyProviderDraft, providerToDraft } from './model'
import { describeProviderSaveError, saveProvider } from './page-provider'
import { describeRegisterAgentError, registerAgent } from './page-registration'
import {
  createAgentRegistrationDraft,
  type AgentRegistrationDraft,
  type AgentRegistrationDraftField,
} from './registration'
import { describeAgentOutputError, fetchAgentOutput } from './output'
import type { AgentInstance, ProviderConfig, ProviderDraft, ProviderDraftField } from './types'

type LoadDataOptions = {
  projectId: string
  orgId: string
  defaultProviderId?: string | null
  showLoading: boolean
}

type PageContext = {
  projectId?: string | null
  orgId?: string | null
  defaultProviderId?: string | null
}

export function createAgentsPageState() {
  const view = $state({
    activeTab: 'instances',
    agents: [] as AgentInstance[],
    providers: [] as ProviderConfig[],
    providerItems: [] as AgentProvider[],
    loading: false,
    error: '',
    registerSheetOpen: false,
    registerSaving: false,
    registerError: '',
    registerFeedback: '',
    pageFeedback: '',
    registrationDraft: createAgentRegistrationDraft([], null) as AgentRegistrationDraft,
    providerConfigOpen: false,
    selectedProviderId: null as string | null,
    providerDraft: createEmptyProviderDraft() as ProviderDraft,
    providerSaving: false,
    providerFeedback: '',
    providerError: '',
    outputSheetOpen: false,
    selectedOutputAgentId: null as string | null,
    outputEntries: [] as AgentOutputEntry[],
    outputLoading: false,
    outputError: '',
  })
  let loadVersion = 0

  const resetRegistrationDraft = (defaultProviderId?: string | null) => {
    view.registrationDraft = createAgentRegistrationDraft(view.providerItems, defaultProviderId)
    view.registerError = ''
    view.registerFeedback = ''
  }

  const resetProviderEditor = () => {
    view.providerConfigOpen = false
    view.selectedProviderId = null
    view.providerDraft = createEmptyProviderDraft()
    view.providerSaving = false
    view.providerFeedback = ''
    view.providerError = ''
  }

  const resetOutput = () => {
    view.outputSheetOpen = false
    view.selectedOutputAgentId = null
    view.outputEntries = []
    view.outputLoading = false
    view.outputError = ''
  }

  return {
    view,
    get selectedProvider() {
      return view.providers.find((provider) => provider.id === view.selectedProviderId) ?? null
    },
    get selectedOutputAgent() {
      return view.agents.find((agent) => agent.id === view.selectedOutputAgentId) ?? null
    },
    setProviderConfigOpen(open: boolean) {
      view.providerConfigOpen = open
      if (!open) {
        view.providerFeedback = ''
        view.providerError = ''
        view.providerSaving = false
      }
    },
    setOutputSheetOpen(open: boolean) {
      view.outputSheetOpen = open
      if (!open) {
        resetOutput()
      }
    },
    async loadData({ projectId, orgId, defaultProviderId, showLoading }: LoadDataOptions) {
      const requestVersion = ++loadVersion
      if (showLoading) {
        view.loading = true
      }
      view.error = ''

      try {
        const nextData = await loadAgentsPageData(projectId, orgId, defaultProviderId ?? null)
        if (requestVersion !== loadVersion) return

        view.providerItems = nextData.providerItems
        view.providers = nextData.providers
        view.agents = nextData.agents
        if (view.outputSheetOpen && view.selectedOutputAgentId) {
          void this.loadOutput(view.selectedOutputAgentId)
        }
      } catch (caughtError) {
        if (requestVersion !== loadVersion) return
        view.error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agents.'
      } finally {
        if (requestVersion === loadVersion && showLoading) {
          view.loading = false
        }
      }
    },
    reset(defaultProviderId?: string | null) {
      loadVersion += 1
      view.agents = []
      view.providers = []
      view.providerItems = []
      view.loading = false
      view.error = ''
      view.pageFeedback = ''
      resetRegistrationDraft(defaultProviderId)
      resetProviderEditor()
      resetOutput()
    },
    updateRegistrationDraft(field: AgentRegistrationDraftField, value: string) {
      view.registrationDraft = {
        ...view.registrationDraft,
        [field]: value,
      }
    },
    handleRegisterOpenChange(open: boolean, defaultProviderId?: string | null) {
      view.registerSheetOpen = open
      if (open) {
        resetRegistrationDraft(defaultProviderId)
        view.pageFeedback = ''
        return
      }

      view.registerError = ''
      view.registerFeedback = ''
    },
    async handleRegisterAgent({ projectId, orgId, defaultProviderId }: PageContext) {
      if (!projectId || !orgId) {
        view.registerError = 'Project context is unavailable.'
        return
      }

      view.registerSaving = true
      view.registerError = ''
      view.registerFeedback = ''

      try {
        const registeredName = await registerAgent({
          projectId,
          draft: view.registrationDraft,
          providerItems: view.providerItems,
        })
        view.registerFeedback = 'Agent created. Refreshing list...'
        await this.loadData({ projectId, orgId, defaultProviderId, showLoading: false })
        view.pageFeedback = `Registered ${registeredName}.`
        view.registerSheetOpen = false
        resetRegistrationDraft(defaultProviderId)
      } catch (caughtError) {
        view.registerError = describeRegisterAgentError(caughtError)
      } finally {
        view.registerSaving = false
      }
    },
    handleConfigureProvider(provider: ProviderConfig) {
      view.selectedProviderId = provider.id
      view.providerDraft = providerToDraft(provider)
      view.providerConfigOpen = true
      view.providerSaving = false
      view.providerFeedback = ''
      view.providerError = ''
    },
    handleProviderDraftChange(field: ProviderDraftField, value: string) {
      view.providerDraft = {
        ...view.providerDraft,
        [field]: value,
      }
    },
    async handleProviderSave() {
      const selectedProvider =
        view.providers.find((provider) => provider.id === view.selectedProviderId) ?? null
      if (!selectedProvider) {
        view.providerError = 'Select a provider to configure.'
        return
      }

      view.providerSaving = true
      view.providerFeedback = ''
      view.providerError = ''

      try {
        const nextState = await saveProvider({
          agents: view.agents,
          providerDraft: view.providerDraft,
          providerItems: view.providerItems,
          providers: view.providers,
          selectedProvider,
        })
        view.providerItems = nextState.providerItems
        view.providers = nextState.providers
        view.agents = nextState.agents
        view.providerDraft = nextState.providerDraft
        view.providerFeedback = 'Provider updated.'
      } catch (caughtError) {
        view.providerError = describeProviderSaveError(caughtError)
      } finally {
        view.providerSaving = false
      }
    },
    handleOpenOutput(agent: AgentInstance) {
      view.selectedOutputAgentId = agent.id
      view.outputSheetOpen = true
      void this.loadOutput(agent.id)
    },
    async loadOutput(agentId: string) {
      view.outputLoading = true
      view.outputError = ''

      try {
        const nextEntries = await fetchAgentOutput(agentId)
        if (view.selectedOutputAgentId !== agentId) return
        view.outputEntries = nextEntries
      } catch (caughtError) {
        if (view.selectedOutputAgentId !== agentId) return
        view.outputError = describeAgentOutputError(caughtError)
      } finally {
        if (view.selectedOutputAgentId === agentId) {
          view.outputLoading = false
        }
      }
    },
    refreshOutput() {
      if (view.selectedOutputAgentId) {
        void this.loadOutput(view.selectedOutputAgentId)
      }
    },
  }
}
