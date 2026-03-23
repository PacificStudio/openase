<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    addTicketDependency,
    createTicketComment,
    createTicketRepoScope,
    deleteTicketComment,
    deleteTicketDependency,
    deleteTicketRepoScope,
    updateTicketComment,
    updateTicket,
    updateTicketRepoScope,
  } from '$lib/api/openase'
  import { statusSync } from '$lib/features/statuses/public'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { createTicketDrawerState } from '../drawer-state.svelte'
  import { runOptimisticTicketMutation } from '../optimistic'
  import {
    buildAddDependencyMutation,
    buildDeleteDependencyMutation,
    buildFieldMutation,
  } from '../mutation-builders'
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
  import { connectTicketDetailStreams } from '../streams'
  import TicketDrawerContent from './ticket-drawer-content.svelte'
  import type { TicketDetail } from '../types'
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
    const ticket = drawerState.ticket
    if (!ticket || !ticketId) return

    const mutation = buildFieldMutation(ticket, drawerState.statuses, draft)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)
    if (Object.keys(mutation.value.body).length === 0) {
      return drawerState.setMutationNotice('No ticket field changes to save.')
    }

    await runMutation({
      start: () => {
        drawerState.savingFields = true
      },
      finish: () => {
        drawerState.savingFields = false
      },
      optimisticUpdate: mutation.value.optimisticUpdate,
      mutate: () => updateTicket(ticketId, mutation.value.body),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleAddDependency(draft: DependencyDraft) {
    const ticket = drawerState.ticket
    if (!ticket || !ticketId) return

    const mutation = buildAddDependencyMutation(ticket, drawerState.dependencyCandidates, draft)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)

    await runMutation({
      start: () => {
        drawerState.creatingDependency = true
      },
      finish: () => {
        drawerState.creatingDependency = false
      },
      optimisticUpdate: mutation.value.optimisticUpdate,
      mutate: () => addTicketDependency(ticketId, mutation.value.body),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleDeleteDependency(dependencyId: string) {
    const ticket = drawerState.ticket
    if (!ticket || !ticketId) return

    const mutation = buildDeleteDependencyMutation(ticket, dependencyId)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)

    await runMutation({
      start: () => {
        drawerState.deletingDependencyId = dependencyId
      },
      finish: () => {
        drawerState.deletingDependencyId = null
      },
      optimisticUpdate: mutation.value.optimisticUpdate,
      mutate: () => deleteTicketDependency(ticketId, dependencyId),
      successMessage: mutation.value.successMessage,
    })
  }

  async function handleCreateRepoScope(draft: RepoScopeDraft) {
    const ticket = drawerState.ticket
    if (!ticket || !projectId || !ticketId) return

    const mutation = buildCreateRepoScopeMutation(drawerState.repoOptions, draft)
    if (!mutation.ok) return drawerState.setMutationError(mutation.error)

    await runMutation({
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

    await runMutation({
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

    await runMutation({
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
    if (!projectId || !ticketId) return false

    drawerState.creatingComment = true
    drawerState.clearMutationMessages()

    try {
      await createTicketComment(ticketId, { body })
      drawerState.setMutationNotice('Comment added.')
      await drawerState.load(projectId, ticketId, { background: true, preserveMessages: true })
      return true
    } catch (caughtError) {
      drawerState.setMutationError(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to add comment.',
      )
      return false
    } finally {
      drawerState.creatingComment = false
    }
  }

  async function handleUpdateComment(commentId: string, body: string) {
    if (!projectId || !ticketId) return false

    drawerState.updatingCommentId = commentId
    drawerState.clearMutationMessages()

    try {
      await updateTicketComment(ticketId, commentId, { body })
      drawerState.setMutationNotice('Comment updated.')
      await drawerState.load(projectId, ticketId, { background: true, preserveMessages: true })
      return true
    } catch (caughtError) {
      drawerState.setMutationError(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update comment.',
      )
      return false
    } finally {
      drawerState.updatingCommentId = null
    }
  }

  async function handleDeleteComment(commentId: string) {
    if (!projectId || !ticketId) return false

    drawerState.deletingCommentId = commentId
    drawerState.clearMutationMessages()

    try {
      const previousComments = drawerState.comments
      drawerState.comments = previousComments.filter((comment) => comment.id !== commentId)
      await deleteTicketComment(ticketId, commentId)
      drawerState.setMutationNotice('Comment deleted.')
      await drawerState.load(projectId, ticketId, { background: true, preserveMessages: true })
      return true
    } catch (caughtError) {
      drawerState.setMutationError(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.',
      )
      await drawerState.load(projectId, ticketId, { background: true, preserveMessages: true })
      return false
    } finally {
      drawerState.deletingCommentId = null
    }
  }

  async function runMutation({
    start,
    finish,
    optimisticUpdate,
    mutate,
    successMessage,
  }: {
    start?: () => void
    finish?: () => void
    optimisticUpdate: (currentTicket: TicketDetail) => TicketDetail
    mutate: () => Promise<unknown>
    successMessage: string
  }) {
    const ticket = drawerState.ticket
    if (!ticket || !projectId || !ticketId) return

    start?.()

    try {
      await runOptimisticTicketMutation({
        ticket,
        optimisticUpdate,
        mutate,
        reload: () =>
          drawerState.load(projectId, ticketId, { background: true, preserveMessages: true }),
        applyTicket: (nextTicket) => {
          drawerState.ticket = nextTicket
        },
        clearMessages: drawerState.clearMutationMessages,
        setError: drawerState.setMutationError,
        setNotice: drawerState.setMutationNotice,
        successMessage,
      })
    } finally {
      finish?.()
    }
  }

  function handleClose() {
    appStore.closeRightPanel()
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col p-0 sm:max-w-xl" showCloseButton={false}>
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
        mutationError={drawerState.mutationError}
        mutationNotice={drawerState.mutationNotice}
        savingFields={drawerState.savingFields}
        creatingDependency={drawerState.creatingDependency}
        deletingDependencyId={drawerState.deletingDependencyId}
        creatingRepoScope={drawerState.creatingRepoScope}
        updatingRepoScopeId={drawerState.updatingRepoScopeId}
        deletingRepoScopeId={drawerState.deletingRepoScopeId}
        creatingComment={drawerState.creatingComment}
        updatingCommentId={drawerState.updatingCommentId}
        deletingCommentId={drawerState.deletingCommentId}
        onClose={handleClose}
        onSaveFields={handleSaveFields}
        onAddDependency={handleAddDependency}
        onDeleteDependency={handleDeleteDependency}
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
