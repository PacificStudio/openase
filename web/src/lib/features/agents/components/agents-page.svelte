<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentsPageData } from '../data'
  import { loadAgentsPageResult } from '../page-data'
  import { saveProviderDraft } from '../provider-actions'
  import { createEmptyProviderDraft, providerToDraft } from '../model'
  import {
    createAgentRegistrationDraft,
    parseAgentRegistrationDraft,
    type AgentRegistrationDraft,
  } from '../registration'
  import {
    registerAgentAndReload,
    registerAgentError,
    runRuntimeAction,
    runtimeActionError,
  } from '../runtime-actions'
  import type { AgentInstance, ProviderConfig } from '../types'
  import AgentsPageContent from './agents-page-content.svelte'

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
  let pageError = $state('')
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )
  let providerConfigOpen = $state(false),
    selectedProviderId = $state<string | null>(null)
  let providerDraft = $state(createEmptyProviderDraft())
  let providerSaving = $state(false),
    providerFeedback = $state(''),
    providerError = $state('')
  let runtimeActionAgentId = $state<string | null>(null)
  let loadVersion = 0

  const selectedProvider = $derived(providers.find((p) => p.id === selectedProviderId) ?? null)
  const canRegister = $derived(!!appStore.currentProject?.id && providerItems.length > 0)
  const registerButtonTitle = $derived(
    providerItems.length === 0
      ? 'Register a provider before creating agents.'
      : appStore.currentProject?.id
        ? undefined
        : 'Project context is unavailable.',
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
    if (showLoading) loading = true
    error = ''

    const result = await loadAgentsPageResult({
      projectId,
      orgId,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id ?? null,
    })
    if (requestVersion !== loadVersion) return

    if (result.ok) {
      applyPageData(result.data)
    } else {
      error = result.error
    }
    if (showLoading) {
      loading = false
    }
  }

  function applyPageData(data: AgentsPageData) {
    providerItems = data.providerItems
    providers = data.providers
    agents = data.agents
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
    if (!open) return void (registerError = registerFeedback = '')
    resetRegistrationDraft()
    pageFeedback = ''
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
        capabilities: parsed.value.capabilities,
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

  async function handleProviderSave() {
    providerSaving = true
    providerFeedback = ''
    providerError = ''
    pageError = ''

    const result = await saveProviderDraft({
      selectedProviderId: selectedProvider?.id ?? null,
      draft: providerDraft,
      providerItems,
      providers,
      agents,
    })
    if (result.ok) {
      providerItems = result.providerItems
      providers = result.providers
      agents = result.agents
      if (result.providerDraft) {
        providerDraft = result.providerDraft
      }
      providerFeedback = 'Provider updated.'
    } else {
      providerError = result.error
    }
    providerSaving = false
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
</script>

<AgentsPageContent
  bind:activeTab
  bind:registerSheetOpen
  bind:providerConfigOpen
  {agents}
  {providers}
  {loading}
  {error}
  {pageFeedback}
  {pageError}
  {runtimeActionAgentId}
  {canRegister}
  {registerButtonTitle}
  onOpenRegister={() => handleRegisterOpenChange(true)}
  onSelectTicket={(ticketId) => {
    appStore.openRightPanel({ type: 'ticket', id: ticketId })
  }}
  onConfigureProvider={handleConfigureProvider}
  onPauseAgent={(agentId) => handleRuntimeAction('pause', agentId)}
  onResumeAgent={(agentId) => handleRuntimeAction('resume', agentId)}
  {providerItems}
  {registrationDraft}
  {registerSaving}
  {registerError}
  {registerFeedback}
  onRegistrationDraftChange={(field, value) => {
    registrationDraft = { ...registrationDraft, [field]: value }
  }}
  onRegisterAgent={handleRegisterAgent}
  onRegisterOpenChange={handleRegisterOpenChange}
  {selectedProvider}
  {providerDraft}
  {providerSaving}
  {providerFeedback}
  {providerError}
  onProviderDraftChange={(field, value) => {
    providerDraft = { ...providerDraft, [field]: value }
  }}
  onProviderSave={handleProviderSave}
/>
