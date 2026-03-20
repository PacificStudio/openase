<script lang="ts">
  import { cn, formatRelativeTime, truncate } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import {
    GitPullRequest,
    Bot,
    AlertTriangle,
    RotateCcw,
    ShieldAlert,
    Wallet,
  } from '@lucide/svelte'
  import type { BoardTicket } from '../types'

  let {
    ticket,
    class: className = '',
    onclick,
  }: {
    ticket: BoardTicket
    class?: string
    onclick?: (ticket: BoardTicket) => void
  } = $props()

  const priorityColors: Record<string, string> = {
    urgent: 'bg-red-500',
    high: 'bg-orange-500',
    medium: 'bg-blue-500',
    low: 'bg-zinc-400',
  }

  const workflowColors: Record<string, string> = {
    coding: 'bg-violet-500/15 text-violet-600 dark:text-violet-400',
    test: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
    security: 'bg-red-500/15 text-red-600 dark:text-red-400',
    review: 'bg-blue-500/15 text-blue-600 dark:text-blue-400',
    deploy: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
  }

  const anomalyConfig: Record<
    string,
    { label: string; variant: 'destructive' | 'secondary'; icon: typeof AlertTriangle }
  > = {
    hook_failed: { label: 'Hook failed', variant: 'destructive', icon: AlertTriangle },
    retry: { label: 'Retrying', variant: 'secondary', icon: RotateCcw },
    awaiting_approval: { label: 'Needs approval', variant: 'secondary', icon: ShieldAlert },
    budget_exhausted: { label: 'Budget exhausted', variant: 'destructive', icon: Wallet },
  }

  function handleClick() {
    onclick?.(ticket)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      handleClick()
    }
  }
</script>

<button
  type="button"
  class={cn(
    'border-border bg-card w-full cursor-pointer rounded-md border p-2.5 text-left',
    'hover:border-border/80 hover:bg-accent/50 transition-colors',
    'focus-visible:ring-ring focus-visible:ring-2 focus-visible:outline-none',
    className,
  )}
  onclick={handleClick}
  onkeydown={handleKeydown}
>
  <div class="flex items-start gap-2">
    <span class={cn('mt-1.5 size-2 shrink-0 rounded-full', priorityColors[ticket.priority])}></span>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5">
        <span class="text-muted-foreground text-xs font-medium">{ticket.identifier}</span>
      </div>
      <p class="text-foreground mt-0.5 text-sm leading-snug font-medium">
        {truncate(ticket.title, 60)}
      </p>
    </div>
  </div>

  <div class="mt-2 flex flex-wrap items-center gap-1.5">
    {#if ticket.workflowType}
      <span
        class={cn(
          'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium',
          workflowColors[ticket.workflowType] ?? 'bg-muted text-muted-foreground',
        )}
      >
        {ticket.workflowType}
      </span>
    {/if}

    {#if ticket.agentName}
      <span class="text-muted-foreground inline-flex items-center gap-0.5 text-[10px]">
        <Bot class="size-3" />
        {ticket.agentName}
      </span>
    {/if}

    {#if ticket.prCount && ticket.prCount > 0}
      <span class="text-muted-foreground inline-flex items-center gap-0.5 text-[10px]">
        <GitPullRequest class="size-3" />
        {ticket.prCount}
        {#if ticket.prStatus}
          <span class="text-muted-foreground/70">· {ticket.prStatus}</span>
        {/if}
      </span>
    {/if}
  </div>

  {#if ticket.anomaly}
    {@const config = anomalyConfig[ticket.anomaly]}
    {#if config}
      <div class="mt-1.5">
        <Badge variant={config.variant} class="h-4 gap-1 text-[10px]">
          <config.icon class="size-2.5" />
          {config.label}
        </Badge>
      </div>
    {/if}
  {/if}

  <div class="text-muted-foreground/70 mt-1.5 text-[10px]">
    {formatRelativeTime(ticket.updatedAt)}
  </div>
</button>
