<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Hand, Terminal, Pause, Play } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import type { AgentInstance } from '../types'
  import type { TranslationKey } from '$lib/i18n'

  let {
    agents,
    onSelectTicket,
    runtimeActionAgentId = null,
    onViewOutput,
    onInterruptAgent,
    onPauseAgent,
    onResumeAgent,
  }: {
    agents: AgentInstance[]
    onSelectTicket?: (ticketId: string) => void
    runtimeActionAgentId?: string | null
    onViewOutput?: (agentId: string) => void
    onInterruptAgent?: (agentId: string) => void
    onPauseAgent?: (agentId: string) => void
    onResumeAgent?: (agentId: string) => void
  } = $props()

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    claimed: 'bg-amber-500',
    running: 'bg-blue-500',
    paused: 'bg-orange-500',
    failed: 'bg-red-500',
    interrupted: 'bg-rose-500',
    terminated: 'bg-slate-500',
  }

  const columnLabelKeys: Record<
    'status' | 'agent' | 'runtimeSummary' | 'lastHeartbeat' | 'completed' | 'cost' | 'actions',
    TranslationKey
  > = {
    status: 'agents.agentList.column.status',
    agent: 'agents.agentList.column.agent',
    runtimeSummary: 'agents.agentList.column.runtimeSummary',
    lastHeartbeat: 'agents.agentList.column.lastHeartbeat',
    completed: 'agents.agentList.column.completed',
    cost: 'agents.agentList.column.cost',
    actions: 'agents.agentList.column.actions',
  } as const

  const runtimeControlLabelKeys: Record<AgentInstance['runtimeControlState'], TranslationKey> = {
    active: 'agents.agentList.runtimeControl.active',
    interrupt_requested: 'agents.agentList.runtimeControl.interruptRequestedStatus',
    pause_requested: 'agents.agentList.runtimeControl.pauseRequestedStatus',
    paused: 'agents.agentList.runtimeControl.pausedStatus',
    retired: 'agents.agentList.runtimeControl.retiredStatus',
  }

  const runtimeControlMessageKeys = {
    updating: 'agents.agentList.runtimeControl.updating',
    interruptRequested: 'agents.agentList.runtimeControl.interruptRequestedMessage',
    pauseRequested: 'agents.agentList.runtimeControl.pauseRequestedMessage',
    pausedInterrupt: 'agents.agentList.runtimeControl.pausedInterrupt',
    noActiveRunsInterrupt: 'agents.agentList.runtimeControl.noActiveRunsToInterrupt',
    noActiveRunsPause: 'agents.agentList.runtimeControl.noActiveRunsToPause',
    onlyClaimedRunningInterrupt: 'agents.agentList.runtimeControl.onlyClaimedOrRunningCanInterrupt',
    onlyClaimedRunningPause: 'agents.agentList.runtimeControl.onlyClaimedOrRunningCanPause',
    interruptAction: 'agents.agentList.runtimeControl.interruptThisAgent',
    pauseAction: 'agents.agentList.runtimeControl.pauseThisAgent',
    alreadyPaused: 'agents.agentList.runtimeControl.alreadyPaused',
    waitForPauseBeforeResume: 'agents.agentList.runtimeControl.waitForPauseBeforeResume',
    pauseBeforeResuming: 'agents.agentList.runtimeControl.pauseBeforeResuming',
    resumeAction: 'agents.agentList.runtimeControl.resumeThisAgent',
  } as const satisfies Record<
    | 'updating'
    | 'interruptRequested'
    | 'pauseRequested'
    | 'pausedInterrupt'
    | 'noActiveRunsInterrupt'
    | 'noActiveRunsPause'
    | 'onlyClaimedRunningInterrupt'
    | 'onlyClaimedRunningPause'
    | 'interruptAction'
    | 'pauseAction'
    | 'alreadyPaused'
    | 'waitForPauseBeforeResume'
    | 'pauseBeforeResuming'
    | 'resumeAction',
    TranslationKey
  >

  const statusLabelKeys: Record<AgentInstance['status'], TranslationKey> = {
    idle: 'agents.status.idle',
    claimed: 'agents.status.claimed',
    running: 'agents.status.running',
    paused: 'agents.status.paused',
    failed: 'agents.status.failed',
    interrupted: 'agents.status.interrupted',
    terminated: 'agents.status.terminated',
  }

  const runtimeControlClasses: Record<AgentInstance['runtimeControlState'], string> = {
    active: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
    interrupt_requested: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300',
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

  function canInterrupt(agent: AgentInstance) {
    return (
      agent.runtimeControlState === 'active' &&
      agent.activeRunCount > 0 &&
      (agent.status === 'claimed' || agent.status === 'running')
    )
  }

  function canResume(agent: AgentInstance) {
    return agent.runtimeControlState === 'paused'
  }

  function interruptTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) {
      return i18nStore.t(runtimeControlMessageKeys.updating)
    }
    if (agent.runtimeControlState === 'interrupt_requested') {
      return i18nStore.t(runtimeControlMessageKeys.interruptRequested)
    }
    if (agent.runtimeControlState === 'pause_requested') {
      return i18nStore.t(runtimeControlMessageKeys.pauseRequested)
    }
    if (agent.runtimeControlState === 'paused') {
      return i18nStore.t(runtimeControlMessageKeys.pausedInterrupt)
    }
    if (agent.activeRunCount === 0) {
      return i18nStore.t(runtimeControlMessageKeys.noActiveRunsInterrupt)
    }
    if (agent.status !== 'claimed' && agent.status !== 'running') {
      return i18nStore.t(runtimeControlMessageKeys.onlyClaimedRunningInterrupt)
    }
    return i18nStore.t(runtimeControlMessageKeys.interruptAction)
  }

  function pauseTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) {
      return i18nStore.t(runtimeControlMessageKeys.updating)
    }
    if (agent.runtimeControlState === 'pause_requested') {
      return i18nStore.t(runtimeControlMessageKeys.pauseRequested)
    }
    if (agent.runtimeControlState === 'paused') {
      return i18nStore.t(runtimeControlMessageKeys.alreadyPaused)
    }
    if (agent.activeRunCount === 0) {
      return i18nStore.t(runtimeControlMessageKeys.noActiveRunsPause)
    }
    if (agent.status !== 'claimed' && agent.status !== 'running') {
      return i18nStore.t(runtimeControlMessageKeys.onlyClaimedRunningPause)
    }
    return i18nStore.t(runtimeControlMessageKeys.pauseAction)
  }

  function resumeTitle(agent: AgentInstance) {
    if (runtimeActionAgentId === agent.id) {
      return i18nStore.t(runtimeControlMessageKeys.updating)
    }
    if (agent.runtimeControlState === 'pause_requested') {
      return i18nStore.t(runtimeControlMessageKeys.waitForPauseBeforeResume)
    }
    if (agent.runtimeControlState !== 'paused') {
      return i18nStore.t(runtimeControlMessageKeys.pauseBeforeResuming)
    }
    return i18nStore.t(runtimeControlMessageKeys.resumeAction)
  }
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-border text-muted-foreground border-b text-left text-xs">
        <th class="pr-2 pb-2 pl-3 font-medium">{i18nStore.t(columnLabelKeys.status)}</th>
        <th class="px-2 pb-2 font-medium">{i18nStore.t(columnLabelKeys.agent)}</th>
        <th class="px-2 pb-2 font-medium">{i18nStore.t(columnLabelKeys.runtimeSummary)}</th>
        <th class="px-2 pb-2 font-medium">{i18nStore.t(columnLabelKeys.lastHeartbeat)}</th>
        <th class="px-2 pb-2 text-right font-medium">{i18nStore.t(columnLabelKeys.completed)}</th>
        <th class="px-2 pb-2 text-right font-medium">{i18nStore.t(columnLabelKeys.cost)}</th>
        <th class="pr-3 pb-2 pl-2 text-right font-medium">{i18nStore.t(columnLabelKeys.actions)}</th
        >
      </tr>
    </thead>
    <tbody>
      {#each agents as agent (agent.id)}
        <tr class="group border-border/50 hover:bg-muted/30 border-b transition-colors">
          <td class="py-2.5 pr-2 pl-3">
            <div class="flex items-center gap-2">
              <span class={cn('size-2 rounded-full', statusColors[agent.status])}></span>
              <span class="text-muted-foreground text-xs">
                {i18nStore.t(statusLabelKeys[agent.status])}
              </span>
              {#if agent.runtimeControlState !== 'active'}
                <span
                  class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${runtimeControlClasses[agent.runtimeControlState]}`}
                >
                  {i18nStore.t(runtimeControlLabelKeys[agent.runtimeControlState])}
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
              {i18nStore.t('agents.agentList.activeRuns', { count: agent.activeRunCount })}
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
              <div class="text-muted-foreground text-xs">
                {i18nStore.t('agents.agentList.concurrentRunsHint')}
              </div>
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
                aria-label={i18nStore.t('agents.agentList.action.viewOutput')}
                title={i18nStore.t('agents.agentList.action.viewAgentOutput')}
                onclick={() => onViewOutput?.(agent.id)}
              >
                <Terminal class="size-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label={i18nStore.t('agents.agentList.action.interruptAgent')}
                disabled={!canInterrupt(agent) || runtimeActionAgentId === agent.id}
                title={interruptTitle(agent)}
                onclick={() => onInterruptAgent?.(agent.id)}
              >
                <Hand class="size-3.5" />
              </Button>
              {#if agent.runtimeControlState === 'paused'}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={i18nStore.t('agents.agentList.action.resumeAgent')}
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
                  aria-label={i18nStore.t('agents.agentList.action.pauseAgent')}
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
