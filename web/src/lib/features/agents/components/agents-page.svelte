<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { createAgent, pauseAgentRuntime, resumeAgentRuntime, updateProvider } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { loadAgentsPageData } from '../data'
  import {
    applyUpdatedProviderState,
    createEmptyProviderDraft,
    parseProviderDraft,
    providerToDraft,
  } from '../model'
  import {
    createAgentRegistrationDraft,
    parseAgentRegistrationDraft,
    type AgentRegistrationDraft,
    type AgentRegistrationDraftField,
  } from '../registration'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  import AgentsPageDrawers from './agents-page-drawers.svelte'
  import AgentsPagePanel from './agents-page-panel.svelte'

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
    if (!selectedProvider) {
      providerError = 'Select a provider to configure.'
      return
    }

    const parsed = parseProviderDraft(providerDraft)
    if (!parsed.ok) {
      providerError = parsed.error
      providerFeedback = ''
      return
    }

    providerSaving = true
    providerFeedback = ''
    providerError = ''

    try {
      const payload = await updateProvider(selectedProvider.id, parsed.value)
      if (payload.provider) {
        applyUpdatedProvider(payload.provider)
        providerFeedback = 'Provider updated.'
        return
      }

      providerError =
        'Provider updated, but the latest provider data could not be refreshed. Please reload the page.'
    } catch (caughtError) {
      providerError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save provider.'
    } finally {
      providerSaving = false
    }
  }

  async function handlePauseAgent(agent: AgentInstance) {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      pageError = 'Project context is unavailable.'
      return
    }

    runtimeControlPendingAgentId = agent.id
    pageError = ''
    pageFeedback = ''

    try {
      await pauseAgentRuntime(agent.id)
      pageFeedback = `Paused ${agent.name}.`
      await loadData({ projectId, orgId, showLoading: false })
    } catch (caughtError) {
      pageError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to pause agent.'
    } finally {
      runtimeControlPendingAgentId = null
    }
  }

  async function handleResumeAgent(agent: AgentInstance) {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      pageError = 'Project context is unavailable.'
      return
    }

    runtimeControlPendingAgentId = agent.id
    pageError = ''
    pageFeedback = ''

    try {
      await resumeAgentRuntime(agent.id)
      pageFeedback = `Resumed ${agent.name}.`
      await loadData({ projectId, orgId, showLoading: false })
    } catch (caughtError) {
      pageError = caughtError instanceof ApiError ? caughtError.detail : 'Failed to resume agent.'
    } finally {
      runtimeControlPendingAgentId = null
    }
  }

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    providerItems = providerItems.map((provider) =>
      provider.id === updatedProvider.id ? updatedProvider : provider,
    )

    const nextState = applyUpdatedProviderState(providers, agents, updatedProvider)
    providers = nextState.providers
    agents = nextState.agents
    if (nextState.provider) {
      providerDraft = providerToDraft(nextState.provider)
    }
  }
</script>

<div class="space-y-4">
  <AgentsPagePanel
    bind:activeTab
    {agents}
    {providers}
    {loading}
    {error}
    {pageError}
    {pageFeedback}
    {runtimeControlPendingAgentId}
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
    onConfigureProvider={handleConfigureProvider}
    onPauseAgent={handlePauseAgent}
    onResumeAgent={handleResumeAgent}
  />
</div>

<AgentsPageDrawers
  bind:registerSheetOpen
  bind:providerConfigOpen
  {providerItems}
  {registrationDraft}
  {registerSaving}
  {registerError}
  {registerFeedback}
  onRegistrationDraftChange={updateRegistrationDraft}
  onRegisterAgent={handleRegisterAgent}
  onRegisterOpenChange={handleRegisterOpenChange}
  {selectedProvider}
  {providerDraft}
  {providerSaving}
  {providerFeedback}
  {providerError}
  onProviderDraftChange={handleProviderDraftChange}
  onProviderSave={handleProviderSave}
/>
