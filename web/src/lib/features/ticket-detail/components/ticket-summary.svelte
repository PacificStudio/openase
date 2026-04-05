<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Calendar from '@lucide/svelte/icons/calendar'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import User from '@lucide/svelte/icons/user'
  import Workflow from '@lucide/svelte/icons/workflow'
  import { cn, formatCurrency, formatRelativeTime } from '$lib/utils'
  import TicketDependencies from './ticket-dependencies.svelte'
  import TicketExternalLinks from './ticket-external-links.svelte'
  import TicketFieldEditor from './ticket-field-editor.svelte'
  import TicketTokenUsageSummary from './ticket-token-usage-summary.svelte'
  import type {
    TicketDetail,
    TicketExternalLinkDraft,
    TicketReferenceOption,
    TicketRun,
  } from '../types'

  let {
    ticket,
    runs = [],
    runsLoaded = false,
    loadingRuns = false,
    runsError = '',
    availableTickets,
    savingFields = false,
    creatingDependency = false,
    deletingDependencyId = null,
    creatingExternalLink = false,
    deletingExternalLinkId = null,
    onLoadRuns,
    onSaveFields,
    onAddDependency,
    onDeleteDependency,
    onCreateExternalLink,
    onDeleteExternalLink,
  }: {
    ticket: TicketDetail
    runs?: TicketRun[]
    runsLoaded?: boolean
    loadingRuns?: boolean
    runsError?: string
    availableTickets: TicketReferenceOption[]
    savingFields?: boolean
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    creatingExternalLink?: boolean
    deletingExternalLinkId?: string | null
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onAddDependency?: (draft: {
      targetTicketId: string
      relation: string
    }) => Promise<boolean> | boolean
    onDeleteDependency?: (dependencyId: string) => void
    onCreateExternalLink?: (draft: TicketExternalLinkDraft) => Promise<boolean> | boolean
    onDeleteExternalLink?: (linkId: string) => void
    onLoadRuns?: () => Promise<void> | void
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

    <TicketTokenUsageSummary {ticket} {runs} {runsLoaded} {loadingRuns} {runsError} {onLoadRuns} />

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

  <Separator />
  <TicketExternalLinks
    links={ticket.externalLinks}
    creating={creatingExternalLink}
    deletingId={deletingExternalLinkId}
    onCreate={onCreateExternalLink}
    onDelete={onDeleteExternalLink}
  />
</div>
