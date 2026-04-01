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
    <!-- Skeleton: summary line -->
    <div class="px-1">
      <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
    </div>
    <!-- Skeleton: agent cards -->
    <div class="space-y-2">
      {#each { length: 3 } as _}
        <div class="border-border bg-card flex items-center gap-3 rounded-lg border px-4 py-3">
          <div class="bg-muted size-8 shrink-0 animate-pulse rounded-full"></div>
          <div class="min-w-0 flex-1 space-y-2">
            <div class="flex items-center gap-2">
              <div class="bg-muted h-4 w-28 animate-pulse rounded"></div>
              <div class="bg-muted h-4 w-16 animate-pulse rounded-full"></div>
            </div>
            <div class="flex items-center gap-3">
              <div class="bg-muted h-3 w-24 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5">
            <div class="bg-muted size-7 animate-pulse rounded"></div>
            <div class="bg-muted size-7 animate-pulse rounded"></div>
          </div>
        </div>
      {/each}
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
