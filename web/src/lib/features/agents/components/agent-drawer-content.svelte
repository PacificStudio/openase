<script lang="ts">
  import { formatCurrency, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Archive, Hand, Pause, Play, Trash2 } from '@lucide/svelte'

  import type { AgentInstance } from '../types'

  let {
    agent,
    actionBusy = false,
    canInterrupt = false,
    canPause = false,
    canResume = false,
    canRetire = false,
    onInterrupt,
    onPause,
    onResume,
    onRetire,
    onDelete,
  }: {
    agent: AgentInstance
    actionBusy?: boolean
    canInterrupt?: boolean
    canPause?: boolean
    canResume?: boolean
    canRetire?: boolean
    onInterrupt?: () => void
    onPause?: () => void
    onResume?: () => void
    onRetire?: () => void
    onDelete?: () => void
  } = $props()
</script>

<div class="flex-1 space-y-5 px-6 py-5">
  <section class="bg-muted/40 rounded-lg px-4 py-3">
    <div class="grid grid-cols-3 gap-4 text-center">
      <div>
        <div class="text-foreground text-lg font-semibold tabular-nums">{agent.activeRunCount}</div>
        <div class="text-muted-foreground text-[11px]">Active</div>
      </div>
      <div>
        <div class="text-foreground text-lg font-semibold tabular-nums">{agent.todayCompleted}</div>
        <div class="text-muted-foreground text-[11px]">Today</div>
      </div>
      <div>
        <div class="text-foreground text-lg font-semibold tabular-nums">
          {formatCurrency(agent.todayCost)}
        </div>
        <div class="text-muted-foreground text-[11px]">Cost</div>
      </div>
    </div>
  </section>

  {#if agent.currentStepSummary}
    <section class="space-y-2">
      <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
        Current step
      </h3>
      <div class="bg-muted/30 rounded-lg border px-3 py-2.5">
        {#if agent.currentStepStatus}
          <Badge variant="secondary" class="mb-1.5 text-[10px]">{agent.currentStepStatus}</Badge>
        {/if}
        <p
          class="text-foreground truncate text-sm leading-relaxed transition-all hover:break-words hover:whitespace-normal"
          title={agent.currentStepSummary}
        >
          {agent.currentStepSummary}
        </p>
        {#if agent.currentStepChangedAt}
          <p class="text-muted-foreground mt-1.5 text-[11px]">
            {formatRelativeTime(agent.currentStepChangedAt)}
          </p>
        {/if}
      </div>
    </section>
  {/if}

  <section class="space-y-2">
    <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">Details</h3>
    <div class="space-y-2.5 text-sm">
      <div class="flex items-center justify-between">
        <span class="text-muted-foreground">Permission</span>
        <span
          class="text-foreground text-xs font-medium capitalize {agent.permissionProfile ===
          'unrestricted'
            ? 'text-amber-600 dark:text-amber-400'
            : ''}"
        >
          {agent.permissionProfile}
        </span>
      </div>
      {#if agent.lastHeartbeat}
        <div class="flex items-center justify-between">
          <span class="text-muted-foreground">Heartbeat</span>
          <span class="text-foreground">{formatRelativeTime(agent.lastHeartbeat)}</span>
        </div>
      {/if}
      {#if agent.runtimeStartedAt}
        <div class="flex items-center justify-between">
          <span class="text-muted-foreground">Runtime started</span>
          <span class="text-foreground">{formatRelativeTime(agent.runtimeStartedAt)}</span>
        </div>
      {/if}
      {#if agent.sessionId}
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted-foreground shrink-0">Session</span>
          <span class="text-foreground truncate font-mono text-xs">{agent.sessionId}</span>
        </div>
      {/if}
    </div>
  </section>

  {#if agent.lastError}
    <section class="space-y-2">
      <h3 class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">Error</h3>
      <div class="border-destructive/30 bg-destructive/5 rounded-lg border px-3 py-2.5">
        <p class="text-destructive text-xs leading-relaxed">{agent.lastError}</p>
      </div>
    </section>
  {/if}

  <div class="border-border flex items-center gap-2 border-t pt-4">
    {#if canInterrupt}
      <Button variant="outline" size="sm" disabled={actionBusy} onclick={onInterrupt}>
        <Hand class="size-3.5" />
        Interrupt
      </Button>
    {/if}
    {#if canResume}
      <Button variant="outline" size="sm" disabled={actionBusy} onclick={onResume}>
        <Play class="size-3.5" />
        Resume
      </Button>
    {:else if canPause}
      <Button variant="outline" size="sm" disabled={actionBusy} onclick={onPause}>
        <Pause class="size-3.5" />
        Pause
      </Button>
    {/if}
    {#if canRetire}
      <Button variant="outline" size="sm" disabled={actionBusy} onclick={onRetire}>
        <Archive class="size-3.5" />
        Retire
      </Button>
    {/if}
    <div class="flex-1"></div>
    <Button
      variant="ghost"
      size="sm"
      class="text-destructive hover:text-destructive"
      disabled={actionBusy}
      onclick={onDelete}
    >
      <Trash2 class="size-3.5" />
      Delete
    </Button>
  </div>
</div>
