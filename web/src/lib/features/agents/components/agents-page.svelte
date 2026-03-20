<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { listAgents, listProviders, listTickets } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { Button } from '$ui/button'
  import * as Tabs from '$ui/tabs'
  import { Plus } from '@lucide/svelte'
  import AgentList from './agent-list.svelte'
  import ProviderList from './provider-list.svelte'
  import type { AgentInstance, ProviderConfig } from '../types'
  let activeTab = $state('instances')
  let agents = $state<AgentInstance[]>([])
  let providers = $state<ProviderConfig[]>([])
  let loading = $state(false)
  let error = $state('')

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId || !orgId) {
      agents = []
      providers = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [agentPayload, providerPayload, ticketPayload] = await Promise.all([
          listAgents(projectId),
          listProviders(orgId),
          listTickets(projectId),
        ])
        if (cancelled) return

        const ticketMap = new Map(ticketPayload.tickets.map((ticket) => [ticket.id, ticket]))
        const providerMap = new Map(
          providerPayload.providers.map((provider) => [provider.id, provider]),
        )

        providers = providerPayload.providers.map((provider) => ({
          id: provider.id,
          name: provider.name,
          adapterType: provider.adapter_type,
          modelName: provider.model_name,
          agentCount: agentPayload.agents.filter((agent) => agent.provider_id === provider.id)
            .length,
          isDefault: appStore.currentOrg?.default_agent_provider_id === provider.id,
        }))

        agents = agentPayload.agents.map((agent) => {
          const provider = providerMap.get(agent.provider_id)
          const currentTicket = agent.current_ticket_id
            ? ticketMap.get(agent.current_ticket_id)
            : null

          return {
            id: agent.id,
            name: agent.name,
            providerName: provider?.name ?? 'Unknown provider',
            modelName: provider?.model_name ?? 'Unknown model',
            status: normalizeAgentStatus(agent.status),
            runtimePhase: normalizeRuntimePhase(agent.runtime_phase),
            currentTicket: currentTicket
              ? {
                  id: currentTicket.id,
                  identifier: currentTicket.identifier,
                  title: currentTicket.title,
                }
              : undefined,
            lastHeartbeat: agent.last_heartbeat_at,
            todayCompleted: agent.total_tickets_completed,
            todayCost: 0,
            capabilities: agent.capabilities,
          }
        })
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

  function normalizeAgentStatus(status: string): AgentInstance['status'] {
    if (
      status === 'idle' ||
      status === 'claimed' ||
      status === 'running' ||
      status === 'failed' ||
      status === 'terminated'
    ) {
      return status
    }

    return status === 'active' ? 'running' : 'idle'
  }

  function normalizeRuntimePhase(runtimePhase: string): AgentInstance['runtimePhase'] {
    if (
      runtimePhase === 'none' ||
      runtimePhase === 'launching' ||
      runtimePhase === 'ready' ||
      runtimePhase === 'failed'
    ) {
      return runtimePhase
    }

    return 'none'
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="text-foreground text-lg font-semibold">Agents</h1>
    <Button size="sm" disabled title="Agent registration is not exposed by the current API">
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
        <ProviderList {providers} />
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</div>
