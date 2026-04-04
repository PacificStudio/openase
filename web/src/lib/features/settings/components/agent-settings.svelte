<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { Agent, AgentProvider, Machine } from '$lib/api/contracts'
  import { listAgents, listMachines, listProviders, updateProject } from '$lib/api/openase'
  import { ProviderCreationDialog } from '$lib/features/catalog-creation'
  import {
    buildProviderCards,
    createProviderEditorState,
    ProviderConfigSheet,
    type ProviderConfig,
    type ProviderDraftField,
  } from '$lib/features/agents'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Separator } from '$ui/separator'
  import AgentSettingsDefaultsCard from './agent-settings-defaults-card.svelte'
  import { buildProviderOptions, parseDefaultProviderSelection } from './agent-settings-model'

  let providers = $state(buildProviderOptions([], []))
  let providerCards = $state<ProviderConfig[]>([])
  let agentItems = $state<Agent[]>([])
  let machineItems = $state<Machine[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false)
  let loadError = $state('')
  let saving = $state(false)
  let selectedDefaultProviderId = $state('')
  let providerConfigOpen = $state(false)
  let providerCreateOpen = $state(false)

  const providerEditor = createProviderEditorState()

  const orgDefaultProvider = $derived(
    providers.find((provider) => provider.id === appStore.currentOrg?.default_agent_provider_id) ??
      null,
  )
  const selectedProvider = $derived(
    providerCards.find((p) => p.id === providerEditor.selectedProviderId) ?? null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id

    if (!projectId || !orgId) {
      providers = []
      providerCards = []
      agentItems = []
      machineItems = []
      providerItems = []
      loading = false
      loadError = ''
      selectedDefaultProviderId = ''
      providerEditor.reset()
      return
    }

    selectedDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''

    let cancelled = false

    const load = async () => {
      loading = true
      loadError = ''

      try {
        const [providerPayload, agentPayload, machinePayload] = await Promise.all([
          listProviders(orgId),
          listAgents(projectId),
          listMachines(orgId),
        ])

        if (cancelled) return

        agentItems = agentPayload.agents
        syncProviderState(providerPayload.providers, agentPayload.agents)
        machineItems = machinePayload.machines
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
      const selectedName = providers.find((p) => p.id === parsed.value)?.name
      toastStore.success(
        parsed.value
          ? `Default agent provider set to ${selectedName ?? 'the selected provider'}.`
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

  function handleConfigureProvider(providerId: string) {
    const provider = providerCards.find((p) => p.id === providerId)
    if (!provider) return
    providerEditor.open(provider)
    providerConfigOpen = true
  }

  function handleProviderDraftChange(field: ProviderDraftField, value: string) {
    providerEditor.updateField(field, value)
  }

  function syncProviderState(
    nextProviderItems: AgentProvider[],
    nextAgentItems: Agent[] = agentItems,
  ) {
    providerItems = nextProviderItems
    providers = buildProviderOptions(nextProviderItems, nextAgentItems)
    providerCards = buildProviderCards(
      nextProviderItems,
      nextAgentItems,
      appStore.currentOrg?.default_agent_provider_id ?? null,
    )
  }

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    syncProviderState(
      providerItems.map((provider) =>
        provider.id === updatedProvider.id ? updatedProvider : provider,
      ),
    )
    const nextProvider = providerCards.find((provider) => provider.id === updatedProvider.id)
    if (nextProvider) {
      providerEditor.open(nextProvider)
    }
  }

  function handleProviderCreated(createdProvider: AgentProvider) {
    syncProviderState([...providerItems, createdProvider])
  }

  async function handleProviderSave() {
    await providerEditor.save(selectedProvider, applyUpdatedProvider)
  }
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Agents</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Configure providers and default routing for this project. Manage agent runtime controls from
      the Agents page in the sidebar.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="space-y-4">
      <div class="border-border bg-card rounded-lg border p-4">
        <div class="space-y-3">
          <div class="bg-muted h-4 w-40 animate-pulse rounded"></div>
          <div class="bg-muted h-3 w-64 animate-pulse rounded"></div>
          <div class="flex items-center gap-3 pt-1">
            <div class="bg-muted h-9 w-48 animate-pulse rounded-md"></div>
            <div class="bg-muted h-8 w-16 animate-pulse rounded-md"></div>
          </div>
        </div>
      </div>
      {#each { length: 2 } as _}
        <div class="border-border bg-card flex items-center gap-3 rounded-lg border p-4">
          <div class="bg-muted size-10 shrink-0 animate-pulse rounded-lg"></div>
          <div class="flex-1 space-y-1.5">
            <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
            <div class="bg-muted h-3 w-48 animate-pulse rounded"></div>
          </div>
          <div class="bg-muted h-5 w-16 animate-pulse rounded-full"></div>
        </div>
      {/each}
    </div>
  {:else if loadError}
    <div class="text-destructive text-sm">{loadError}</div>
  {:else}
    <AgentSettingsDefaultsCard
      {providers}
      {selectedDefaultProviderId}
      orgDefaultProviderId={appStore.currentOrg?.default_agent_provider_id ?? null}
      orgDefaultProviderName={orgDefaultProvider?.name ?? null}
      machineCount={machineItems.length}
      {saving}
      onAddProvider={() => {
        providerCreateOpen = true
      }}
      onSelectionChange={(value) => {
        selectedDefaultProviderId = value
      }}
      onSave={handleSaveDefaultProvider}
      onConfigure={handleConfigureProvider}
    />
  {/if}
</div>

<ProviderConfigSheet
  bind:open={providerConfigOpen}
  provider={selectedProvider}
  machines={machineItems}
  draft={providerEditor.draft}
  saving={providerEditor.saving}
  onDraftChange={handleProviderDraftChange}
  onSave={handleProviderSave}
/>

{#if appStore.currentOrg?.id}
  <ProviderCreationDialog
    orgId={appStore.currentOrg.id}
    bind:open={providerCreateOpen}
    title="Add provider"
    description="Register a provider for this organization without leaving the current project settings."
    onCreated={handleProviderCreated}
  />
{/if}
