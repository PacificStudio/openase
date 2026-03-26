<script lang="ts">
  import { Tabs, TabsContent, TabsList, TabsTrigger } from '$ui/tabs'
  import TicketDiscussion from './ticket-discussion.svelte'
  import TicketHeader from './ticket-header.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketSummary from './ticket-summary.svelte'
  import type {
    HookExecution,
    TicketActivity,
    TicketComment,
    TicketDetail,
    TicketReferenceOption,
    TicketRepoOption,
    TicketStatusOption,
  } from '../types'

  let {
    ticket,
    hooks,
    comments,
    activities,
    statuses,
    dependencyCandidates,
    repoOptions,
    mutationError = '',
    mutationNotice = '',
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
    onClose,
    onSaveFields,
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
  }: {
    ticket: TicketDetail
    hooks: HookExecution[]
    comments: TicketComment[]
    activities: TicketActivity[]
    statuses: TicketStatusOption[]
    dependencyCandidates: TicketReferenceOption[]
    repoOptions: TicketRepoOption[]
    mutationError?: string
    mutationNotice?: string
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
    onClose?: () => void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onAddDependency?: (draft: { targetTicketId: string; relation: string }) => void
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
      isPrimaryScope: boolean
    }) => void
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
        prStatus: string
        ciStatus: string
        isPrimaryScope: boolean
      },
    ) => void
    onDeleteScope?: (scopeId: string) => void
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
  } = $props()
</script>

<TicketHeader {ticket} {statuses} {savingFields} {onClose} {onSaveFields} />

{#if mutationNotice}
  <div class="border-border bg-muted/40 mx-6 mt-4 rounded-md border px-3 py-2 text-xs">
    {mutationNotice}
  </div>
{/if}

{#if mutationError}
  <div
    class="border-destructive/30 bg-destructive/10 text-destructive mx-6 mt-4 rounded-md border px-3 py-2 text-xs"
  >
    {mutationError}
  </div>
{/if}

<Tabs value="summary" class="flex flex-1 flex-col overflow-hidden">
  <TabsList class="mx-6 mt-4 shrink-0">
    <TabsTrigger value="summary">Summary</TabsTrigger>
    <TabsTrigger value="code">Code</TabsTrigger>
    <TabsTrigger value="hooks">Hooks</TabsTrigger>
    <TabsTrigger value="activity">Activity</TabsTrigger>
  </TabsList>

  <div class="flex-1 overflow-y-auto">
    <TabsContent value="summary" class="mt-0">
      <TicketSummary
        {ticket}
        availableTickets={dependencyCandidates}
        {savingFields}
        {creatingDependency}
        {deletingDependencyId}
        {creatingExternalLink}
        {deletingExternalLinkId}
        {onSaveFields}
        {onAddDependency}
        {onDeleteDependency}
        {onCreateExternalLink}
        {onDeleteExternalLink}
      />
    </TabsContent>

    <TabsContent value="code" class="mt-0">
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
    </TabsContent>

    <TabsContent value="hooks" class="mt-0">
      <TicketHooks {hooks} />
    </TabsContent>

    <TabsContent value="activity" class="mt-0">
      <TicketDiscussion
        {comments}
        {activities}
        {creatingComment}
        {updatingCommentId}
        {deletingCommentId}
        {onCreateComment}
        {onUpdateComment}
        {onDeleteComment}
      />
    </TabsContent>
  </div>
</Tabs>
