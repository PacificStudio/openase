<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Calendar from '@lucide/svelte/icons/calendar'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import User from '@lucide/svelte/icons/user'
  import Workflow from '@lucide/svelte/icons/workflow'
  import { cn, formatCount, formatCurrency, formatRelativeTime } from '$lib/utils'
  import TicketDependencies from './ticket-dependencies.svelte'
  import TicketExternalLinks from './ticket-external-links.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketRuntimeStateCard from './ticket-runtime-state-card.svelte'
  import type { DependencyDraft } from '../mutation-shared'
  import type {
    HookExecution,
    TicketDetail,
    TicketReferenceOption,
    TicketRepoOption,
  } from '../types'

  let {
    ticket,
    hooks,
    dependencyCandidates,
    repoOptions,
    creatingDependency = false,
    deletingDependencyId = null,
    creatingExternalLink = false,
    deletingExternalLinkId = null,
    creatingRepoScope = false,
    updatingRepoScopeId = null,
    deletingRepoScopeId = null,
    resumingRetry = false,
    resettingWorkspace = false,
    onResumeRetry,
    onResetWorkspace,
    onAddDependency,
    onDeleteDependency,
    onCreateExternalLink,
    onDeleteExternalLink,
    onCreateScope,
    onUpdateScope,
    onDeleteScope,
  }: {
    ticket: TicketDetail
    hooks: HookExecution[]
    dependencyCandidates: TicketReferenceOption[]
    repoOptions: TicketRepoOption[]
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    creatingExternalLink?: boolean
    deletingExternalLinkId?: string | null
    creatingRepoScope?: boolean
    updatingRepoScopeId?: string | null
    deletingRepoScopeId?: string | null
    resumingRetry?: boolean
    resettingWorkspace?: boolean
    onResumeRetry?: () => Promise<void> | void
    onResetWorkspace?: () => Promise<void> | void
    onAddDependency?: (draft: DependencyDraft) => Promise<boolean> | boolean
    onDeleteDependency?: (dependencyId: string) => void
    onCreateExternalLink?: (draft: {
      type: string
      url: string
      externalId: string
      title: string
      status: string
      relation: string
    }) => Promise<boolean> | boolean
    onDeleteExternalLink?: (linkId: string) => void
    onCreateScope?: (draft: {
      repoId: string
      branchName: string
      pullRequestUrl: string
    }) => Promise<boolean> | boolean
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
      },
    ) => void
    onDeleteScope?: (scopeId: string) => void
  } = $props()

  const costPercent = $derived.by(() =>
    ticket.budgetUsd > 0 ? Math.round((ticket.costAmount / ticket.budgetUsd) * 100) : 0,
  )
  const costOverBudget = $derived(costPercent > 80)
</script>

<div
  class="border-border w-full shrink-0 border-t md:sticky md:top-0 md:w-72 md:self-start md:border-t-0 md:border-l"
>
  <div class="flex flex-col gap-4 px-4 py-4">
    <TicketRuntimeStateCard
      {ticket}
      {resumingRetry}
      {resettingWorkspace}
      {onResumeRetry}
      {onResetWorkspace}
    />

    <Separator />

    <section class="space-y-3">
      <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
        Details
      </span>
      <div class="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2.5 text-xs">
        {#if ticket.workflow}
          <div class="text-muted-foreground flex items-center gap-1.5">
            <Workflow class="size-3.5" />
            <span>Workflow</span>
          </div>
          <div class="text-foreground break-words">{ticket.workflow.name}</div>
        {/if}

        <div class="text-muted-foreground flex items-center gap-1.5">
          <Bot class="size-3.5" />
          <span>Agent</span>
        </div>
        <div class="flex items-center gap-1.5">
          {#if ticket.assignedAgent}
            <span class="inline-block size-1.5 rounded-full bg-green-400" title="Online"></span>
            <span class="text-foreground break-words">{ticket.assignedAgent.name}</span>
            <Badge variant="outline" class="h-4 shrink-0 py-0 text-[10px]">
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
        </div>

        <div class="text-muted-foreground flex items-center gap-1.5">
          <RotateCcw class="size-3.5" />
          <span>Attempts</span>
        </div>
        <div class="text-foreground">{ticket.attemptCount}</div>

        <div class="text-muted-foreground">Input Tokens</div>
        <div class="text-foreground">{formatCount(ticket.costTokensInput)}</div>

        <div class="text-muted-foreground">Output Tokens</div>
        <div class="text-foreground">{formatCount(ticket.costTokensOutput)}</div>

        <div class="text-muted-foreground flex items-center gap-1.5">
          <User class="size-3.5" />
          <span>Created by</span>
        </div>
        <div class="text-foreground break-all">{ticket.createdBy}</div>

        <div class="text-muted-foreground flex items-center gap-1.5">
          <Calendar class="size-3.5" />
          <span>Created</span>
        </div>
        <div class="text-foreground">{formatRelativeTime(ticket.createdAt)}</div>
      </div>
    </section>

    <Separator />

    <TicketExternalLinks
      links={ticket.externalLinks}
      creating={creatingExternalLink}
      deletingId={deletingExternalLinkId}
      onCreate={onCreateExternalLink}
      onDelete={onDeleteExternalLink}
    />

    <Separator />

    <TicketDependencies
      {ticket}
      availableTickets={dependencyCandidates}
      {creatingDependency}
      {deletingDependencyId}
      {onAddDependency}
      {onDeleteDependency}
    />

    <Separator />

    <TicketRepos
      {ticket}
      repos={repoOptions}
      {creatingRepoScope}
      {updatingRepoScopeId}
      {deletingRepoScopeId}
      {onCreateScope}
      {onUpdateScope}
      {onDeleteScope}
    />

    {#if hooks.length > 0}
      <Separator />
      <TicketHooks {hooks} />
    {/if}
  </div>
</div>
