<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import {
    addTicketDependency,
    createTicketRepoScope,
    deleteTicketDependency,
    deleteTicketRepoScope,
    updateTicket,
    updateTicketRepoScope,
  } from '$lib/api/openase'
  import { runTicketDrawerMutation } from '../drawer-mutations'
  import { createTicketCommentHandlers } from '../comment-mutations'
  import { statusSync } from '$lib/features/statuses/public'
  import { createTicketDrawerState } from '../drawer-state.svelte'
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
  import TicketDrawerShell from './ticket-drawer-shell.svelte'
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
  async function reloadDetail(activeProjectId: string, activeTicketId: string) {
    await drawerState.load(activeProjectId, activeTicketId, {
      background: true,
      preserveMessages: true,
    })
  }

  const commentHandlers = createTicketCommentHandlers({
    getProjectId: () => projectId,
    getTicketId: () => ticketId,
    reload: reloadDetail,
    clearMessages: drawerState.clearMutationMessages,
    setError: drawerState.setMutationError,
    setNotice: drawerState.setMutationNotice,
    setCreatingComment: (value) => (drawerState.creatingComment = value),
    setUpdatingCommentId: (value) => (drawerState.updatingCommentId = value),
    setDeletingCommentId: (value) => (drawerState.deletingCommentId = value),
  })

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
      void reloadDetail(projectId, ticketId)
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

    await runTicketDrawerMutation({
      ticket,
      optimisticUpdate,
      mutate,
      reload: () => reloadDetail(projectId, ticketId),
      applyTicket: (nextTicket) => (drawerState.ticket = nextTicket),
      clearMessages: drawerState.clearMutationMessages,
      setError: drawerState.setMutationError,
      setNotice: drawerState.setMutationNotice,
      successMessage,
      start,
      finish,
    })
  }
</script>

<TicketDrawerShell
  bind:open
  title={drawerState.ticket?.identifier ?? 'Ticket detail'}
  loading={drawerState.loading}
  error={drawerState.error}
>
  {#if drawerState.ticket}
    <TicketDrawerContent
      ticket={drawerState.ticket}
      hooks={drawerState.hooks}
      activities={drawerState.activities}
      comments={drawerState.comments}
      statuses={drawerState.statuses}
      dependencyCandidates={drawerState.dependencyCandidates}
      repoOptions={drawerState.repoOptions}
      mutationError={drawerState.mutationError}
      mutationNotice={drawerState.mutationNotice}
      savingFields={drawerState.savingFields}
      creatingDependency={drawerState.creatingDependency}
      deletingDependencyId={drawerState.deletingDependencyId}
      creatingComment={drawerState.creatingComment}
      updatingCommentId={drawerState.updatingCommentId}
      deletingCommentId={drawerState.deletingCommentId}
      creatingRepoScope={drawerState.creatingRepoScope}
      updatingRepoScopeId={drawerState.updatingRepoScopeId}
      deletingRepoScopeId={drawerState.deletingRepoScopeId}
      onClose={() => appStore.closeRightPanel()}
      onSaveFields={handleSaveFields}
      onAddDependency={handleAddDependency}
      onDeleteDependency={handleDeleteDependency}
      onCreateComment={commentHandlers.create}
      onUpdateComment={commentHandlers.update}
      onDeleteComment={commentHandlers.delete}
      onCreateScope={handleCreateRepoScope}
      onUpdateScope={handleUpdateRepoScope}
      onDeleteScope={handleDeleteRepoScope}
    />
  {/if}
</TicketDrawerShell>
