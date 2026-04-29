<script lang="ts">
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
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
    if (next.has(agentId)) next.delete(agentId)
    else next.add(agentId)
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

  const statusLabels: Record<AgentInstance['status'], TranslationKey> = {
    idle: 'agents.status.idle',
    claimed: 'agents.status.claimed',
    running: 'agents.status.running',
    paused: 'agents.status.paused',
    failed: 'agents.status.failed',
    interrupted: 'agents.status.interrupted',
    terminated: 'agents.status.terminated',
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

  const runStatusLabels: Record<AgentRunInstance['status'], TranslationKey> = {
    launching: 'agents.runtime.launching',
    ready: 'agents.runtime.ready',
    executing: 'agents.runtime.executing',
    completed: 'agents.runtime.completed',
    errored: 'agents.runtime.errored',
    interrupted: 'agents.runtime.interrupted',
    terminated: 'agents.runtime.terminated',
  }

  function canControlActiveRun(agent: AgentInstance) {
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
    return (runsByAgentId.get(agentId) ?? []).filter(
      (r) => r.status === 'launching' || r.status === 'ready' || r.status === 'executing',
    )
  }

  const truncate = (text: string, max: number) =>
    text.length > max ? text.slice(0, max) + '…' : text
</script>

{#if agents.length === 0}
  <div
    class="border-border bg-card animate-fade-in-up rounded-xl border border-dashed px-4 py-14 text-center"
  >
    <div class="bg-muted/60 mx-auto mb-4 flex size-12 items-center justify-center rounded-full">
      <Bot class="text-muted-foreground size-5" />
    </div>
    <p class="text-foreground text-sm font-medium">{i18nStore.t('agents.noAgentsRegistered')}</p>
    <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-sm">
      {i18nStore.t('agents.noAgentsDescription')}
    </p>
  </div>
{:else}
  <div class="space-y-2" data-tour="agent-cards-list">
    {#each agents as agent (agent.id)}
      {@const Icon = adapterIcons[agent.adapterType] ?? Cpu}
      {@const runs = activeRuns(agent.id)}
      {@const expanded = expandedIds.has(agent.id)}
      {@const hasRuns = runs.length > 0}

      <div class="border-border/60 bg-card/60 rounded-xl border">
        <div
          class="flex flex-wrap items-center gap-x-2 gap-y-1 px-3 py-2.5 sm:flex-nowrap sm:gap-3 sm:px-4"
        >
          <button
            type="button"
            class="flex shrink-0 items-center"
            onclick={() => hasRuns && toggleExpand(agent.id)}
            disabled={!hasRuns}
            aria-label={expanded ? i18nStore.t('agents.collapse') : i18nStore.t('agents.expand')}
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
            {i18nStore.t(statusLabels[agent.status])} · {i18nStore.t(
              agent.activeRunCount === 1 ? 'agents.taskCountOne' : 'agents.taskCountOther',
              { count: agent.activeRunCount },
            )}
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
                ? i18nStore.t('agents.interruptRequested')
                : agent.runtimeControlState === 'pause_requested'
                  ? i18nStore.t('agents.pauseRequested')
                  : agent.runtimeControlState === 'retired'
                    ? i18nStore.t('agents.retired')
                    : i18nStore.t('agents.status.paused')}
            </span>
          {/if}

          <span class="hidden flex-1 sm:block"></span>

          {#if agent.lastHeartbeat}
            <span class="text-muted-foreground hidden text-[11px] whitespace-nowrap sm:inline">
              {i18nStore.t('agents.heartbeat', {
                time: formatRelativeTime(agent.lastHeartbeat),
              })}
            </span>
          {/if}

          <div class="ml-auto flex shrink-0 items-center gap-0.5 sm:ml-0">
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={i18nStore.t('agents.editAgent')}
              title={i18nStore.t('agents.editAgent')}
              onclick={() => onSelectAgent?.(agent.id)}
            >
              <Pencil class="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={i18nStore.t('agents.interruptAgent')}
              disabled={!canControlActiveRun(agent) || runtimeActionAgentId === agent.id}
              title={i18nStore.t('agents.interruptRun')}
              onclick={() => onInterruptAgent?.(agent.id)}
            >
              <Hand class="size-3.5" />
            </Button>
            {#if agent.runtimeControlState === 'paused'}
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={i18nStore.t('agents.resumeAgent')}
                disabled={!canResume(agent) || runtimeActionAgentId === agent.id}
                title={i18nStore.t('agents.resumeThisAgent')}
                onclick={() => onResumeAgent?.(agent.id)}
              >
                <Play class="size-3.5" />
              </Button>
            {:else}
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={i18nStore.t('agents.pauseAgent')}
                disabled={!canControlActiveRun(agent) || runtimeActionAgentId === agent.id}
                title={i18nStore.t('agents.pauseThisAgent')}
                onclick={() => onPauseAgent?.(agent.id)}
              >
                <Pause class="size-3.5" />
              </Button>
            {/if}
          </div>
        </div>

        {#if expanded && hasRuns}
          <div class="border-border border-t px-3 py-2 sm:px-4">
            <div class="space-y-1.5">
              {#each runs as run (run.id)}
                <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
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
                    {i18nStore.t(runStatusLabels[run.status])}
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
