<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Workflow from '@lucide/svelte/icons/workflow'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
import User from '@lucide/svelte/icons/user'
import Calendar from '@lucide/svelte/icons/calendar'
import Link2 from '@lucide/svelte/icons/link-2'
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
    <div class="text-muted-foreground text-xs leading-relaxed">
      {ticket.description}
    </div>
    <Separator />
  {/if}

  <div class="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-2.5 text-xs">
    {#if ticket.workflow}
      <div class="text-muted-foreground flex items-center gap-1.5">
        <Workflow class="size-3.5" />
        <span>Workflow</span>
      </div>
      <div class="text-foreground">{ticket.workflow.name}</div>
    {/if}

    <div class="text-muted-foreground flex items-center gap-1.5">
      <Bot class="size-3.5" />
      <span>Agent</span>
    </div>
    <div class="flex items-center gap-1.5">
      {#if ticket.assignedAgent}
        <span class="inline-block size-1.5 rounded-full bg-green-400" title="Online"></span>
        <span class="text-foreground">{ticket.assignedAgent.name}</span>
        <Badge variant="outline" class="h-4 py-0 text-[10px]">
          {ticket.assignedAgent.provider}
        </Badge>
      {:else}
        <span class="text-muted-foreground italic">Unassigned</span>
      {/if}
    </div>

    <div class="text-muted-foreground flex items-center gap-1.5">
      <DollarSign class="size-3.5" />
      <span>Cost</span>
    </div>
    <div class="flex items-center gap-2">
      <span class={cn('text-foreground', costOverBudget && 'text-red-400')}>
        {formatCurrency(ticket.costAmount)}
      </span>
      <span class="text-muted-foreground">/ {formatCurrency(ticket.budgetUsd)}</span>
      <div class="bg-muted h-1 max-w-16 flex-1 overflow-hidden rounded-full">
        <div
          class={cn(
            'h-full rounded-full transition-all',
            costOverBudget ? 'bg-red-400' : 'bg-green-400',
          )}
          style="width: {Math.min(costPercent, 100)}%"
        ></div>
      </div>
    </div>

    <div class="text-muted-foreground flex items-center gap-1.5">
      <RotateCcw class="size-3.5" />
      <span>Attempts</span>
    </div>
    <div class="text-foreground">{ticket.attemptCount}</div>

    <div class="text-muted-foreground flex items-center gap-1.5">
      <User class="size-3.5" />
      <span>Created by</span>
    </div>
    <div class="text-foreground">{ticket.createdBy}</div>

    <div class="text-muted-foreground flex items-center gap-1.5">
      <Calendar class="size-3.5" />
      <span>Created</span>
    </div>
    <div class="text-foreground">{formatRelativeTime(ticket.createdAt)}</div>
  </div>

  {#if ticket.dependencies.length > 0}
    <Separator />
    <div class="flex flex-col gap-1.5">
      <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
        Dependencies
      </span>
      {#each ticket.dependencies as dep}
        <div class="flex items-center gap-2 text-xs">
          <span class="text-muted-foreground font-mono">{dep.identifier}</span>
          <span class="text-foreground truncate">{dep.title}</span>
          <Badge variant="outline" class="ml-auto h-4 shrink-0 py-0 text-[10px]">
            {dep.relation}
          </Badge>
        </div>
      {/each}
    </div>
  {/if}

  {#if ticket.externalLinks.length > 0}
    <Separator />
    <div class="flex flex-col gap-2">
      <span class="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        External Links
      </span>
      {#each ticket.externalLinks as link}
        <a
          class="flex items-start gap-2 rounded-md border border-border/60 bg-muted/30 px-2.5 py-2 text-xs transition-colors hover:bg-muted/60"
          href={link.url}
          target="_blank"
          rel="noreferrer"
        >
          <Link2 class="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate text-foreground">{link.title || link.externalId}</span>
              <Badge variant="outline" class="h-4 py-0 text-[10px] shrink-0">
                {link.type}
              </Badge>
            </div>
            <div class="mt-1 flex items-center gap-2 text-[10px] text-muted-foreground">
              <span class="font-mono">{link.externalId}</span>
              <span>{link.relation}</span>
              {#if link.status}
                <span>{link.status}</span>
              {/if}
            </div>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</div>
