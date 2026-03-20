<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { createAgent, listAgents, listProviders, listTickets } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { Button } from '$ui/button'
  import * as Tabs from '$ui/tabs'
  import { Plus } from '@lucide/svelte'
  import AgentList from './agent-list.svelte'
  import AgentRegistrationSheet from './agent-registration-sheet.svelte'
  import ProviderList from './provider-list.svelte'
  import type { AgentPayload, AgentProvider, Ticket } from '$lib/api/contracts'
  import { buildAgentRows, buildProviderCards } from '../data'
  import {
    createAgentRegistrationDraft,
    parseAgentRegistrationDraft,
    type AgentRegistrationDraft,
    type AgentRegistrationDraftField,
  } from '../model'
  import type { AgentInstance, ProviderConfig } from '../types'
  let activeTab = $state('instances')
  let agents = $state<AgentInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let providerItems = $state<AgentProvider[]>([])
  let loading = $state(false)
  let error = $state('')
  let registerSheetOpen = $state(false)
  let registerSaving = $state(false)
  let registerError = $state('')
  let registerFeedback = $state('')
  let pageFeedback = $state('')
  let registrationDraft = $state<AgentRegistrationDraft>(
    createAgentRegistrationDraft([], appStore.currentOrg?.default_agent_provider_id),
  )

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      providers = []
      providerItems = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const snapshot = await loadAgentSnapshot(projectId, orgId)
        if (cancelled) return

        applySnapshot(snapshot)
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agents.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
      onEvent: () => {
        void load()
      },
      onError: (streamError) => {
        console.error('Agents stream error:', streamError)
      },
    })

    return () => {
      cancelled = true
      disconnect()
    }
  })

  function updateRegistrationDraft(field: AgentRegistrationDraftField, value: string) {
    registrationDraft = {
      ...registrationDraft,
      [field]: value,
    }
  }

  function resetRegistrationDraft() {
    registrationDraft = createAgentRegistrationDraft(
      providerItems,
      appStore.currentOrg?.default_agent_provider_id,
    )
    registerError = ''
    registerFeedback = ''
  }

  function handleRegisterOpenChange(open: boolean) {
    registerSheetOpen = open
    if (open) {
      resetRegistrationDraft()
      pageFeedback = ''
      return
    }

    registerError = ''
    registerFeedback = ''
  }

  async function handleRegisterAgent() {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
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

    try {
      await createAgent(projectId, {
        provider_id: parsed.value.providerId,
        name: parsed.value.name,
        workspace_path: parsed.value.workspacePath,
        capabilities: parsed.value.capabilities,
      })

      registerFeedback = 'Agent created. Refreshing list…'
      const snapshot = await loadAgentSnapshot(projectId, orgId)
      applySnapshot(snapshot)
      pageFeedback = `Registered ${parsed.value.name}.`
      registerSheetOpen = false
      resetRegistrationDraft()
    } catch (caughtError) {
      registerError =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to register agent.'
    } finally {
      registerSaving = false
    }
  }

  async function loadAgentSnapshot(projectId: string, orgId: string) {
    const [agentPayload, providerPayload, ticketPayload] = await Promise.all([
      listAgents(projectId),
      listProviders(orgId),
      listTickets(projectId),
    ])

    return {
      agentPayload,
      providerPayload,
      ticketPayload,
    }
  }

  function applySnapshot(snapshot: {
    agentPayload: AgentPayload
    providerPayload: { providers: AgentProvider[] }
    ticketPayload: { tickets: Ticket[] }
  }) {
    providerItems = snapshot.providerPayload.providers
    providers = buildProviderCards(snapshot.providerPayload.providers, snapshot.agentPayload.agents)
    agents = buildAgentRows(
      snapshot.providerPayload.providers,
      snapshot.ticketPayload.tickets,
      snapshot.agentPayload.agents,
    )
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-foreground text-lg font-semibold">Agents</h1>
    <Button
      size="sm"
      onclick={() => handleRegisterOpenChange(true)}
      disabled={!appStore.currentProject?.id || providerItems.length === 0}
      title={providerItems.length === 0
        ? 'Register a provider before creating agents.'
        : appStore.currentProject?.id
          ? undefined
          : 'Project context is unavailable.'}
    >
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
    {#if pageFeedback}
      <div
        class="rounded-md border border-emerald-500/40 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-300"
      >
        {pageFeedback}
      </div>
    {/if}

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
        <ProviderList {providers} />
      </Tabs.Content>
    </Tabs.Root>
  {/if}

  <AgentRegistrationSheet
    bind:open={registerSheetOpen}
    providers={providerItems}
    draft={registrationDraft}
    saving={registerSaving}
    error={registerError}
    feedback={registerFeedback}
    onDraftChange={updateRegistrationDraft}
    onSubmit={handleRegisterAgent}
    onOpenChange={handleRegisterOpenChange}
  />
</div>
