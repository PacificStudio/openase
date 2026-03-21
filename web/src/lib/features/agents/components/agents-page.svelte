<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { pauseAgentRuntime, resumeAgentRuntime } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { createEmptyProviderDraft, providerToDraft } from '../model'
  import {
    createAgentRegistrationDraft,
    type AgentRegistrationDraft,
    type AgentRegistrationDraftField,
  } from '../registration'
  import {
    applyUpdatedProviderResult,
    runAgentsPageLoad,
    runAgentRegistration,
    runProviderSave,
  } from '../page-actions'
  import { createRuntimeControlHandler } from '../runtime-control'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  import { createContentViewModel } from './agents-page-content-view-model'
  import AgentsPageContent from './agents-page-content.svelte'

  let activeTab = $state('instances')
  let agents = $state<AgentInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false),
    error = $state('')
  let pageError = $state('')
  let registerSheetOpen = $state(false)
  let registerSaving = $state(false)
  let registerError = $state(''),
    registerFeedback = $state(''),
    pageFeedback = $state('')
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )
  let providerConfigOpen = $state(false),
    selectedProviderId = $state<string | null>(null)
  let providerDraft = $state(createEmptyProviderDraft())
  let providerSaving = $state(false),
    providerFeedback = $state(''),
    providerError = $state('')
  let runtimeControlPendingAgentId = $state<string | null>(null)
  let loadVersion = 0

  const selectedProvider = $derived(
    providers.find((provider) => provider.id === selectedProviderId) ?? null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      providers = []
      providerItems = []
      resetRegistrationDraft()
      resetProviderEditor()
      return
    }

    void loadData({ projectId, orgId, showLoading: true })

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
      onEvent: () => {
        void loadData({ projectId, orgId, showLoading: false })
      },
      onError: (streamError) => {
        console.error('Agents stream error:', streamError)
      },
    })

    return () => {
      loadVersion += 1
      disconnect()
    }
  })

  $effect(() => {
    if (!providerConfigOpen) {
      providerFeedback = providerError = ''
      providerSaving = false
    }
  })

  async function loadData({
    projectId,
    orgId,
    showLoading,
  }: {
    projectId: string
    orgId: string
    showLoading: boolean
  }) {
    const result = await runAgentsPageLoad({
      projectId,
      orgId,
      showLoading,
      requestVersion: ++loadVersion,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id,
      setLoading: (value) => (loading = value),
      setError: (value) => (error = value),
      setProviderItems: (items) => (providerItems = items),
      setProviders: (nextProviders) => (providers = nextProviders),
      setAgents: (nextAgents) => (agents = nextAgents),
    })
    if (result.requestVersion === loadVersion) {
      result.apply()
    }
  }

  function updateRegistrationDraft(field: AgentRegistrationDraftField, value: string) {
    registrationDraft = {
      ...registrationDraft,
      [field]: value,
    }
  }

  function resetRegistrationDraft() {
    registrationDraft = createAgentRegistrationDraft(
      providerItems,
      appStore.currentOrg?.default_agent_provider_id,
    )
    registerError = registerFeedback = ''
  }

  function handleRegisterOpenChange(open: boolean) {
    registerSheetOpen = open
    if (open) {
      resetRegistrationDraft()
      pageFeedback = ''
      return
    }

    registerError = registerFeedback = ''
  }

  async function handleRegisterAgent() {
    await runAgentRegistration({
      projectId: appStore.currentProject?.id,
      orgId: appStore.currentOrg?.id,
      getDraft: () => registrationDraft,
      getProviderItems: () => providerItems,
      loadData,
      resetDraft: resetRegistrationDraft,
      setSheetOpen: (open) => (registerSheetOpen = open),
      setSaving: (saving) => (registerSaving = saving),
      setError: (message) => (registerError = message),
      setFeedback: (message) => (registerFeedback = message),
      setPageFeedback: (message) => (pageFeedback = message),
    })
  }

  function resetProviderEditor() {
    providerConfigOpen = false
    selectedProviderId = null
    providerDraft = createEmptyProviderDraft()
    providerSaving = false
    providerFeedback = providerError = ''
  }

  function handleConfigureProvider(provider: ProviderConfig) {
    selectedProviderId = provider.id
    providerDraft = providerToDraft(provider)
    providerConfigOpen = true
    providerSaving = false
    providerFeedback = providerError = ''
  }

  function handleProviderDraftChange(field: ProviderDraftField, value: string) {
    providerDraft = {
      ...providerDraft,
      [field]: value,
    }
  }

  async function handleProviderSave() {
    await runProviderSave({
      selectedProvider,
      draft: providerDraft,
      applyUpdatedProvider,
      setSaving: (saving) => (providerSaving = saving),
      setFeedback: (message) => (providerFeedback = message),
      setError: (message) => (providerError = message),
    })
  }

  const handlePauseAgent = createRuntimeControlHandler({
    action: pauseAgentRuntime,
    successPrefix: 'Paused',
    failureMessage: 'Failed to pause agent.',
    getProjectId: () => appStore.currentProject?.id,
    getOrgId: () => appStore.currentOrg?.id,
    loadData,
    setPendingAgentId: (value) => (runtimeControlPendingAgentId = value),
    setPageError: (value) => (pageError = value),
    setPageFeedback: (value) => (pageFeedback = value),
  })
  const handleResumeAgent = createRuntimeControlHandler({
    action: resumeAgentRuntime,
    successPrefix: 'Resumed',
    failureMessage: 'Failed to resume agent.',
    getProjectId: () => appStore.currentProject?.id,
    getOrgId: () => appStore.currentOrg?.id,
    loadData,
    setPendingAgentId: (value) => (runtimeControlPendingAgentId = value),
    setPageError: (value) => (pageError = value),
    setPageFeedback: (value) => (pageFeedback = value),
  })

  const contentViewModel = $derived(
    createContentViewModel({
      agents,
      providers,
      loading,
      error,
      pageError,
      pageFeedback,
      runtimeControlPendingAgentId,
      projectId: appStore.currentProject?.id,
      providerItems,
      registrationDraft,
      registerSaving,
      registerError,
      registerFeedback,
      onRegistrationDraftChange: updateRegistrationDraft,
      onRegisterAgent: handleRegisterAgent,
      onRegisterOpenChange: handleRegisterOpenChange,
      onOpenTicket: (ticketId: string) => {
        appStore.openRightPanel({ type: 'ticket', id: ticketId })
      },
      onConfigureProvider: handleConfigureProvider,
      onPauseAgent: handlePauseAgent,
      onResumeAgent: handleResumeAgent,
      selectedProvider,
      providerDraft,
      providerSaving,
      providerFeedback,
      providerError,
      onProviderDraftChange: handleProviderDraftChange,
      onProviderSave: handleProviderSave,
    }),
  )

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    applyUpdatedProviderResult({
      providerItems,
      providers,
      agents,
      updatedProvider,
      setProviderItems: (items) => (providerItems = items),
      setProviders: (nextProviders) => (providers = nextProviders),
      setAgents: (nextAgents) => (agents = nextAgents),
      setProviderDraft: (draft) => (providerDraft = draft),
    })
  }
</script>

<AgentsPageContent
  bind:activeTab
  bind:registerSheetOpen
  bind:providerConfigOpen
  viewModel={contentViewModel}
/>
