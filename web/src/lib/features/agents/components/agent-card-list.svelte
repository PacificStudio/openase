<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as Tooltip from '$ui/tooltip'
  import {
    Hand,
    Pause,
    Play,
    ChevronDown,
    ChevronRight,
    Pencil,
    TerminalSquare,
    Bot,
    Sparkles,
    Cpu,
    CircleX,
  } from '@lucide/svelte'
  import type { AgentInstance, AgentRunInstance } from '../types'

  let {
    agents,
    runsByAgentId,
    runtimeActionAgentId = null,
    onSelectAgent,
    onSelectTicket,
    onInterruptAgent,
    onPauseAgent,
    onResumeAgent,
  }: {
    agents: AgentInstance[]
    runsByAgentId: Map<string, AgentRunInstance[]>
    runtimeActionAgentId?: string | null
    onSelectAgent?: (agentId: string) => void
    onSelectTicket?: (ticketId: string) => void
    onInterruptAgent?: (agentId: string) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
  } = $props()

  let expandedIds = $state<Set<string>>(new Set())

  function toggleExpand(agentId: string) {
    const next = new Set(expandedIds)
    if (next.has(agentId)) {
      next.delete(agentId)
    } else {
      next.add(agentId)
    }
    expandedIds = next
  }

  const adapterIcons: Record<string, typeof TerminalSquare> = {
    'claude-code-cli': TerminalSquare,
    'codex-app-server': Bot,
    'gemini-cli': Sparkles,
    custom: Cpu,
  }

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'text-emerald-500',
    claimed: 'text-amber-500',
    running: 'text-blue-500',
    paused: 'text-orange-500',
    failed: 'text-red-500',
    interrupted: 'text-rose-500',
    terminated: 'text-slate-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    claimed: 'Claimed',
    running: 'Running',
    paused: 'Paused',
    failed: 'Failed',
    interrupted: 'Interrupted',
    terminated: 'Terminated',
  }

  const runStatusColors: Record<AgentRunInstance['status'], string> = {
    launching: 'bg-amber-500',
    ready: 'bg-sky-500',
    executing: 'bg-blue-500',
    completed: 'bg-emerald-500',
    errored: 'bg-red-500',
    interrupted: 'bg-rose-500',
    terminated: 'bg-slate-500',
  }

  function canInterrupt(agent: AgentInstance) {
    return (
      agent.runtimeControlState === 'active' &&
      agent.activeRunCount > 0 &&
      (agent.status === 'claimed' || agent.status === 'running')
    )
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

  function activeRuns(agentId: string): AgentRunInstance[] {
    const runs = runsByAgentId.get(agentId)
    if (!runs) return []
    return runs.filter(
      (r) => r.status === 'launching' || r.status === 'ready' || r.status === 'executing',
    )
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
  <div class="space-y-2">
    {#each agents as agent (agent.id)}
      {@const Icon = adapterIcons[agent.adapterType] ?? Cpu}
      {@const runs = activeRuns(agent.id)}
      {@const expanded = expandedIds.has(agent.id)}
      {@const hasRuns = runs.length > 0}

      <div class="border-border/60 bg-card/60 rounded-xl border">
        <div class="flex items-center gap-3 px-4 py-2.5">
          <button
            type="button"
            class="flex shrink-0 items-center"
            onclick={() => hasRuns && toggleExpand(agent.id)}
            disabled={!hasRuns}
            aria-label={expanded ? 'Collapse' : 'Expand'}
          >
            {#if hasRuns}
              {#if expanded}
                <ChevronDown class="text-muted-foreground size-3.5" />
              {:else}
                <ChevronRight class="text-muted-foreground size-3.5" />
              {/if}
            {:else}
              <span class="size-3.5"></span>
            {/if}
          </button>

          <Icon class={cn('size-4 shrink-0', statusColors[agent.status])} />

          <button
            type="button"
            class="text-foreground min-w-0 shrink truncate text-sm font-semibold hover:underline"
            onclick={() => onSelectAgent?.(agent.id)}
          >
            {agent.name}
          </button>

          <span class="text-muted-foreground text-xs whitespace-nowrap">
            {statusLabels[agent.status]} · {agent.activeRunCount} task{agent.activeRunCount !== 1
              ? 's'
              : ''}
          </span>

          {#if agent.runtimeControlState !== 'active'}
            <span
              class={cn(
                'inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium whitespace-nowrap',
                agent.runtimeControlState === 'pause_requested'
                  ? 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300'
                  : agent.runtimeControlState === 'interrupt_requested'
                    ? 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300'
                    : agent.runtimeControlState === 'retired'
                      ? 'border-zinc-500/30 bg-zinc-500/10 text-zinc-700 dark:text-zinc-300'
                      : 'border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-300',
              )}
            >
              {agent.runtimeControlState === 'interrupt_requested'
                ? 'Interrupt Requested'
                : agent.runtimeControlState === 'pause_requested'
                  ? 'Pause Requested'
                  : agent.runtimeControlState === 'retired'
                    ? 'Retired'
                    : 'Paused'}
            </span>
          {/if}

          <span class="flex-1"></span>

          {#if agent.lastHeartbeat}
            <span class="text-muted-foreground hidden text-[11px] whitespace-nowrap sm:inline">
              heartbeat {formatRelativeTime(agent.lastHeartbeat)}
            </span>
          {/if}

          <div class="flex shrink-0 items-center gap-0.5">
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Edit agent"
              title="Edit agent"
              onclick={() => onSelectAgent?.(agent.id)}
            >
              <Pencil class="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Interrupt agent"
              disabled={!canInterrupt(agent) || runtimeActionAgentId === agent.id}
              title="Interrupt this agent run"
              onclick={() => onInterruptAgent?.(agent.id)}
            >
              <Hand class="size-3.5" />
            </Button>
            {#if agent.runtimeControlState === 'paused'}
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="Resume agent"
                disabled={!canResume(agent) || runtimeActionAgentId === agent.id}
                title="Resume this agent"
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
                title="Pause this agent"
                onclick={() => onPauseAgent?.(agent.id)}
              >
                <Pause class="size-3.5" />
              </Button>
            {/if}
          </div>
        </div>

        {#if expanded && hasRuns}
          <div class="border-border border-t px-4 py-2">
            <div class="space-y-1.5">
              {#each runs as run (run.id)}
                <div class="flex items-center gap-2 text-xs">
                  <span class={cn('size-2 shrink-0 rounded-full', runStatusColors[run.status])}
                  ></span>
                  <button
                    type="button"
                    class="text-primary shrink-0 font-mono hover:underline"
                    onclick={() => onSelectTicket?.(run.ticket.id)}
                  >
                    {run.ticket.identifier}
                  </button>
                  <span class="text-muted-foreground min-w-0 truncate">
                    {truncate(run.ticket.title, 40)}
                  </span>
                  <span class="text-muted-foreground/60 text-[10px] whitespace-nowrap">
                    {run.status}
                  </span>
                  {#if run.lastHeartbeat}
                    <span class="text-muted-foreground text-[10px] whitespace-nowrap">
                      {formatRelativeTime(run.lastHeartbeat)}
                    </span>
                  {/if}
                  {#if run.lastError}
                    <Tooltip.Provider>
                      <Tooltip.Root>
                        <Tooltip.Trigger class="inline-flex text-red-500">
                          <CircleX class="size-3" />
                        </Tooltip.Trigger>
                        <Tooltip.Portal>
                          <Tooltip.Content
                            side="top"
                            class="bg-popover text-popover-foreground max-w-64 rounded-md border px-3 py-2 text-xs shadow-md"
                          >
                            {run.lastError}
                          </Tooltip.Content>
                        </Tooltip.Portal>
                      </Tooltip.Root>
                    </Tooltip.Provider>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/each}
  </div>
{/if}
