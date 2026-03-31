<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider, Machine } from '$lib/api/contracts'
  import { listAgents, listMachines, listProviders, updateProject } from '$lib/api/openase'
  import {
    applyUpdatedProviderState,
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
  let machineItems = $state<Machine[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false)
  let loadError = $state('')
  let saving = $state(false)
  let selectedDefaultProviderId = $state('')
  let providerConfigOpen = $state(false)

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

        providerItems = providerPayload.providers
        providers = buildProviderOptions(providerPayload.providers, agentPayload.agents)
        providerCards = buildProviderCards(
          providerPayload.providers,
          agentPayload.agents,
          appStore.currentOrg?.default_agent_provider_id ?? null,
        )
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

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    providerItems = providerItems.map((p) => (p.id === updatedProvider.id ? updatedProvider : p))

    const nextState = applyUpdatedProviderState(providerCards, [], updatedProvider)
    providerCards = nextState.providers
    if (nextState.provider) providerEditor.open(nextState.provider)

    providers = buildProviderOptions(providerItems, [])
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
    <div class="text-muted-foreground text-sm">Loading agent settings…</div>
  {:else if loadError}
    <div class="text-destructive text-sm">{loadError}</div>
  {:else}
    <AgentSettingsDefaultsCard
      {providers}
      {selectedDefaultProviderId}
      orgDefaultProviderId={appStore.currentOrg?.default_agent_provider_id ?? null}
      orgDefaultProviderName={orgDefaultProvider?.name ?? null}
      {saving}
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
