<script lang="ts">
  import * as Tabs from '$ui/tabs'
  import type { AgentInstance, AgentRunInstance, ProviderConfig } from '../types'
  import AgentRunList from './agent-run-list.svelte'
  import AgentList from './agent-list.svelte'
  import ProviderList from './provider-list.svelte'

  let {
    activeTab = $bindable('runtime'),
    agents,
    agentRuns,
    providers,
    loading = false,
    error = '',
    runtimeActionAgentId = null,
    onSelectTicket,
    onViewOutput,
    onConfigureProvider,
    onPauseAgent,
    onResumeAgent,
  }: {
    activeTab?: string
    agents: AgentInstance[]
    agentRuns: AgentRunInstance[]
    providers: ProviderConfig[]
    loading?: boolean
    error?: string
    runtimeActionAgentId?: string | null
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
    onConfigureProvider?: (provider: ProviderConfig) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
  } = $props()
</script>

<div class="border-border/60 bg-card/60 space-y-4 rounded-xl border p-4 sm:p-5">
  {#if loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-md border px-4 py-10 text-center text-sm"
    >
      Loading agents...
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <Tabs.Root bind:value={activeTab}>
      <Tabs.List variant="line" class="px-1">
        <Tabs.Trigger value="runtime">Runtime</Tabs.Trigger>
        <Tabs.Trigger value="definitions">Definitions</Tabs.Trigger>
        <Tabs.Trigger value="providers">Providers</Tabs.Trigger>
      </Tabs.List>
      <Tabs.Content value="runtime" class="pt-3">
        <AgentRunList
          {agentRuns}
          onSelectTicket={(ticketId) => onSelectTicket?.(ticketId)}
          onViewOutput={(agentId) => onViewOutput?.(agentId)}
        />
      </Tabs.Content>
      <Tabs.Content value="definitions" class="pt-3">
        <AgentList
          {agents}
          {runtimeActionAgentId}
          onSelectTicket={(ticketId) => onSelectTicket?.(ticketId)}
          onViewOutput={(agentId) => onViewOutput?.(agentId)}
          {onPauseAgent}
          {onResumeAgent}
        />
      </Tabs.Content>
      <Tabs.Content value="providers" class="pt-3">
        <ProviderList {providers} onConfigure={onConfigureProvider} />
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</div>
