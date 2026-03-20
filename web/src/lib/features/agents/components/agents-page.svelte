<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { listAgents, listProviders, listTickets, updateProvider } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { capabilityCatalog } from '$lib/features/capabilities'
  import { Button } from '$ui/button'
  import * as Tabs from '$ui/tabs'
  import { Plus } from '@lucide/svelte'
  import {
    applyUpdatedProviderState,
    buildAgentRows,
    buildProviderCards,
    createEmptyProviderDraft,
    parseProviderDraft,
    providerToDraft,
  } from '../model'
  import AgentList from './agent-list.svelte'
  import ProviderConfigSheet from './provider-config-sheet.svelte'
  import ProviderList from './provider-list.svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import type { AgentInstance, ProviderConfig, ProviderDraftField } from '../types'
  let activeTab = $state('instances')
  let agents = $state<AgentInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let loading = $state(false)
  let error = $state('')
  let providerConfigOpen = $state(false)
  let selectedProviderId = $state<string | null>(null)
  let providerDraft = $state(createEmptyProviderDraft())
  let providerSaving = $state(false)
  let providerFeedback = $state('')
  let providerError = $state('')
  let loadVersion = 0
  const agentRegistrationCapability = capabilityCatalog.agentRegistration
  const selectedProvider = $derived(
    providers.find((provider) => provider.id === selectedProviderId) ?? null,
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      providers = []
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
      providerFeedback = ''
      providerError = ''
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
      const [agentPayload, providerPayload, ticketPayload] = await Promise.all([
        listAgents(projectId),
        listProviders(orgId),
        listTickets(projectId),
      ])
      if (requestVersion !== loadVersion) return

      providers = buildProviderCards(
        providerPayload.providers,
        agentPayload.agents,
        appStore.currentOrg?.default_agent_provider_id ?? null,
      )
      agents = buildAgentRows(providerPayload.providers, ticketPayload.tickets, agentPayload.agents)
    } catch (caughtError) {
      if (requestVersion !== loadVersion) return
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agents.'
    } finally {
      if (requestVersion === loadVersion && showLoading) {
        loading = false
      }
    }
  }

  function resetProviderEditor() {
    providerConfigOpen = false
    selectedProviderId = null
    providerDraft = createEmptyProviderDraft()
    providerSaving = false
    providerFeedback = ''
    providerError = ''
  }

  function handleConfigureProvider(provider: ProviderConfig) {
    selectedProviderId = provider.id
    providerDraft = providerToDraft(provider)
    providerConfigOpen = true
    providerSaving = false
    providerFeedback = ''
    providerError = ''
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

  function applyUpdatedProvider(updatedProvider: AgentProvider) {
    const nextState = applyUpdatedProviderState(providers, agents, updatedProvider)
    providers = nextState.providers
    agents = nextState.agents
    if (nextState.provider) {
      providerDraft = providerToDraft(nextState.provider)
    }
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-foreground text-lg font-semibold">Agents</h1>
    <Button size="sm" disabled title={agentRegistrationCapability.summary}>
      <Plus class="size-3.5" />
      Register Agent
    </Button>
  </div>

  {#if loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-md border px-4 py-10 text-center text-sm"
    >
      Loading agents…
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <Tabs.Root bind:value={activeTab}>
      <Tabs.List variant="line">
        <Tabs.Trigger value="instances">Instances</Tabs.Trigger>
        <Tabs.Trigger value="providers">Providers</Tabs.Trigger>
      </Tabs.List>
      <Tabs.Content value="instances" class="pt-3">
        <AgentList
          {agents}
          onSelectTicket={(ticketId) => {
            appStore.openRightPanel({ type: 'ticket', id: ticketId })
          }}
        />
      </Tabs.Content>
      <Tabs.Content value="providers" class="pt-3">
        <ProviderList {providers} onConfigure={handleConfigureProvider} />
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</div>

<ProviderConfigSheet
  bind:open={providerConfigOpen}
  provider={selectedProvider}
  draft={providerDraft}
  saving={providerSaving}
  feedback={providerFeedback}
  error={providerError}
  onDraftChange={handleProviderDraftChange}
  onSave={handleProviderSave}
/>
