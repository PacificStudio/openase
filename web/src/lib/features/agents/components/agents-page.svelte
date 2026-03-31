<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentsPageData } from '../data'
  import { loadAgentsPageResult } from '../page-data'
  import { createAgentRegistrationDraft } from '../registration'
  import type { AgentRegistrationDraft, AgentRegistrationDraftField } from '../registration'
  import type { AgentInstance, AgentRunInstance } from '../types'
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
      resetRegistrationDraft()
      return
    }

    void loadData({ projectId, orgId, showLoading: true })
    const disconnect = connectAgentsPageStreams(projectId, orgId, () => {
      void loadData({ projectId, orgId, showLoading: false })
    })

    return () => {
      loadVersion += 1
      disconnect()
    }
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
    const mapped = mapAgentsPageData(data)
    providerItems = mapped.providerItems
    agents = mapped.agents
    agentRuns = mapped.agentRuns
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
  onOpenChange={(open) => {
    agentDrawerOpen = open
    if (!open) selectedAgentId = null
  }}
  onDeleted={() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (projectId && orgId) {
      void loadData({ projectId, orgId, showLoading: false })
    }
  }}
  onEditProvider={() => {
    agentDrawerOpen = false
    selectedAgentId = null
    const orgSlug = appStore.currentOrg?.slug
    const projectSlug = appStore.currentProject?.slug
    if (orgSlug && projectSlug) {
      window.location.hash = ''
      window.location.href = `/orgs/${orgSlug}/projects/${projectSlug}/settings#agents`
    }
  }}
/>
