import {
  createTicketRepoScope,
  deleteTicketRepoScope,
  updateTicket,
  updateTicketRepoScope,
} from '$lib/api/openase'
import {
  handleCreateTicketComment,
  handleDeleteTicketComment,
  loadTicketCommentHistory,
  handleUpdateTicketComment,
} from './drawer-comment-actions'
import { runTicketDrawerMutation } from './drawer-mutation'
import {
  handleCreateExternalLinkAction,
  handleDeleteExternalLinkAction,
} from './drawer-external-link-actions'
import {
  handleAddDependencyAction,
  handleDeleteDependencyAction,
  handleResetWorkspaceAction,
  handleResumeRetryAction,
  handleSaveFieldsAction,
} from './drawer-ticket-actions'
import {
  buildCreateRepoScopeMutation,
  buildDeleteRepoScopeMutation,
  buildUpdateRepoScopeMutation,
} from './repo-scope-builders'
import {
  nextRepoScopesForMutation,
  type DependencyDraft,
  type ExistingRepoScopeDraft,
  type RepoScopeDraft,
  type TicketFieldDraft,
} from './mutation-shared'
import type { TicketDetail, TicketExternalLinkDraft } from './types'
import type { TicketDrawerState } from './drawer-state.svelte'

type BuildDrawerMutation = (ticket: TicketDetail) => {
  ticket: TicketDetail
  projectId?: string | null
  ticketId?: string | null
  load: TicketDrawerState['load']
  applyTicket(nextTicket: TicketDetail): void
  clearMessages(): void
  setError(message: string): void
  setNotice(message: string): void
}

type DrawerActionInput = {
  getProjectId(): string | null | undefined
  getTicketId(): string | null | undefined
  drawerState: TicketDrawerState
  buildDrawerMutation: BuildDrawerMutation
}

export function createTicketDrawerActions(input: DrawerActionInput) {
  const getProjectId = () => input.getProjectId()
  const getTicketId = () => input.getTicketId()

  return {
    handleSaveFields(draft: TicketFieldDraft) {
      return handleSaveFieldsAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        draft,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    async handlePriorityChange(priority: TicketDetail['priority']) {
      const ticket = input.drawerState.ticket
      const ticketId = getTicketId()
      if (!ticket || !ticketId || priority === ticket.priority) return

      await runTicketDrawerMutation({
        ...input.buildDrawerMutation(ticket),
        start: () => {
          input.drawerState.savingFields = true
        },
        finish: () => {
          input.drawerState.savingFields = false
        },
        optimisticUpdate: (currentTicket) => ({
          ...currentTicket,
          priority,
        }),
        mutate: () => updateTicket(ticketId, { priority }),
        successMessage: 'Priority updated.',
      })
    },
    async handleArchive() {
      const ticket = input.drawerState.ticket
      const ticketId = getTicketId()
      if (!ticket || !ticketId) return

      await runTicketDrawerMutation({
        ...input.buildDrawerMutation(ticket),
        start: () => {
          input.drawerState.archiving = true
        },
        finish: () => {
          input.drawerState.archiving = false
        },
        optimisticUpdate: (currentTicket) => ({
          ...currentTicket,
          archived: true,
        }),
        mutate: () => updateTicket(ticketId, { archived: true }),
        successMessage: 'Ticket archived.',
      })
    },
    handleAddDependency(draft: DependencyDraft) {
      return handleAddDependencyAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        draft,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    handleDeleteDependency(dependencyId: string) {
      return handleDeleteDependencyAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        dependencyId,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    handleResetWorkspace() {
      return handleResetWorkspaceAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    handleCreateExternalLink(draft: TicketExternalLinkDraft) {
      return handleCreateExternalLinkAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        draft,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    handleDeleteExternalLink(linkId: string) {
      return handleDeleteExternalLinkAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        linkId,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    async handleCreateRepoScope(draft: RepoScopeDraft) {
      const ticket = input.drawerState.ticket
      const projectId = getProjectId()
      const ticketId = getTicketId()
      if (!ticket || !projectId || !ticketId) return false

      const mutation = buildCreateRepoScopeMutation(ticket, input.drawerState.repoOptions, draft)
      if (!mutation.ok) {
        input.drawerState.setMutationError(mutation.error)
        return false
      }

      return runTicketDrawerMutation({
        ...input.buildDrawerMutation(ticket),
        start: () => {
          input.drawerState.creatingRepoScope = true
        },
        finish: () => {
          input.drawerState.creatingRepoScope = false
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
    },
    async handleUpdateRepoScope(scopeId: string, draft: ExistingRepoScopeDraft) {
      const ticket = input.drawerState.ticket
      const projectId = getProjectId()
      const ticketId = getTicketId()
      if (!ticket || !projectId || !ticketId) return

      const mutation = buildUpdateRepoScopeMutation(ticket, scopeId, draft)
      if (!mutation.ok) {
        input.drawerState.setMutationError(mutation.error)
        return
      }

      await runTicketDrawerMutation({
        ...input.buildDrawerMutation(ticket),
        start: () => {
          input.drawerState.updatingRepoScopeId = scopeId
        },
        finish: () => {
          input.drawerState.updatingRepoScopeId = null
        },
        optimisticUpdate: mutation.value.optimisticUpdate,
        mutate: () => updateTicketRepoScope(projectId, ticketId, scopeId, mutation.value.body),
        successMessage: mutation.value.successMessage,
      })
    },
    async handleDeleteRepoScope(scopeId: string) {
      const ticket = input.drawerState.ticket
      const projectId = getProjectId()
      const ticketId = getTicketId()
      if (!ticket || !projectId || !ticketId) return

      const mutation = buildDeleteRepoScopeMutation(ticket, scopeId)
      if (!mutation.ok) {
        input.drawerState.setMutationError(mutation.error)
        return
      }

      await runTicketDrawerMutation({
        ...input.buildDrawerMutation(ticket),
        start: () => {
          input.drawerState.deletingRepoScopeId = scopeId
        },
        finish: () => {
          input.drawerState.deletingRepoScopeId = null
        },
        optimisticUpdate: mutation.value.optimisticUpdate,
        mutate: () => deleteTicketRepoScope(projectId, ticketId, scopeId),
        successMessage: mutation.value.successMessage,
      })
    },
    handleCreateComment(body: string) {
      return handleCreateTicketComment({
        projectId: getProjectId(),
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        body,
      })
    },
    handleUpdateComment(commentId: string, body: string) {
      return handleUpdateTicketComment({
        projectId: getProjectId(),
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        commentId,
        body,
      })
    },
    handleDeleteComment(commentId: string) {
      return handleDeleteTicketComment({
        projectId: getProjectId(),
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        commentId,
      })
    },
    handleResumeRetry() {
      return handleResumeRetryAction({
        ticketId: getTicketId(),
        drawerState: input.drawerState,
        buildDrawerMutation: input.buildDrawerMutation,
      })
    },
    handleLoadCommentHistory(commentId: string) {
      return loadTicketCommentHistory({
        ticketId: getTicketId(),
        commentId,
      })
    },
  }
}
