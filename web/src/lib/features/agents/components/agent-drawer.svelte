<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { ApiError } from '$lib/api/client'
  import { deleteAgent, pauseAgent, resumeAgent } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Separator } from '$ui/separator'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Pause, Pencil, Play, Trash2 } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let {
    open = $bindable(false),
    agent,
    onOpenChange,
    onDeleted,
    onEditProvider,
  }: {
    open?: boolean
    agent: AgentInstance | null
    onOpenChange?: (open: boolean) => void
    onDeleted?: (agentId: string) => void
    onEditProvider?: (providerId: string) => void
  } = $props()

  let actionBusy = $state(false)

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

  function canPause(a: AgentInstance) {
    return (
      a.runtimeControlState === 'active' &&
      a.activeRunCount > 0 &&
      (a.status === 'claimed' || a.status === 'running')
    )
  }

  function canResume(a: AgentInstance) {
    return a.runtimeControlState === 'paused'
  }

  async function handlePause() {
    if (!agent) return
    actionBusy = true
    try {
      await pauseAgent(agent.id)
      toastStore.success(`Pause requested for "${agent.name}".`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to pause agent.')
    } finally {
      actionBusy = false
    }
  }

  async function handleResume() {
    if (!agent) return
    actionBusy = true
    try {
      await resumeAgent(agent.id)
      toastStore.success(`Resumed "${agent.name}".`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to resume agent.')
    } finally {
      actionBusy = false
    }
  }

  async function handleDelete() {
    if (!agent) return
    const confirmed = window.confirm(
      `Delete "${agent.name}"? This agent definition will be permanently removed.`,
    )
    if (!confirmed) return

    actionBusy = true
    try {
      await deleteAgent(agent.id)
      toastStore.success(`Deleted agent "${agent.name}".`)
      onDeleted?.(agent.id)
      onOpenChange?.(false)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete agent.')
    } finally {
      actionBusy = false
    }
  }
</script>

<Sheet
  bind:open
  onOpenChange={(value) => {
    open = value
    onOpenChange?.(value)
  }}
>
  <SheetContent class="overflow-y-auto sm:max-w-md">
    {#if !agent}
      <SheetHeader>
        <SheetTitle>Agent</SheetTitle>
        <SheetDescription>No agent selected.</SheetDescription>
      </SheetHeader>
    {:else}
      <SheetHeader>
        <SheetTitle class="flex items-center gap-2">
          <span class={cn('size-2.5 shrink-0 rounded-full', statusColors[agent.status])}></span>
          {agent.name}
        </SheetTitle>
        <SheetDescription>
          {agent.providerName} &middot; {agent.modelName}
        </SheetDescription>
      </SheetHeader>

      <div class="space-y-5 px-4">
        <div class="space-y-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Status</span>
            <div class="flex items-center gap-2">
              <span class="text-foreground">{statusLabels[agent.status]}</span>
              {#if agent.runtimeControlState !== 'active'}
                <Badge variant="outline" class="text-[10px]">
                  {agent.runtimeControlState === 'pause_requested' ? 'Pause Requested' : 'Paused'}
                </Badge>
              {/if}
            </div>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Active runs</span>
            <span class="text-foreground tabular-nums">{agent.activeRunCount}</span>
          </div>
          {#if agent.lastHeartbeat}
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Last heartbeat</span>
              <span class="text-foreground">{formatRelativeTime(agent.lastHeartbeat)}</span>
            </div>
          {/if}
          {#if agent.runtimeStartedAt}
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Runtime started</span>
              <span class="text-foreground">{formatRelativeTime(agent.runtimeStartedAt)}</span>
            </div>
          {/if}
          {#if agent.sessionId}
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">Session</span>
              <span class="text-foreground font-mono text-xs">{agent.sessionId}</span>
            </div>
          {/if}
        </div>

        <Separator />

        {#if agent.currentStepSummary}
          <div class="space-y-2">
            <div class="text-foreground text-sm font-medium">Current step</div>
            {#if agent.currentStepStatus}
              <Badge variant="secondary" class="text-[10px]">{agent.currentStepStatus}</Badge>
            {/if}
            <p class="text-muted-foreground text-sm">{agent.currentStepSummary}</p>
            {#if agent.currentStepChangedAt}
              <p class="text-muted-foreground text-xs">
                Updated {formatRelativeTime(agent.currentStepChangedAt)}
              </p>
            {/if}
          </div>
          <Separator />
        {/if}

        <div class="space-y-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Completed today</span>
            <span class="text-foreground tabular-nums">{agent.todayCompleted}</span>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">Cost today</span>
            <span class="text-foreground tabular-nums">{formatCurrency(agent.todayCost)}</span>
          </div>
        </div>

        {#if agent.lastError}
          <div class="border-destructive/40 bg-destructive/10 rounded-md border px-3 py-2">
            <p class="text-destructive text-xs">{agent.lastError}</p>
          </div>
        {/if}

        <Separator />

        <div class="flex flex-wrap gap-2">
          {#if onEditProvider}
            <Button
              variant="outline"
              size="sm"
              onclick={() => onEditProvider?.(agent.providerId)}
            >
              <Pencil class="size-3.5" />
              Edit Provider
            </Button>
          {/if}
          {#if canResume(agent)}
            <Button variant="outline" size="sm" disabled={actionBusy} onclick={handleResume}>
              <Play class="size-3.5" />
              Resume
            </Button>
          {:else if canPause(agent)}
            <Button variant="outline" size="sm" disabled={actionBusy} onclick={handlePause}>
              <Pause class="size-3.5" />
              Pause
            </Button>
          {/if}
          <Button
            variant="outline"
            size="sm"
            class="text-destructive hover:text-destructive"
            disabled={actionBusy}
            onclick={handleDelete}
          >
            <Trash2 class="size-3.5" />
            Delete
          </Button>
        </div>
      </div>
    {/if}
  </SheetContent>
</Sheet>
