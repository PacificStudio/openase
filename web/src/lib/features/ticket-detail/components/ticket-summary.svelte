<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Calendar from '@lucide/svelte/icons/calendar'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
  import Link2 from '@lucide/svelte/icons/link-2'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import User from '@lucide/svelte/icons/user'
  import Workflow from '@lucide/svelte/icons/workflow'
  import { cn, formatCurrency, formatRelativeTime } from '$lib/utils'
  import TicketDependencies from './ticket-dependencies.svelte'
  import TicketFieldEditor from './ticket-field-editor.svelte'
  import type { TicketDetail, TicketReferenceOption } from '../types'

  let {
    ticket,
    availableTickets,
    savingFields = false,
    creatingDependency = false,
    deletingDependencyId = null,
    onSaveFields,
    onAddDependency,
    onDeleteDependency,
  }: {
    ticket: TicketDetail
    availableTickets: TicketReferenceOption[]
    savingFields?: boolean
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onAddDependency?: (draft: { targetTicketId: string; relation: string }) => void
    onDeleteDependency?: (dependencyId: string) => void
  } = $props()

  const costPercent = $derived.by(() =>
    ticket.budgetUsd > 0 ? Math.round((ticket.costAmount / ticket.budgetUsd) * 100) : 0,
  )
  const costOverBudget = $derived(costPercent > 80)
</script>

<div class="flex flex-col gap-4 px-6 py-5">
  <TicketFieldEditor {ticket} saving={savingFields} onSave={onSaveFields} />

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

  <Separator />

  <TicketDependencies
    {ticket}
    {availableTickets}
    {creatingDependency}
    {deletingDependencyId}
    {onAddDependency}
    {onDeleteDependency}
  />

  {#if ticket.externalLinks.length > 0}
    <Separator />
    <div class="flex flex-col gap-2">
      <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
        External Links
      </span>
      {#each ticket.externalLinks as link (link.id)}
        <a
          class="border-border/60 bg-muted/30 hover:bg-muted/60 flex items-start gap-2 rounded-md border px-2.5 py-2 text-xs transition-colors"
          href={link.url}
          target="_blank"
          rel="noreferrer"
        >
          <Link2 class="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-foreground truncate">{link.title || link.externalId}</span>
              <Badge variant="outline" class="h-4 shrink-0 py-0 text-[10px]">
                {link.type}
              </Badge>
            </div>
            <div class="text-muted-foreground mt-1 flex items-center gap-2 text-[10px]">
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
