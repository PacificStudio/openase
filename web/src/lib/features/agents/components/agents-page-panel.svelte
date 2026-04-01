<script lang="ts">
  import type { AgentInstance, AgentRunInstance } from '../types'
  import AgentCardList from './agent-card-list.svelte'

  let {
    agents,
    agentRuns,
    loading = false,
    error = '',
    runtimeActionAgentId = null,
    onSelectAgent,
    onSelectTicket,
    onPauseAgent,
    onResumeAgent,
  }: {
    agents: AgentInstance[]
    agentRuns: AgentRunInstance[]
    loading?: boolean
    error?: string
    runtimeActionAgentId?: string | null
    onSelectAgent?: (agentId: string) => void
    onSelectTicket?: (ticketId: string) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
  } = $props()

  const runsByAgentId = $derived(
    agentRuns.reduce<Map<string, AgentRunInstance[]>>((map, run) => {
      const list = map.get(run.agentId)
      if (list) {
        list.push(run)
      } else {
        map.set(run.agentId, [run])
      }
      return map
    }, new Map()),
  )

  const totalRunning = $derived(
    agents.filter((a) => a.status === 'running' || a.status === 'claimed').length,
  )
</script>

<div class="space-y-4">
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
    {#if agents.length > 0}
      <div class="text-muted-foreground px-1 text-sm">
        <span class="text-foreground font-medium">{totalRunning}/{agents.length}</span> agents running
      </div>
    {/if}

    <AgentCardList
      {agents}
      {runsByAgentId}
      {runtimeActionAgentId}
      {onSelectAgent}
      {onSelectTicket}
      {onPauseAgent}
      {onResumeAgent}
    />
  {/if}
</div>
