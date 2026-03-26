<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentsPageData } from '../data'
  import { loadAgentsPageResult } from '../page-data'
  import { applyUpdatedProviderState } from '../model'
  import { createAgentRegistrationDraft, parseAgentRegistrationDraft } from '../registration'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  import {
    registerAgentAndReload,
    registerAgentError,
    runRuntimeAction,
    runtimeActionError,
  } from '../runtime-actions'
  import { createAgentOutputState } from './agent-output-state.svelte'
  import { wireAgentOutputStream } from './agent-output-stream.svelte'
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
    pageFeedback = $state(''),
    pageError = $state('')
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )
  let providerConfigOpen = $state(false),
    outputSheetOpen = $state(false),
    loadVersion = 0
  const outputState = createAgentOutputState(),
    providerEditor = createProviderEditorState()
  let runtimeActionAgentId = $state<string | null>(null)

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
      outputState.reset()
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

  wireAgentOutputStream({
    projectId: () => appStore.currentProject?.id,
    isOpen: () => outputSheetOpen,
    outputState,
  })

  async function loadData(input: { projectId: string; orgId: string; showLoading: boolean }) {
    const requestVersion = ++loadVersion
    if (input.showLoading) loading = true
    error = ''

    const result = await loadAgentsPageResult({
      projectId: input.projectId,
      orgId: input.orgId,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id ?? null,
    })
    if (requestVersion !== loadVersion) return

    if (result.ok) {
      applyPageData(result.data)
    } else {
      error = result.error
    }
    if (input.showLoading) loading = false
  }

  function applyPageData(data: AgentsPageData) {
    providerItems = data.providerItems
    providers = data.providers
    agents = data.agents
  }

  function updateRegistrationDraft(field: AgentRegistrationDraftField, value: string) {
    registrationDraft = { ...registrationDraft, [field]: value }
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
    pageError = ''

    try {
      const result = await registerAgentAndReload({
        projectId,
        orgId,
        defaultProviderId: appStore.currentOrg?.default_agent_provider_id ?? null,
        providerId: parsed.value.providerId,
        name: parsed.value.name,
        workspacePath: parsed.value.workspacePath,
      })
      applyPageData(result.data)
      registerFeedback = 'Agent created.'
      pageFeedback = result.feedback
      registerSheetOpen = false
      resetRegistrationDraft()
    } catch (caughtError) {
      registerError = registerAgentError(caughtError)
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

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    providerItems = providerItems.map((provider) =>
      provider.id === updatedProvider.id ? updatedProvider : provider,
    )

    const nextState = applyUpdatedProviderState(providers, agents, updatedProvider)
    providers = nextState.providers
    agents = nextState.agents
    if (nextState.provider) providerEditor.open(nextState.provider)
  }

  async function handleProviderSave() {
    pageError = ''
    await providerEditor.save(selectedProvider, applyUpdatedProvider)
  }

  async function handleRuntimeAction(action: 'pause' | 'resume', agentId: string) {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      pageError = 'Project context is unavailable.'
      return
    }

    runtimeActionAgentId = agentId
    pageFeedback = ''
    pageError = ''

    try {
      const result = await runRuntimeAction({
        action,
        agentId,
        projectId,
        orgId,
        defaultProviderId: appStore.currentOrg?.default_agent_provider_id ?? null,
      })
      applyPageData(result.data)
      pageFeedback = result.feedback
    } catch (caughtError) {
      pageError = runtimeActionError(action, caughtError)
    } finally {
      runtimeActionAgentId = null
    }
  }

  function handleOutputOpenChange(open: boolean) {
    outputSheetOpen = open
    if (!open) outputState.reset()
  }
</script>

<AgentsPagePanel
  bind:activeTab
  {agents}
  {providers}
  {loading}
  {error}
  {pageFeedback}
  {pageError}
  {runtimeActionAgentId}
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
  onViewOutput={(agentId) => {
    outputState.open(agentId)
    outputSheetOpen = true
  }}
  onConfigureProvider={handleConfigureProvider}
  onPauseAgent={(agentId) => handleRuntimeAction('pause', agentId)}
  onResumeAgent={(agentId) => handleRuntimeAction('resume', agentId)}
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
