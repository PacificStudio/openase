<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentsPageData } from '../data'
  import { loadAgentsPageResult } from '../page-data'
  import { createAgentRegistrationDraft } from '../registration'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance, AgentRunInstance } from '../types'
  import {
    markAgentsPageCacheDirty,
    readAgentsPageCache,
    writeAgentsPageCache,
  } from '../agents-page-cache'
  import AgentDrawer from './agent-drawer.svelte'
  import AgentsPageContent from './agents-page-content.svelte'
  import { registerAgentPageAction, runAgentRuntimePageAction } from './agents-page-actions'
  import { mapAgentsPageData } from './agents-page-helpers'
  import { connectAgentsPageStreams } from './agents-page-streams'

  let agents = $state<AgentInstance[]>([])
  let agentRuns = $state<AgentRunInstance[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false),
    error = $state('')
  let registerSheetOpen = $state(false)
  let registerSaving = $state(false)
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )
  let loadVersion = 0
  let activeLoadKey = ''
  let reloadQueued = false
  let reloadInFlight = false
  let runtimeActionAgentId = $state<string | null>(null)
  let agentDrawerOpen = $state(false)
  let selectedAgentId = $state<string | null>(null)

  const selectedAgent = $derived(agents.find((agent) => agent.id === selectedAgentId) ?? null)

  $effect(() => {
    const projectId = appStore.currentProject?.id,
      orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      agentRuns = []
      providerItems = []
      loading = false
      error = ''
      resetRegistrationDraft()
      return
    }

    const loadKey = `${projectId}:${orgId}`
    activeLoadKey = loadKey
    reloadQueued = false
    reloadInFlight = false
    const cached = readAgentsPageCache(projectId, orgId)
    if (cached) {
      applyPageStateSnapshot(cached.snapshot, { persist: false })
      loading = false
      error = ''
      if (cached.dirty) {
        void requestReload({ projectId, orgId, loadKey })
      }
    } else {
      void loadData({ projectId, orgId, loadKey, showLoading: true })
    }

    const disconnect = connectAgentsPageStreams(projectId, orgId, () => {
      markAgentsPageCacheDirty(projectId, orgId)
      if (registerSheetOpen) return
      void requestReload({ projectId, orgId, loadKey })
    })

    return () => {
      loadVersion += 1
      if (activeLoadKey === loadKey) {
        activeLoadKey = ''
      }
      reloadQueued = false
      reloadInFlight = false
      disconnect()
    }
  })

  async function loadData(input: {
    projectId: string
    orgId: string
    loadKey: string
    showLoading: boolean
  }) {
    const requestVersion = ++loadVersion
    if (input.showLoading) loading = true
    if (input.showLoading) error = ''

    const result = await loadAgentsPageResult({
      projectId: input.projectId,
      orgId: input.orgId,
      defaultProviderId: appStore.currentOrg?.default_agent_provider_id ?? null,
    })
    if (requestVersion !== loadVersion || activeLoadKey !== input.loadKey) return

    if (result.ok) {
      applyPageData(result.data)
      error = ''
    } else {
      if (input.showLoading || agents.length === 0) {
        error = result.error
      } else {
        toastStore.error(result.error)
      }
    }
    if (input.showLoading) loading = false
  }

  function applyPageData(data: AgentsPageData) {
    applyPageStateSnapshot(mapAgentsPageData(data))
  }

  function applyPageStateSnapshot(
    snapshot: {
      providerItems: AgentProvider[]
      agents: AgentInstance[]
      agentRuns: AgentRunInstance[]
    },
    options: { persist?: boolean } = {},
  ) {
    providerItems = snapshot.providerItems
    agents = snapshot.agents
    agentRuns = snapshot.agentRuns
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (options.persist !== false && projectId && orgId) {
      writeAgentsPageCache(projectId, orgId, snapshot)
    }
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

  async function handleRuntimeAction(action: 'interrupt' | 'pause' | 'resume', agentId: string) {
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

  async function requestReload(input: { projectId: string; orgId: string; loadKey: string }) {
    reloadQueued = true
    if (reloadInFlight) {
      return
    }

    while (reloadQueued && activeLoadKey === input.loadKey) {
      reloadQueued = false
      reloadInFlight = true
      try {
        await loadData({ ...input, showLoading: false })
      } finally {
        reloadInFlight = false
      }
    }
  }
</script>

<AgentsPageContent
  bind:registerSheetOpen
  canRegister={!!appStore.currentProject?.id && providerItems.length > 0}
  {agents}
  {agentRuns}
  {loading}
  {error}
  {runtimeActionAgentId}
  registerButtonTitle={providerItems.length === 0
    ? 'Register a provider before creating agents.'
    : appStore.currentProject?.id
      ? undefined
      : 'Project context is unavailable.'}
  onOpenRegister={() => handleRegisterOpenChange(true)}
  onSelectAgent={(agentId) => {
    selectedAgentId = agentId
    agentDrawerOpen = true
  }}
  onSelectTicket={(ticketId) => appStore.openRightPanel({ type: 'ticket', id: ticketId })}
  onInterruptAgent={(agentId) => handleRuntimeAction('interrupt', agentId)}
  onPauseAgent={(agentId) => handleRuntimeAction('pause', agentId)}
  onResumeAgent={(agentId) => handleRuntimeAction('resume', agentId)}
  {providerItems}
  {registrationDraft}
  currentOrgSlug={appStore.currentOrg?.slug}
  currentProjectSlug={appStore.currentProject?.slug}
  {registerSaving}
  onRegistrationDraftChange={updateRegistrationDraft}
  onRegisterAgent={handleRegisterAgent}
  onRegisterOpenChange={handleRegisterOpenChange}
/>

<AgentDrawer
  bind:open={agentDrawerOpen}
  agent={selectedAgent}
  providers={providerItems}
  onOpenChange={(open) => {
    agentDrawerOpen = open
    if (!open) selectedAgentId = null
  }}
  onDeleted={() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (projectId && orgId) {
      void loadData({
        projectId,
        orgId,
        loadKey: activeLoadKey || `${projectId}:${orgId}`,
        showLoading: false,
      })
    }
  }}
  onUpdated={() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (projectId && orgId) {
      void loadData({
        projectId,
        orgId,
        loadKey: activeLoadKey || `${projectId}:${orgId}`,
        showLoading: false,
      })
    }
  }}
/>
