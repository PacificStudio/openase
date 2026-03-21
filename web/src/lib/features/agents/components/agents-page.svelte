<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { createAgent } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { loadAgentsPageData } from '../data'
  import { applyUpdatedProviderState } from '../model'
  import { createAgentRegistrationDraft, parseAgentRegistrationDraft } from '../registration'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  import { createAgentOutputState } from './agent-output-state.svelte'
  import { createProviderEditorState } from './provider-editor-state.svelte'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

  let activeTab = $state('instances')
  let agents = $state<AgentInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false),
    error = $state('')
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
    const requestVersion = ++loadVersion
    if (showLoading) {
      loading = true
    }
    error = ''

    try {
      const nextData = await loadAgentsPageData(
        projectId,
        orgId,
        appStore.currentOrg?.default_agent_provider_id ?? null,
      )
      if (requestVersion !== loadVersion) return

      providerItems = nextData.providerItems
      providers = nextData.providers
      agents = nextData.agents
    } catch (caughtError) {
      if (requestVersion !== loadVersion) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agents.'
    } finally {
      if (requestVersion === loadVersion && showLoading) {
        loading = false
      }
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
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      registerError = 'Project context is unavailable.'
      return
    }

    const parsed = parseAgentRegistrationDraft(registrationDraft, providerItems)
    if (!parsed.ok) {
      registerError = parsed.error
      return
    }

    registerSaving = true
    registerError = ''
    registerFeedback = ''

    try {
      await createAgent(projectId, {
        provider_id: parsed.value.providerId,
        name: parsed.value.name,
        workspace_path: parsed.value.workspacePath,
        capabilities: parsed.value.capabilities,
      })

      registerFeedback = 'Agent created. Refreshing list...'
      await loadData({ projectId, orgId, showLoading: false })
      pageFeedback = `Registered ${parsed.value.name}.`
      registerSheetOpen = false
      resetRegistrationDraft()
    } catch (caughtError) {
      registerError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to register agent.'
    } finally {
      registerSaving = false
    }
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

<AgentsPagePanel
  bind:activeTab
  {agents}
  {providers}
  {loading}
  {error}
  {pageFeedback}
  canRegister={!!appStore.currentProject?.id && providerItems.length > 0}
  registerButtonTitle={providerItems.length === 0
    ? 'Register a provider before creating agents.'
    : appStore.currentProject?.id
      ? undefined
      : 'Project context is unavailable.'}
  onOpenRegister={() => handleRegisterOpenChange(true)}
  onSelectTicket={(ticketId) => {
    appStore.openRightPanel({ type: 'ticket', id: ticketId })
  }}
  onViewOutput={handleOpenAgentOutput}
  onConfigureProvider={handleConfigureProvider}
/>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:providerConfigOpen
  bind:outputSheetOpen
  {providerItems}
  {registrationDraft}
  {registerSaving}
  {registerError}
  {registerFeedback}
  onRegistrationDraftChange={updateRegistrationDraft}
  onRegisterAgent={handleRegisterAgent}
  onRegisterOpenChange={handleRegisterOpenChange}
  {selectedProvider}
  providerDraft={providerEditor.draft}
  providerSaving={providerEditor.saving}
  providerFeedback={providerEditor.feedback}
  providerError={providerEditor.error}
  {selectedOutputAgent}
  outputEntries={outputState.entries}
  outputLoading={outputState.loading}
  outputError={outputState.error}
  outputStreamState={outputState.streamState}
  onProviderDraftChange={handleProviderDraftChange}
  onProviderSave={handleProviderSave}
  onOutputOpenChange={handleOutputOpenChange}
/>
