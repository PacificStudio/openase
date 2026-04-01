<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { EphemeralChatPanel } from '$lib/features/chat'
  import { Badge } from '$ui/badge'
  import { Separator } from '$ui/separator'
  import Bot from '@lucide/svelte/icons/bot'
  import Calendar from '@lucide/svelte/icons/calendar'
  import DollarSign from '@lucide/svelte/icons/dollar-sign'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import User from '@lucide/svelte/icons/user'
  import Workflow from '@lucide/svelte/icons/workflow'
  import { cn, formatCount, formatCurrency, formatRelativeTime } from '$lib/utils'
  import TicketCommentsThread from './ticket-comments-thread.svelte'
  import TicketDependencies from './ticket-dependencies.svelte'
  import TicketExternalLinks from './ticket-external-links.svelte'
  import TicketHeader from './ticket-header.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketRuntimeStateCard from './ticket-runtime-state-card.svelte'
  import type { DependencyDraft } from '../mutation-shared'
  import type {
    HookExecution,
    TicketCommentRevision,
    TicketDetail,
    TicketReferenceOption,
    TicketRepoOption,
    TicketStatusOption,
    TicketTimelineItem,
  } from '../types'

  let {
    ticket,
    projectId,
    hooks,
    timeline,
    statuses,
    dependencyCandidates,
    repoOptions,
    savingFields = false,
    creatingDependency = false,
    deletingDependencyId = null,
    creatingExternalLink = false,
    deletingExternalLinkId = null,
    creatingRepoScope = false,
    updatingRepoScopeId = null,
    deletingRepoScopeId = null,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    resumingRetry = false,
    onClose,
    onSaveFields,
    onResumeRetry,
    onAddDependency,
    onDeleteDependency,
    onCreateExternalLink,
    onDeleteExternalLink,
    onCreateScope,
    onUpdateScope,
    onDeleteScope,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
    onLoadCommentHistory,
  }: {
    ticket: TicketDetail
    projectId: string
    hooks: HookExecution[]
    timeline: TicketTimelineItem[]
    statuses: TicketStatusOption[]
    dependencyCandidates: TicketReferenceOption[]
    repoOptions: TicketRepoOption[]
    savingFields?: boolean
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    creatingExternalLink?: boolean
    deletingExternalLinkId?: string | null
    creatingRepoScope?: boolean
    updatingRepoScopeId?: string | null
    deletingRepoScopeId?: string | null
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    resumingRetry?: boolean
    onClose?: () => void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onResumeRetry?: () => Promise<void> | void
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
      prStatus: string
      ciStatus: string
    }) => Promise<boolean> | boolean
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
        prStatus: string
        ciStatus: string
      },
    ) => void
    onDeleteScope?: (scopeId: string) => void
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
    onLoadCommentHistory?: (
      commentId: string,
    ) => Promise<TicketCommentRevision[]> | TicketCommentRevision[]
  } = $props()

  const costPercent = $derived.by(() =>
    ticket.budgetUsd > 0 ? Math.round((ticket.costAmount / ticket.budgetUsd) * 100) : 0,
  )
  const costOverBudget = $derived(costPercent > 80)
  let assistantOpen = $state(false)
  let previousTicketId = ''

  $effect(() => {
    if (ticket.id === previousTicketId) {
      return
    }

    previousTicketId = ticket.id
    assistantOpen = false
  })
</script>

<TicketHeader
  {ticket}
  {statuses}
  {savingFields}
  {onClose}
  {onSaveFields}
  {assistantOpen}
  onToggleAssistant={() => (assistantOpen = !assistantOpen)}
/>

{#if assistantOpen}
  <div class="border-border h-[26rem] border-b">
    <EphemeralChatPanel
      source="ticket_detail"
      organizationId={appStore.currentOrg?.id ?? ''}
      defaultProviderId={appStore.currentProject?.default_agent_provider_id ?? null}
      context={{ projectId, ticketId: ticket.id }}
      title="Ticket AI"
      placeholder="Ask about failures, execution history, or how to split work…"
    />
  </div>
{/if}

<div class="flex flex-1 flex-col overflow-hidden md:flex-row">
  <TicketCommentsThread
    {ticket}
    {timeline}
    {savingFields}
    {creatingComment}
    {updatingCommentId}
    {deletingCommentId}
    {onSaveFields}
    {onCreateComment}
    {onUpdateComment}
    {onDeleteComment}
    {onLoadCommentHistory}
  />

  <!-- Right sidebar: metadata -->
  <div class="border-border w-full shrink-0 overflow-y-auto border-t md:w-80 md:border-t-0">
    <div class="flex flex-col gap-5 px-5 py-5">
      <TicketRuntimeStateCard {ticket} {resumingRetry} {onResumeRetry} />

      <Separator />

      <!-- Metadata grid -->
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

      <!-- External Links -->
      <TicketExternalLinks
        links={ticket.externalLinks}
        creating={creatingExternalLink}
        deletingId={deletingExternalLinkId}
        onCreate={onCreateExternalLink}
        onDelete={onDeleteExternalLink}
      />

      <Separator />

      <!-- Dependencies -->
      <TicketDependencies
        {ticket}
        availableTickets={dependencyCandidates}
        {creatingDependency}
        {deletingDependencyId}
        {onAddDependency}
        {onDeleteDependency}
      />

      <Separator />

      <!-- Repositories -->
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

      <!-- Hooks -->
      {#if hooks.length > 0}
        <Separator />
        <TicketHooks {hooks} />
      {/if}
    </div>
  </div>
</div>
