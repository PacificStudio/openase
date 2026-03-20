<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Workflow from '@lucide/svelte/icons/workflow'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import User from '@lucide/svelte/icons/user'
  import Calendar from '@lucide/svelte/icons/calendar'
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import type { TicketDetail } from '../types'

  let { ticket }: { ticket: TicketDetail } = $props()

  const costPercent = $derived.by(() =>
    ticket.budgetUsd > 0 ? Math.round((ticket.costAmount / ticket.budgetUsd) * 100) : 0,
  )
  const costOverBudget = $derived(costPercent > 80)
</script>

<div class="flex flex-col gap-3 px-5 py-3">
  {#if ticket.description}
    <div class="text-xs leading-relaxed text-muted-foreground">
      {ticket.description}
    </div>
    <Separator />
  {/if}

  <div class="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-2.5 text-xs">
    {#if ticket.workflow}
      <div class="flex items-center gap-1.5 text-muted-foreground">
        <Workflow class="size-3.5" />
        <span>Workflow</span>
      </div>
      <div class="text-foreground">{ticket.workflow.name}</div>
    {/if}

    <div class="flex items-center gap-1.5 text-muted-foreground">
      <Bot class="size-3.5" />
      <span>Agent</span>
    </div>
    <div class="flex items-center gap-1.5">
      {#if ticket.assignedAgent}
        <span
          class="inline-block size-1.5 rounded-full bg-green-400"
          title="Online"
        ></span>
        <span class="text-foreground">{ticket.assignedAgent.name}</span>
        <Badge variant="outline" class="text-[10px] py-0 h-4">
          {ticket.assignedAgent.provider}
        </Badge>
      {:else}
        <span class="text-muted-foreground italic">Unassigned</span>
      {/if}
    </div>

    <div class="flex items-center gap-1.5 text-muted-foreground">
      <DollarSign class="size-3.5" />
      <span>Cost</span>
    </div>
    <div class="flex items-center gap-2">
      <span class={cn('text-foreground', costOverBudget && 'text-red-400')}>
        {formatCurrency(ticket.costAmount)}
      </span>
      <span class="text-muted-foreground">/ {formatCurrency(ticket.budgetUsd)}</span>
      <div class="h-1 flex-1 max-w-16 rounded-full bg-muted overflow-hidden">
        <div
          class={cn(
            'h-full rounded-full transition-all',
            costOverBudget ? 'bg-red-400' : 'bg-green-400',
          )}
          style="width: {Math.min(costPercent, 100)}%"
        ></div>
      </div>
    </div>

    <div class="flex items-center gap-1.5 text-muted-foreground">
      <RotateCcw class="size-3.5" />
      <span>Attempts</span>
    </div>
    <div class="text-foreground">{ticket.attemptCount}</div>

    <div class="flex items-center gap-1.5 text-muted-foreground">
      <User class="size-3.5" />
      <span>Created by</span>
    </div>
    <div class="text-foreground">{ticket.createdBy}</div>

    <div class="flex items-center gap-1.5 text-muted-foreground">
      <Calendar class="size-3.5" />
      <span>Created</span>
    </div>
    <div class="text-foreground">{formatRelativeTime(ticket.createdAt)}</div>
  </div>

  {#if ticket.dependencies.length > 0}
    <Separator />
    <div class="flex flex-col gap-1.5">
      <span class="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        Dependencies
      </span>
      {#each ticket.dependencies as dep}
        <div class="flex items-center gap-2 text-xs">
          <span class="font-mono text-muted-foreground">{dep.identifier}</span>
          <span class="truncate text-foreground">{dep.title}</span>
          <Badge variant="outline" class="ml-auto text-[10px] py-0 h-4 shrink-0">
            {dep.relation}
          </Badge>
        </div>
      {/each}
    </div>
  {/if}
</div>
