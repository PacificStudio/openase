<script lang="ts">
  import type { StreamConnectionState } from '$lib/api/sse'
  import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat/project-ai-focus'
  import { appStore } from '$lib/stores/app.svelte'
  import { EphemeralChatPanel } from '$lib/features/chat'
  import TicketDrawerMainTabs from './ticket-drawer-main-tabs.svelte'
  import TicketHeader from './ticket-header.svelte'
  import TicketDrawerSidebar from './ticket-drawer-sidebar.svelte'
  import type { DependencyDraft } from '../mutation-shared'
  import type {
    HookExecution,
    TicketCommentRevision,
    TicketDetail,
    TicketReferenceOption,
    TicketRepoOption,
    TicketRun,
    TicketRunTranscriptBlock,
    TicketStatusOption,
    TicketTimelineItem,
  } from '../types'

  let {
    ticket,
    projectId,
    hooks,
    timeline,
    runs = [],
    currentRun = null,
    runBlocks = [],
    loadingRunId = null,
    runStreamState = 'idle',
    recoveringRunTranscript = false,
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
    onSelectRun,
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
    runs?: TicketRun[]
    currentRun?: TicketRun | null
    runBlocks?: TicketRunTranscriptBlock[]
    loadingRunId?: string | null
    runStreamState?: StreamConnectionState
    recoveringRunTranscript?: boolean
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
    onSelectRun?: (runId: string) => Promise<void> | void
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
    }) => Promise<boolean> | boolean
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
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

  let assistantOpen = $state(false)
  let previousTicketId = ''
  const projectAIFocusOwner = 'ticket-drawer'

  $effect(() => {
    if (ticket.id === previousTicketId) {
      return
    }

    previousTicketId = ticket.id
    assistantOpen = false
  })

  $effect(() => {
    if (!projectId || !ticket.id) {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
      return
    }

    appStore.setProjectAssistantFocus(
      projectAIFocusOwner,
      {
        kind: 'ticket',
        projectId,
        ticketId: ticket.id,
        ticketIdentifier: ticket.identifier,
        ticketTitle: ticket.title,
        ticketStatus: ticket.status.name,
        selectedArea: 'detail',
      },
      PROJECT_AI_FOCUS_PRIORITY.overlay,
    )

    return () => {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
    }
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
  <div class="flex flex-1 flex-col overflow-hidden border-r">
    <TicketDrawerMainTabs
      {ticket}
      {timeline}
      {runs}
      {currentRun}
      {runBlocks}
      {loadingRunId}
      {runStreamState}
      {recoveringRunTranscript}
      {savingFields}
      {creatingComment}
      {updatingCommentId}
      {deletingCommentId}
      {resumingRetry}
      {onSaveFields}
      {onSelectRun}
      {onResumeRetry}
      {onCreateComment}
      {onUpdateComment}
      {onDeleteComment}
      {onLoadCommentHistory}
    />
  </div>
  <TicketDrawerSidebar
    {ticket}
    {hooks}
    {dependencyCandidates}
    {repoOptions}
    {creatingDependency}
    {deletingDependencyId}
    {creatingExternalLink}
    {deletingExternalLinkId}
    {creatingRepoScope}
    {updatingRepoScopeId}
    {deletingRepoScopeId}
    {resumingRetry}
    {onResumeRetry}
    {onAddDependency}
    {onDeleteDependency}
    {onCreateExternalLink}
    {onDeleteExternalLink}
    {onCreateScope}
    {onUpdateScope}
    {onDeleteScope}
  />
</div>
