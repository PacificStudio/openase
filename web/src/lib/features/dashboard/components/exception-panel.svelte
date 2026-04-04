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
    'hook.failed': XCircle,
    'ticket.budget_exhausted': DollarSign,
    'agent.failed': Pause,
    'ticket.retry_paused': AlertTriangle,
  }

  const colorMap: Record<ExceptionItem['type'], string> = {
    'hook.failed': 'text-red-500',
    'ticket.budget_exhausted': 'text-amber-500',
    'agent.failed': 'text-orange-500',
    'ticket.retry_paused': 'text-yellow-500',
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Exceptions</h3>
    {#if exceptions.length > 0}
      <span
        class="bg-destructive/10 text-destructive flex size-5 items-center justify-center rounded-full text-[10px] font-medium"
      >
        {exceptions.length}
      </span>
    {/if}
  </div>

  <div class="divide-border divide-y">
    {#each exceptions as item, idx (item.id)}
      {@const Icon = iconMap[item.type]}
      <div class="animate-stagger flex items-start gap-3 px-4 py-3" style="--stagger-index: {idx}">
        <Icon class={cn('mt-0.5 size-4 shrink-0', colorMap[item.type])} />
        <div class="min-w-0 flex-1">
          <p class="text-foreground text-sm leading-snug">{item.message}</p>
          <div class="mt-1 flex items-center gap-2">
            {#if item.ticketIdentifier}
              <span class="text-muted-foreground font-mono text-xs">
                {item.ticketIdentifier}
              </span>
            {/if}
            <span class="text-muted-foreground text-xs">
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
