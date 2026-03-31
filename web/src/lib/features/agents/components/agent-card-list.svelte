<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Pencil, Pause, Play } from '@lucide/svelte'
  import type { AgentInstance, AgentRunInstance } from '../types'

  let {
    agents,
    runsByAgentId,
    runtimeActionAgentId = null,
    onSelectAgent,
    onSelectTicket,
    onPauseAgent,
    onResumeAgent,
  }: {
    agents: AgentInstance[]
    runsByAgentId: Map<string, AgentRunInstance[]>
    runtimeActionAgentId?: string | null
    onSelectAgent?: (agentId: string) => void
    onSelectTicket?: (ticketId: string) => void
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
  }

  const runtimeControlClasses: Record<AgentInstance['runtimeControlState'], string> = {
    active: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
    pause_requested: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300',
    paused: 'border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-300',
  }

  const runStatusDotColors: Record<AgentRunInstance['status'], string> = {
    launching: 'bg-amber-500',
    ready: 'bg-sky-500',
    executing: 'bg-blue-500',
    completed: 'bg-emerald-500',
    errored: 'bg-red-500',
    terminated: 'bg-slate-500',
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
    if (agent.runtimeControlState === 'pause_requested')
      return 'Pause requested. Waiting for the runtime to stop.'
    if (agent.runtimeControlState === 'paused') return 'Agent runtime is already paused.'
    if (agent.activeRunCount === 0) return 'This agent definition has no active AgentRuns to pause.'
    if (agent.status !== 'claimed' && agent.status !== 'running')
      return 'Only claimed or running agents can be paused.'
    return 'Pause this agent'
  }

  function resumeTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) return 'Updating runtime control...'
    if (agent.runtimeControlState === 'pause_requested')
      return 'Wait for the runtime to finish pausing before resuming.'
    if (agent.runtimeControlState !== 'paused') return 'Pause this agent before resuming it.'
    return 'Resume this agent'
  }

  function latestRun(agentId: string): AgentRunInstance | undefined {
    const runs = runsByAgentId.get(agentId)
    return runs?.[0]
  }

  function truncate(text: string, max: number): string {
    return text.length > max ? text.slice(0, max) + '…' : text
  }
</script>

{#if agents.length === 0}
  <div
    class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm"
  >
    No agent definitions registered yet. Register an agent to get started.
  </div>
{:else}
  <div class="space-y-3">
    {#each agents as agent (agent.id)}
      {@const run = latestRun(agent.id)}
      <div class="border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3">
        <!-- Left: agent info -->
        <button
          type="button"
          class="min-w-0 flex-1 text-left"
          onclick={() => onSelectAgent?.(agent.id)}
        >
          <div class="flex flex-wrap items-center gap-2">
            <span class={cn('size-2.5 shrink-0 rounded-full', statusColors[agent.status])}></span>
            <span class="text-foreground text-sm font-semibold hover:underline">{agent.name}</span>
            <span class="text-muted-foreground text-xs">{statusLabels[agent.status]}</span>
            {#if agent.runtimeControlState !== 'active'}
              <span
                class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium ${runtimeControlClasses[agent.runtimeControlState]}`}
              >
                {runtimeControlLabels[agent.runtimeControlState]}
              </span>
            {/if}
          </div>
          <div
            class="text-muted-foreground mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs"
          >
            <span>{agent.providerName} &middot; {agent.modelName}</span>
            <span>{agent.todayCompleted} completed today</span>
            <span>{formatCurrency(agent.todayCost)} spent</span>
            {#if agent.lastHeartbeat}
              <span>heartbeat {formatRelativeTime(agent.lastHeartbeat)}</span>
            {/if}
          </div>
        </button>

        <!-- Center: latest step / run snapshot -->
        <div class="hidden shrink-0 items-center gap-2 text-xs sm:flex">
          {#if agent.currentStepStatus}
            <span
              class="text-muted-foreground rounded-full border px-1.5 py-0.5 text-[10px] font-medium"
            >
              {agent.currentStepStatus}
            </span>
            {#if agent.currentStepSummary}
              <span class="text-muted-foreground max-w-40 truncate">
                {truncate(agent.currentStepSummary, 30)}
              </span>
            {/if}
            {#if agent.currentStepChangedAt}
              <span class="text-muted-foreground text-[11px]">
                {formatRelativeTime(agent.currentStepChangedAt)}
              </span>
            {/if}
          {:else if run}
            <span class={cn('size-2 shrink-0 rounded-full', runStatusDotColors[run.status])}></span>
            <button
              type="button"
              onclick={() => onSelectTicket?.(run.ticket.id)}
              class="text-primary shrink-0 hover:underline"
            >
              {run.ticket.identifier}
            </button>
            <span class="text-muted-foreground max-w-32 truncate">
              {truncate(run.ticket.title, 24)}
            </span>
            {#if run.lastHeartbeat}
              <span class="text-muted-foreground text-[11px]">
                {formatRelativeTime(run.lastHeartbeat)}
              </span>
            {/if}
            {#if run.lastError}
              <span class="text-[11px] text-red-600 dark:text-red-300" title={run.lastError}>
                err
              </span>
            {/if}
          {:else if agent.currentTicket}
            <button
              type="button"
              onclick={() => agent.currentTicket && onSelectTicket?.(agent.currentTicket.id)}
              class="text-primary shrink-0 hover:underline"
            >
              {agent.currentTicket.identifier}
            </button>
            <span class="text-muted-foreground max-w-32 truncate">
              {truncate(agent.currentTicket.title, 24)}
            </span>
          {/if}
        </div>

        <!-- Right: actions -->
        <div class="flex shrink-0 items-center gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label="Edit agent"
            title="Edit agent"
            onclick={() => onSelectAgent?.(agent.id)}
          >
            <Pencil class="size-3.5" />
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
      </div>
    {/each}
  </div>
{/if}
