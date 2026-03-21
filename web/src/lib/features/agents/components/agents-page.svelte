<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { pauseAgentRuntime, resumeAgentRuntime } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { applyUpdatedProviderState } from '../model'
  import {
    createAgentRegistrationDraft,
    type AgentRegistrationDraft,
    type AgentRegistrationDraftField,
  } from '../registration'
  import { runAgentsPageLoad, runAgentRegistration } from '../page-actions'
  import { createRuntimeControlHandler } from '../runtime-control'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  import { createAgentOutputState } from './agent-output-state.svelte'
  import { createProviderEditorState } from './provider-editor-state.svelte'
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
    outputSheetOpen = $state(false),
    loadVersion = 0
  let runtimeControlPendingAgentId = $state<string | null>(null)
  const outputState = createAgentOutputState(),
    providerEditor = createProviderEditorState()

  const selectedProvider = $derived(
    providers.find((provider) => provider.id === providerEditor.selectedProviderId) ?? null,
  )
  const selectedOutputAgent = $derived(
    agents.find((agent) => agent.id === outputState.selectedAgentId) ?? null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      providers = []
      providerItems = []
      resetRegistrationDraft()
      providerEditor.reset()
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
      providerEditor.clearMessages()
    }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const agentId = outputState.selectedAgentId

    if (!outputSheetOpen || !projectId || !agentId) {
      if (!outputSheetOpen) {
        outputState.reset()
      }
      return
    }

    void outputState.load(projectId, agentId, true)

    const disconnect = connectEventStream(
      `/api/v1/projects/${projectId}/agents/${agentId}/output/stream`,
      {
        onEvent: (frame) => outputState.handleFrame(agentId, frame),
        onStateChange: (state) => {
          outputState.streamState = state
        },
        onError: (streamError) => {
          console.error('Agent output stream error:', streamError)
        },
      },
    )

    return () => {
      outputState.invalidate()
      disconnect()
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

  function handleConfigureProvider(provider: ProviderConfig) {
    providerEditor.open(provider)
    providerConfigOpen = true
  }

  function handleProviderDraftChange(field: ProviderDraftField, value: string) {
    providerEditor.updateField(field, value)
  }

  async function handleProviderSave() {
    await providerEditor.save(selectedProvider, applyUpdatedProvider)
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
      onViewOutput: handleOpenAgentOutput,
      onConfigureProvider: handleConfigureProvider,
      onPauseAgent: handlePauseAgent,
      onResumeAgent: handleResumeAgent,
      selectedProvider,
      providerDraft: providerEditor.draft,
      providerSaving: providerEditor.saving,
      providerFeedback: providerEditor.feedback,
      providerError: providerEditor.error,
      selectedOutputAgent,
      outputEntries: outputState.entries,
      outputLoading: outputState.loading,
      outputError: outputState.error,
      outputStreamState: outputState.streamState,
      onProviderDraftChange: handleProviderDraftChange,
      onProviderSave: handleProviderSave,
      onOutputOpenChange: handleOutputOpenChange,
    }),
  )

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    providerItems = providerItems.map((provider) =>
      provider.id === updatedProvider.id ? updatedProvider : provider,
    )

    const nextState = applyUpdatedProviderState(providers, agents, updatedProvider)
    providers = nextState.providers
    agents = nextState.agents
    if (nextState.provider) {
      providerEditor.open(nextState.provider)
    }
  }

  async function handleOpenAgentOutput(agentId: string) {
    outputState.open(agentId)
    outputSheetOpen = true
  }

  function handleOutputOpenChange(open: boolean) {
    outputSheetOpen = open
    if (!open) {
      outputState.reset()
    }
  }
</script>

<AgentsPageContent
  bind:activeTab
  bind:registerSheetOpen
  bind:providerConfigOpen
  bind:outputSheetOpen
  viewModel={contentViewModel}
/>
