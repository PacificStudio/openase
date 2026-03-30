<script lang="ts">
  import { onMount } from 'svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { writeHashSelection } from '$lib/utils/hash-state'
  import type { AgentProvider, Machine } from '$lib/api/contracts'
  import type { AgentsPageData } from '../data'
  import { loadAgentsPageResult } from '../page-data'
  import { applyUpdatedProviderState } from '../model'
  import { createAgentRegistrationDraft } from '../registration'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type {
    AgentInstance,
    AgentRunInstance,
    ProviderConfig,
    ProviderDraftField,
  } from '../types'
  import AgentsPageContent from './agents-page-content.svelte'
  import { registerAgentPageAction, runAgentRuntimePageAction } from './agents-page-actions'
  import { createAgentOutputState } from './agent-output-state.svelte'
  import { mapAgentsPageData, resolveAgentPageTab, type AgentPageTab } from './agents-page-helpers'
  import { wireAgentOutputStream } from './agent-output-stream.svelte'
  import { createProviderEditorState } from './provider-editor-state.svelte'

  let activeTab = $state<AgentPageTab>('runtime')
  let hashSyncReady = $state(false)
  let agents = $state<AgentInstance[]>([])
  let agentRuns = $state<AgentRunInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let machineItems = $state<Machine[]>([])
  let loading = $state(false),
    error = $state('')
  let registerSheetOpen = $state(false)
  let registerSaving = $state(false)
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )
  let providerConfigOpen = $state(false),
    outputSheetOpen = $state(false),
    loadVersion = 0
  const outputState = createAgentOutputState(),
    providerEditor = createProviderEditorState()
  let outputAgentId = $state<string | null>(null)
  let runtimeActionAgentId = $state<string | null>(null)

  const selectedProvider = $derived(
    providers.find((provider) => provider.id === providerEditor.selectedProviderId) ?? null,
  )
  const selectedOutputAgent = $derived(agents.find((agent) => agent.id === outputAgentId) ?? null)

  onMount(() => {
    activeTab = resolveAgentPageTab()
    hashSyncReady = true

    const handleHashChange = () => (activeTab = resolveAgentPageTab())

    window.addEventListener('hashchange', handleHashChange)

    return () => {
      window.removeEventListener('hashchange', handleHashChange)
    }
  })

  $effect(() => {
    if (!hashSyncReady) {
      return
    }

    writeHashSelection(activeTab)
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      agentRuns = []
      providers = []
      providerItems = []
      machineItems = []
      resetRegistrationDraft()
      providerEditor.reset()
      outputAgentId = null
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

  wireAgentOutputStream({
    projectId: () => appStore.currentProject?.id,
    isOpen: () => outputSheetOpen,
    selectedAgentId: () => outputAgentId,
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
    ;({ providerItems, machineItems, providers, agents, agentRuns } = mapAgentsPageData(data))
  }

  function updateRegistrationDraft(field: AgentRegistrationDraftField, value: string) {
    registrationDraft = { ...registrationDraft, [field]: value }
  }

  function resetRegistrationDraft() {
    registrationDraft = createAgentRegistrationDraft(
      providerItems,
      appStore.currentOrg?.default_agent_provider_id,
    )
  }

  function handleRegisterOpenChange(open: boolean) {
    registerSheetOpen = open
    if (open) {
      resetRegistrationDraft()
    }
  }

  async function handleRegisterAgent() {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    registerSaving = true

    const result = await registerAgentPageAction({
      projectId,
      orgId,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id,
      draft: registrationDraft,
      providerItems,
    })
    registerSaving = false

    if (!result.ok) {
      toastStore.error(result.error)
      return
    }

    applyPageData(result.data)
    toastStore.success(result.feedback)
    registerSheetOpen = false
    resetRegistrationDraft()
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
    await providerEditor.save(selectedProvider, applyUpdatedProvider)
  }

  async function handleRuntimeAction(action: 'pause' | 'resume', agentId: string) {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    runtimeActionAgentId = agentId
    const result = await runAgentRuntimePageAction({
      action,
      agentId,
      projectId,
      orgId,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id,
    })
    runtimeActionAgentId = null

    if (!result.ok) {
      toastStore.error(result.error)
      return
    }

    applyPageData(result.data)
    toastStore.success(result.feedback)
  }

  function handleOutputOpenChange(open: boolean) {
    outputSheetOpen = open
    if (!open) {
      outputAgentId = null
      outputState.reset()
    }
  }
</script>

<AgentsPageContent
  bind:activeTab
  bind:registerSheetOpen
  bind:providerConfigOpen
  bind:outputSheetOpen
  canRegister={!!appStore.currentProject?.id && providerItems.length > 0}
  {agents}
  {agentRuns}
  {providers}
  {loading}
  {error}
  {runtimeActionAgentId}
  registerButtonTitle={providerItems.length === 0
    ? 'Register a provider before creating agents.'
    : appStore.currentProject?.id
      ? undefined
      : 'Project context is unavailable.'}
  onOpenRegister={() => handleRegisterOpenChange(true)}
  onSelectTicket={(ticketId) => appStore.openRightPanel({ type: 'ticket', id: ticketId })}
  onViewOutput={(agentId) => {
    outputAgentId = agentId
    outputState.open(agentId)
    outputSheetOpen = true
  }}
  onConfigureProvider={handleConfigureProvider}
  onPauseAgent={(agentId) => handleRuntimeAction('pause', agentId)}
  onResumeAgent={(agentId) => handleRuntimeAction('resume', agentId)}
  {providerItems}
  {machineItems}
  {registrationDraft}
  currentOrgSlug={appStore.currentOrg?.slug}
  currentProjectSlug={appStore.currentProject?.slug}
  {registerSaving}
  onRegistrationDraftChange={updateRegistrationDraft}
  onRegisterAgent={handleRegisterAgent}
  onRegisterOpenChange={handleRegisterOpenChange}
  {selectedProvider}
  providerDraft={providerEditor.draft}
  providerSaving={providerEditor.saving}
  {selectedOutputAgent}
  outputEntries={outputState.entries}
  outputSteps={outputState.stepEntries}
  outputLoading={outputState.loading}
  outputError={outputState.error}
  outputStreamState={outputState.streamState}
  onProviderDraftChange={handleProviderDraftChange}
  onProviderSave={handleProviderSave}
  onOutputOpenChange={handleOutputOpenChange}
/>
