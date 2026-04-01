<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import {
    createTicketRepoScope,
    deleteTicketRepoScope,
    updateTicketRepoScope,
  } from '$lib/api/openase'
  import {
    handleCreateTicketComment,
    handleDeleteTicketComment,
    loadTicketCommentHistory,
    handleUpdateTicketComment,
  } from '../drawer-comment-actions'
  import { statusSync } from '$lib/features/statuses/public'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { createTicketDrawerState } from '../drawer-state.svelte'
  import { runTicketDrawerMutation } from '../drawer-mutation'
  import {
    buildCreateRepoScopeMutation,
    buildDeleteRepoScopeMutation,
    buildUpdateRepoScopeMutation,
  } from '../repo-scope-builders'
  import {
    nextRepoScopesForMutation,
    type DependencyDraft,
    type ExistingRepoScopeDraft,
    type RepoScopeDraft,
    type TicketFieldDraft,
  } from '../mutation-shared'
  import {
    handleCreateExternalLinkAction,
    handleDeleteExternalLinkAction,
  } from '../drawer-external-link-actions'
  import {
    handleAddDependencyAction,
    handleDeleteDependencyAction,
    handleResumeRetryAction,
    handleSaveFieldsAction,
  } from '../drawer-ticket-actions'
  import { connectTicketDetailStreams } from '../streams'
  import TicketDrawerContent from './ticket-drawer-content.svelte'
  import type { TicketDetail, TicketExternalLinkDraft } from '../types'

  let {
    open = $bindable(false),
    projectId,
    ticketId,
    onOpenChange,
  }: {
    open?: boolean
    projectId?: string | null
    ticketId?: string | null
    onOpenChange?: (open: boolean) => void
  } = $props()

  const drawerState = createTicketDrawerState()

  function buildDrawerMutation(ticket: TicketDetail) {
    return {
      ticket,
      projectId,
      ticketId,
      load: drawerState.load,
      applyTicket: (nextTicket: TicketDetail) => {
        drawerState.ticket = nextTicket
      },
      clearMessages: drawerState.clearMutationMessages,
      setError: drawerState.setMutationError,
      setNotice: drawerState.setMutationNotice,
    }
  }

  $effect(() => {
    onOpenChange?.(open)
  })

  $effect(() => {
    const currentProjectId = projectId
    const currentTicketId = ticketId
    const statusVersion = statusSync.version
    if (!open || !currentProjectId || !currentTicketId) {
      if (!open) drawerState.reset()
      return
    }

    void statusVersion
    void drawerState.load(currentProjectId, currentTicketId)
  })

  $effect(() => {
    if (!open || !projectId || !ticketId) {
      return
    }

    let runStreamNeedsRecovery = false

    return connectTicketDetailStreams(projectId, ticketId, {
      onRelevantEvent: () => {
        void drawerState.refreshTimeline(projectId, ticketId)
      },
      onRunFrame: (frame) => {
        drawerState.applyRunStreamFrame(frame)
      },
      onRunStateChange: (state) => {
        drawerState.setRunStreamState(state)
        if (state === 'retrying') {
          runStreamNeedsRecovery = true
          return
        }
        if (state === 'live' && runStreamNeedsRecovery) {
          runStreamNeedsRecovery = false
          void drawerState.recoverRunTranscript(projectId, ticketId)
        }
      },
    })
  })

  async function handleSaveFields(draft: TicketFieldDraft) {
    await handleSaveFieldsAction({
      ticketId,
      drawerState,
      draft,
      buildDrawerMutation,
    })
  }

  async function handleAddDependency(draft: DependencyDraft) {
    return await handleAddDependencyAction({
      ticketId,
      drawerState,
      draft,
      buildDrawerMutation,
    })
  }

  async function handleDeleteDependency(dependencyId: string) {
    await handleDeleteDependencyAction({
      ticketId,
      drawerState,
      dependencyId,
      buildDrawerMutation,
    })
  }

  async function handleCreateExternalLink(draft: TicketExternalLinkDraft) {
    return handleCreateExternalLinkAction({
      ticketId,
      drawerState,
      draft,
      buildDrawerMutation,
    })
  }

  async function handleDeleteExternalLink(linkId: string) {
    await handleDeleteExternalLinkAction({
      ticketId,
      drawerState,
      linkId,
      buildDrawerMutation,
    })
  }

  async function handleCreateRepoScope(draft: RepoScopeDraft) {
    const ticket = drawerState.ticket
    if (!ticket || !projectId || !ticketId) return false

    const mutation = buildCreateRepoScopeMutation(drawerState.repoOptions, draft)
    if (!mutation.ok) {
      drawerState.setMutationError(mutation.error)
      return false
    }

    return await runTicketDrawerMutation({
      ...buildDrawerMutation(ticket),
      start: () => {
        drawerState.creatingRepoScope = true
      },
      finish: () => {
        drawerState.creatingRepoScope = false
      },
      optimisticUpdate: (currentTicket) => ({
        ...currentTicket,
        repoScopes: nextRepoScopesForMutation(
          currentTicket.repoScopes,
          mutation.value.optimisticScope,
        ),
      }),
      mutate: () => createTicketRepoScope(projectId, ticketId, mutation.value.body),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleUpdateRepoScope(scopeId: string, draft: ExistingRepoScopeDraft) {
    const ticket = drawerState.ticket
    if (!ticket || !projectId || !ticketId) return

    const mutation = buildUpdateRepoScopeMutation(ticket, scopeId, draft)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)

    await runTicketDrawerMutation({
      ...buildDrawerMutation(ticket),
      start: () => {
        drawerState.updatingRepoScopeId = scopeId
      },
      finish: () => {
        drawerState.updatingRepoScopeId = null
      },
      optimisticUpdate: mutation.value.optimisticUpdate,
      mutate: () => updateTicketRepoScope(projectId, ticketId, scopeId, mutation.value.body),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleDeleteRepoScope(scopeId: string) {
    const ticket = drawerState.ticket
    if (!ticket || !projectId || !ticketId) return

    const mutation = buildDeleteRepoScopeMutation(ticket, scopeId)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)

    await runTicketDrawerMutation({
      ...buildDrawerMutation(ticket),
      start: () => {
        drawerState.deletingRepoScopeId = scopeId
      },
      finish: () => {
        drawerState.deletingRepoScopeId = null
      },
      optimisticUpdate: mutation.value.optimisticUpdate,
      mutate: () => deleteTicketRepoScope(projectId, ticketId, scopeId),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleCreateComment(body: string) {
    return handleCreateTicketComment({ projectId, ticketId, drawerState, body })
  }

  async function handleUpdateComment(commentId: string, body: string) {
    return handleUpdateTicketComment({ projectId, ticketId, drawerState, commentId, body })
  }

  async function handleDeleteComment(commentId: string) {
    return handleDeleteTicketComment({ projectId, ticketId, drawerState, commentId })
  }

  const handleResumeRetry = () =>
    handleResumeRetryAction({ ticketId, drawerState, buildDrawerMutation })

  const handleLoadCommentHistory = (commentId: string) =>
    loadTicketCommentHistory({ ticketId, commentId })
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col p-0 sm:max-w-[80vw]"
    showCloseButton={false}
  >
    <SheetHeader class="sr-only">
      <SheetTitle>{drawerState.ticket?.identifier ?? 'Ticket detail'}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    {#if drawerState.loading}
      <!-- Skeleton: ticket detail layout -->
      <div class="flex flex-col overflow-hidden">
        <!-- Skeleton header -->
        <div class="border-border flex shrink-0 items-center gap-3 border-b px-5 py-3">
          <div class="bg-muted h-5 w-5 animate-pulse rounded"></div>
          <div class="bg-muted h-5 w-16 animate-pulse rounded"></div>
          <div class="bg-muted h-4 w-48 animate-pulse rounded"></div>
          <div class="ml-auto flex items-center gap-2">
            <div class="bg-muted h-6 w-20 animate-pulse rounded-full"></div>
            <div class="bg-muted h-6 w-6 animate-pulse rounded"></div>
          </div>
        </div>

        <div class="flex flex-1 overflow-hidden">
          <!-- Skeleton left: timeline -->
          <div class="flex min-w-0 flex-1 flex-col gap-4 p-5">
            {#each { length: 4 } as _}
              <div class="flex gap-3">
                <div class="bg-muted size-7 shrink-0 animate-pulse rounded-full"></div>
                <div class="flex-1 space-y-2">
                  <div class="flex items-center gap-2">
                    <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
                    <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
                  </div>
                  <div class="bg-muted h-16 w-full animate-pulse rounded-lg"></div>
                </div>
              </div>
            {/each}
          </div>

          <!-- Skeleton right: metadata sidebar -->
          <div class="border-border w-80 shrink-0 border-l">
            <div class="flex flex-col gap-5 px-5 py-5">
              <!-- Runtime state -->
              <div class="bg-muted h-16 w-full animate-pulse rounded-lg"></div>

              <div class="bg-border h-px"></div>

              <!-- Details grid -->
              <div class="space-y-3">
                <div class="bg-muted h-3 w-12 animate-pulse rounded"></div>
                {#each { length: 5 } as _}
                  <div class="flex items-center gap-3">
                    <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
                    <div class="bg-muted h-3 w-24 animate-pulse rounded"></div>
                  </div>
                {/each}
              </div>

              <div class="bg-border h-px"></div>

              <!-- Links section -->
              <div class="space-y-2">
                <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
                <div class="bg-muted h-8 w-full animate-pulse rounded-md"></div>
              </div>

              <div class="bg-border h-px"></div>

              <!-- Dependencies -->
              <div class="space-y-2">
                <div class="bg-muted h-3 w-24 animate-pulse rounded"></div>
                <div class="bg-muted h-8 w-full animate-pulse rounded-md"></div>
              </div>

              <div class="bg-border h-px"></div>

              <!-- Repos -->
              <div class="space-y-2">
                <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
                <div class="bg-muted h-8 w-full animate-pulse rounded-md"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    {:else if drawerState.error}
      <div
        class="text-destructive flex flex-1 items-center justify-center px-6 text-center text-sm"
      >
        {drawerState.error}
      </div>
    {:else if drawerState.ticket}
      <TicketDrawerContent
        projectId={projectId ?? ''}
        ticket={drawerState.ticket}
        hooks={drawerState.hooks}
        timeline={drawerState.timeline}
        runs={drawerState.runs}
        currentRun={drawerState.currentRun}
        runBlocks={drawerState.runBlocks}
        loadingRunId={drawerState.loadingRunId}
        runStreamState={drawerState.runStreamState}
        recoveringRunTranscript={drawerState.recoveringRunTranscript}
        statuses={drawerState.statuses}
        dependencyCandidates={drawerState.dependencyCandidates}
        repoOptions={drawerState.repoOptions}
        savingFields={drawerState.savingFields}
        creatingDependency={drawerState.creatingDependency}
        deletingDependencyId={drawerState.deletingDependencyId}
        creatingExternalLink={drawerState.creatingExternalLink}
        deletingExternalLinkId={drawerState.deletingExternalLinkId}
        creatingRepoScope={drawerState.creatingRepoScope}
        updatingRepoScopeId={drawerState.updatingRepoScopeId}
        deletingRepoScopeId={drawerState.deletingRepoScopeId}
        creatingComment={drawerState.creatingComment}
        updatingCommentId={drawerState.updatingCommentId}
        deletingCommentId={drawerState.deletingCommentId}
        resumingRetry={drawerState.resumingRetry}
        onClose={appStore.closeRightPanel}
        onSaveFields={handleSaveFields}
        onSelectRun={(runId) =>
          projectId && ticketId ? drawerState.selectRun(projectId, ticketId, runId) : undefined}
        onResumeRetry={handleResumeRetry}
        onAddDependency={handleAddDependency}
        onDeleteDependency={handleDeleteDependency}
        onCreateExternalLink={handleCreateExternalLink}
        onDeleteExternalLink={handleDeleteExternalLink}
        onCreateScope={handleCreateRepoScope}
        onUpdateScope={handleUpdateRepoScope}
        onDeleteScope={handleDeleteRepoScope}
        onCreateComment={handleCreateComment}
        onUpdateComment={handleUpdateComment}
        onDeleteComment={handleDeleteComment}
        onLoadCommentHistory={handleLoadCommentHistory}
      />
    {/if}
  </SheetContent>
</Sheet>
