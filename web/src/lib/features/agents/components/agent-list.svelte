<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Terminal, Pause, Play } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let {
    agents,
    onSelectTicket,
    runtimeActionAgentId = null,
    onViewOutput,
    onPauseAgent,
    onResumeAgent,
  }: {
    agents: AgentInstance[]
    onSelectTicket?: (ticketId: string) => void
    runtimeActionAgentId?: string | null
    onViewOutput?: (agentId: string) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
  } = $props()

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    claimed: 'bg-amber-500',
    running: 'bg-blue-500',
    paused: 'bg-orange-500',
    failed: 'bg-red-500',
    terminated: 'bg-slate-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    claimed: 'Claimed',
    running: 'Running',
    paused: 'Paused',
    failed: 'Failed',
    terminated: 'Terminated',
  }

  const runtimeControlLabels: Record<AgentInstance['runtimeControlState'], string> = {
    active: 'Active',
    pause_requested: 'Pause Requested',
    paused: 'Paused',
    retired: 'Retired',
  }

  const runtimeControlClasses: Record<AgentInstance['runtimeControlState'], string> = {
    active: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
    pause_requested: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300',
    paused: 'border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-300',
    retired: 'border-zinc-500/30 bg-zinc-500/10 text-zinc-700 dark:text-zinc-300',
  }

  function canPause(agent: AgentInstance) {
    return (
      agent.runtimeControlState === 'active' &&
      agent.activeRunCount > 0 &&
      (agent.status === 'claimed' || agent.status === 'running')
    )
  }

  function canResume(agent: AgentInstance) {
    return agent.runtimeControlState === 'paused'
  }

  function pauseTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) return 'Updating runtime control...'
    if (agent.runtimeControlState === 'pause_requested') {
      return 'Pause requested. Waiting for the runtime to stop.'
    }
    if (agent.runtimeControlState === 'paused') return 'Agent runtime is already paused.'
    if (agent.activeRunCount === 0) return 'This agent definition has no active AgentRuns to pause.'
    if (agent.status !== 'claimed' && agent.status !== 'running') {
      return 'Only claimed or running agents can be paused.'
    }
    return 'Pause this agent'
  }

  function resumeTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) return 'Updating runtime control...'
    if (agent.runtimeControlState === 'pause_requested') {
      return 'Wait for the runtime to finish pausing before resuming.'
    }
    if (agent.runtimeControlState !== 'paused') return 'Pause this agent before resuming it.'
    return 'Resume this agent'
  }
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-border text-muted-foreground border-b text-left text-xs">
        <th class="pr-2 pb-2 pl-3 font-medium">Status</th>
        <th class="px-2 pb-2 font-medium">Agent</th>
        <th class="px-2 pb-2 font-medium">Runtime Summary</th>
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
              {#if agent.runtimeControlState !== 'active'}
                <span
                  class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${runtimeControlClasses[agent.runtimeControlState]}`}
                >
                  {runtimeControlLabels[agent.runtimeControlState]}
                </span>
              {/if}
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
            <div class="text-foreground text-xs tabular-nums">
              {agent.activeRunCount} active run(s)
            </div>
            {#if agent.currentTicket}
              <button
                type="button"
                onclick={() => agent.currentTicket && onSelectTicket?.(agent.currentTicket.id)}
                class="text-primary text-xs hover:underline"
              >
                {agent.currentTicket.identifier}
              </button>
              <div class="text-muted-foreground max-w-48 truncate text-xs">
                {agent.currentTicket.title}
              </div>
            {:else if agent.activeRunCount > 1}
              <div class="text-muted-foreground text-xs">See Runtime tab for concurrent runs.</div>
            {:else}
              <span class="text-muted-foreground/50 text-xs">&mdash;</span>
            {/if}
          </td>
          <td class="px-2 py-2.5">
            {#if agent.lastHeartbeat}
              <span class="text-muted-foreground text-xs">
                {formatRelativeTime(agent.lastHeartbeat)}
              </span>
            {:else}
              <span class="text-muted-foreground/50 text-xs">&mdash;</span>
            {/if}
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
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="View output"
                title="View agent output"
                onclick={() => onViewOutput?.(agent.id)}
              >
                <Terminal class="size-3.5" />
              </Button>
              {#if agent.runtimeControlState === 'paused'}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Resume agent"
                  disabled={!canResume(agent) || runtimeActionAgentId === agent.id}
                  title={resumeTitle(agent)}
                  onclick={() => onResumeAgent?.(agent.id)}
                >
                  <Play class="size-3.5" />
                </Button>
              {:else}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Pause agent"
                  disabled={!canPause(agent) || runtimeActionAgentId === agent.id}
                  title={pauseTitle(agent)}
                  onclick={() => onPauseAgent?.(agent.id)}
                >
                  <Pause class="size-3.5" />
                </Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
