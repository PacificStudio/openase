<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { ExceptionItem } from '../types'
  import { AlertTriangle, DollarSign, Pause, XCircle } from '@lucide/svelte'
  import type { Component } from 'svelte'

  let {
    exceptions,
    class: className = '',
  }: {
    exceptions: ExceptionItem[]
    class?: string
  } = $props()

  const iconMap: Record<ExceptionItem['type'], Component> = {
    hook_failed: XCircle,
    budget_alert: DollarSign,
    agent_stalled: Pause,
    retry_paused: AlertTriangle,
  }

  const colorMap: Record<ExceptionItem['type'], string> = {
    hook_failed: 'text-red-500',
    budget_alert: 'text-amber-500',
    agent_stalled: 'text-orange-500',
    retry_paused: 'text-yellow-500',
  }
</script>

<div class={cn('rounded-md border border-border bg-card', className)}>
  <div class="flex items-center justify-between border-b border-border px-4 py-3">
    <h3 class="text-sm font-medium text-foreground">Exceptions</h3>
    {#if exceptions.length > 0}
      <span class="flex size-5 items-center justify-center rounded-full bg-destructive/10 text-[10px] font-medium text-destructive">
        {exceptions.length}
      </span>
    {/if}
  </div>

  <div class="divide-y divide-border">
    {#each exceptions as item (item.id)}
      {@const Icon = iconMap[item.type]}
      <div class="flex items-start gap-3 px-4 py-3">
        <Icon class={cn('size-4 mt-0.5 shrink-0', colorMap[item.type])} />
        <div class="flex-1 min-w-0">
          <p class="text-sm text-foreground leading-snug">{item.message}</p>
          <div class="mt-1 flex items-center gap-2">
            {#if item.ticketIdentifier}
              <span class="text-xs font-mono text-muted-foreground">
                {item.ticketIdentifier}
              </span>
            {/if}
            <span class="text-xs text-muted-foreground">
              {formatRelativeTime(item.timestamp)}
            </span>
          </div>
        </div>
      </div>
    {:else}
      <div class="px-4 py-8 text-center text-xs text-muted-foreground">
        No exceptions. All systems nominal.
      </div>
    {/each}
  </div>
</div>
