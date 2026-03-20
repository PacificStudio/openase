<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Terminal, Pause, Play } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let { agents }: { agents: AgentInstance[] } = $props()

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    claimed: 'bg-sky-500',
    running: 'bg-blue-500',
    failed: 'bg-red-500',
    terminated: 'bg-slate-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    claimed: 'Claimed',
    running: 'Running',
    failed: 'Failed',
    terminated: 'Terminated',
  }

  const runtimePhaseClasses: Record<AgentInstance['runtimePhase'], string> = {
    none: 'border-border bg-background text-muted-foreground',
    launching: 'border-sky-500/25 bg-sky-500/10 text-sky-700',
    ready: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700',
    failed: 'border-rose-500/25 bg-rose-500/10 text-rose-700',
  }

  function runtimeLabel(agent: AgentInstance): string {
    if (agent.status === 'failed' || agent.runtimePhase === 'failed') {
      return 'Launch failed'
    }
    if (agent.status === 'terminated') {
      return 'Terminated'
    }
    if (agent.runtimePhase === 'launching') {
      return 'Launching Codex session'
    }
    if (agent.status === 'claimed' && agent.runtimePhase === 'none') {
      return 'Claimed, waiting for launcher'
    }
    if (agent.status === 'running' && agent.runtimePhase === 'ready' && agent.sessionId) {
      return 'Ready'
    }
    if (agent.status === 'running') {
      return 'Running'
    }
    if (agent.status === 'claimed') {
      return 'Claimed'
    }
    return statusLabels[agent.status]
  }

  function heartbeatLabel(value?: string | null): string {
    if (!value) {
      return 'No heartbeat yet'
    }

    return formatRelativeTime(value)
  }
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-border text-muted-foreground border-b text-left text-xs">
        <th class="pr-2 pb-2 pl-3 font-medium">Status</th>
        <th class="px-2 pb-2 font-medium">Agent</th>
        <th class="px-2 pb-2 font-medium">Runtime</th>
        <th class="px-2 pb-2 font-medium">Current Ticket</th>
        <th class="px-2 pb-2 font-medium">Last Heartbeat</th>
        <th class="px-2 pb-2 text-right font-medium">Completed</th>
        <th class="px-2 pb-2 text-right font-medium">Cost</th>
        <th class="pr-3 pb-2 pl-2 text-right font-medium">Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each agents as agent (agent.id)}
        <tr class="group border-border/50 hover:bg-muted/30 border-b transition-colors">
          <td class="py-2.5 pr-2 pl-3">
            <div class="flex items-center gap-2">
              <span class={cn('size-2 rounded-full', statusColors[agent.status])}></span>
              <span class="text-muted-foreground text-xs">{statusLabels[agent.status]}</span>
            </div>
          </td>
          <td class="px-2 py-2.5">
            <div class="flex items-center gap-2">
              <span class="text-foreground font-medium">{agent.name}</span>
              <Badge variant="secondary" class="text-[10px]">{agent.providerName}</Badge>
            </div>
            <div class="text-muted-foreground text-xs">{agent.modelName}</div>
          </td>
          <td class="px-2 py-2.5">
            <div class="flex flex-col items-start gap-1">
              <Badge class={runtimePhaseClasses[agent.runtimePhase]}>{runtimeLabel(agent)}</Badge>
              {#if agent.sessionId}
                <span class="text-muted-foreground font-mono text-xs">{agent.sessionId}</span>
              {:else if agent.lastError}
                <span class="max-w-56 truncate text-xs text-rose-700">{agent.lastError}</span>
              {/if}
            </div>
          </td>
          <td class="px-2 py-2.5">
            {#if agent.currentTicket}
              <a
                href="/tickets/{agent.currentTicket.id}"
                class="text-primary text-xs hover:underline"
              >
                {agent.currentTicket.identifier}
              </a>
              <div class="text-muted-foreground max-w-48 truncate text-xs">
                {agent.currentTicket.title}
              </div>
            {:else}
              <span class="text-muted-foreground/50 text-xs">&mdash;</span>
            {/if}
          </td>
          <td class="px-2 py-2.5">
            <span class="text-muted-foreground text-xs">
              {heartbeatLabel(agent.lastHeartbeat)}
            </span>
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-foreground text-xs tabular-nums">{agent.todayCompleted}</span>
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-foreground text-xs tabular-nums">
              {formatCurrency(agent.todayCost)}
            </span>
          </td>
          <td class="py-2.5 pr-3 pl-2">
            <div
              class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100"
            >
              <Button variant="ghost" size="icon-xs" aria-label="View output">
                <Terminal class="size-3.5" />
              </Button>
              {#if agent.status === 'running'}
                <Button variant="ghost" size="icon-xs" aria-label="Pause agent">
                  <Pause class="size-3.5" />
                </Button>
              {:else if agent.status === 'idle' || agent.status === 'claimed'}
                <Button variant="ghost" size="icon-xs" aria-label="Resume agent">
                  <Play class="size-3.5" />
                </Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
