<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { listAgents, listProviders, updateProject } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Separator } from '$ui/separator'
  import AgentSettingsDefaultsCard from './agent-settings-defaults-card.svelte'
  import { buildProviderOptions, parseDefaultProviderSelection } from './agent-settings-model'

  let providers = $state(buildProviderOptions([], []))
  let loading = $state(false)
  let loadError = $state('')
  let saving = $state(false)
  let selectedDefaultProviderId = $state('')

  const selectedDefaultProvider = $derived(
    providers.find((provider) => provider.id === selectedDefaultProviderId) ?? null,
  )
  const orgDefaultProvider = $derived(
    providers.find((provider) => provider.id === appStore.currentOrg?.default_agent_provider_id) ??
      null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id

    if (!projectId || !orgId) {
      providers = []
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
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Agents</h2>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">
      Default provider selection for this project. Manage agent inventory and runtime controls from
      the Agents page in the sidebar.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading agent settings…</div>
  {:else if loadError}
    <div class="text-destructive text-sm">{loadError}</div>
  {:else}
    <div class="max-w-lg">
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
  {/if}
</div>
