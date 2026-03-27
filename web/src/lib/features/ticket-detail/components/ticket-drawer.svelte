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

    return connectTicketDetailStreams(projectId, ticketId, () => {
      void drawerState.load(projectId, ticketId, {
        background: true,
        preserveMessages: true,
      })
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
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col p-0 sm:max-w-[60vw]"
    showCloseButton={false}
  >
    <SheetHeader class="sr-only">
      <SheetTitle>{drawerState.ticket?.identifier ?? 'Ticket detail'}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    {#if drawerState.loading}
      <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
        Loading ticket detail…
      </div>
    {:else if drawerState.error}
      <div
        class="text-destructive flex flex-1 items-center justify-center px-6 text-center text-sm"
      >
        {drawerState.error}
      </div>
    {:else if drawerState.ticket}
      <TicketDrawerContent
        ticket={drawerState.ticket}
        hooks={drawerState.hooks}
        comments={drawerState.comments}
        activities={drawerState.activities}
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
        onClose={appStore.closeRightPanel}
        onSaveFields={handleSaveFields}
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
      />
    {/if}
  </SheetContent>
</Sheet>
