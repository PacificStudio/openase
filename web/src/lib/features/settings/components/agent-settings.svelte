<script lang="ts">
  import { page } from '$app/state'
  import { ApiError } from '$lib/api/client'
  import { deleteAgent, listAgents, listProviders, updateProject } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Separator } from '$ui/separator'
  import AgentSettingsDefaultsCard from './agent-settings-defaults-card.svelte'
  import AgentSettingsInventory from './agent-settings-inventory.svelte'
  import AgentSettingsOverview from './agent-settings-overview.svelte'
  import {
    buildGovernanceAgents,
    buildProviderOptions,
    parseDefaultProviderSelection,
    type GovernanceAgent,
  } from './agent-settings-model'

  const agentsConsoleHref = $derived(`/agents${page.url.search}`)

  let providers = $state(buildProviderOptions([], []))
  let agents = $state<GovernanceAgent[]>([])
  let loading = $state(false)
  let loadError = $state('')
  let saving = $state(false)
  let deletingAgentId = $state<string | null>(null)
  let selectedDefaultProviderId = $state('')

  const selectedDefaultProvider = $derived(
    providers.find((provider) => provider.id === selectedDefaultProviderId) ?? null,
  )
  const orgDefaultProvider = $derived(
    providers.find((provider) => provider.id === appStore.currentOrg?.default_agent_provider_id) ??
      null,
  )
  const runningAgents = $derived(
    agents.filter((agent) => agent.status === 'running' || agent.status === 'claimed').length,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id

    if (!projectId || !orgId) {
      providers = []
      agents = []
      loading = false
      loadError = ''
      selectedDefaultProviderId = ''
      return
    }

    selectedDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''

    let cancelled = false

    const load = async () => {
      loading = true
      loadError = ''

      try {
        const [providerPayload, agentPayload] = await Promise.all([
          listProviders(orgId),
          listAgents(projectId),
        ])

        if (cancelled) return

        providers = buildProviderOptions(providerPayload.providers, agentPayload.agents)
        agents = buildGovernanceAgents(agentPayload.agents, providerPayload.providers)
      } catch (caughtError) {
        if (cancelled) return
        loadError =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agent settings.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  async function handleSaveDefaultProvider() {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    const parsed = parseDefaultProviderSelection(selectedDefaultProviderId, providers)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    saving = true

    try {
      const payload = await updateProject(projectId, {
        default_agent_provider_id: parsed.value,
      })
      appStore.currentProject = payload.project
      toastStore.success(
        parsed.value
          ? `Default agent provider set to ${selectedDefaultProvider?.name ?? 'the selected provider'}.`
          : 'Project now inherits the organization default provider.',
      )
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save default provider.',
      )
    } finally {
      saving = false
    }
  }

  async function reloadInventory(orgId: string, projectId: string) {
    const [providerPayload, agentPayload] = await Promise.all([
      listProviders(orgId),
      listAgents(projectId),
    ])

    providers = buildProviderOptions(providerPayload.providers, agentPayload.agents)
    agents = buildGovernanceAgents(agentPayload.agents, providerPayload.providers)
  }

  async function handleDeleteAgent(agent: GovernanceAgent) {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    deletingAgentId = agent.id

    try {
      await deleteAgent(agent.id)
      await reloadInventory(orgId, projectId)
      toastStore.success(`Deleted agent "${agent.name}".`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete agent.',
      )
    } finally {
      deletingAgentId = null
    }
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Agents</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Manage default provider selection, registered agent inventory, and agent governance.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading agent governance settings…</div>
  {:else if loadError}
    <div class="text-destructive text-sm">{loadError}</div>
  {:else}
    <AgentSettingsOverview
      selectedDefaultProviderName={selectedDefaultProvider?.name ?? null}
      orgDefaultProviderName={orgDefaultProvider?.name ?? null}
      agentCount={agents.length}
      runningAgentCount={runningAgents}
    />

    <div class="grid gap-6 xl:grid-cols-[minmax(0,22rem),minmax(0,1fr)]">
      <div class="space-y-6">
        <AgentSettingsDefaultsCard
          {providers}
          {selectedDefaultProviderId}
          selectedDefaultProviderName={selectedDefaultProvider?.name ?? null}
          orgDefaultProviderId={appStore.currentOrg?.default_agent_provider_id ?? null}
          orgDefaultProviderName={orgDefaultProvider?.name ?? null}
          {saving}
          onSelectionChange={(value) => {
            selectedDefaultProviderId = value
          }}
          onSave={handleSaveDefaultProvider}
        />
      </div>

      <AgentSettingsInventory
        {agents}
        {agentsConsoleHref}
        {deletingAgentId}
        onDelete={handleDeleteAgent}
      />
    </div>
  {/if}
</div>
