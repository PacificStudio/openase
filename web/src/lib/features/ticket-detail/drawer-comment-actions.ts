import { ApiError } from '$lib/api/client'
import { createTicketComment, deleteTicketComment, updateTicketComment } from '$lib/api/openase'

type LoadOptions = {
  background?: boolean
  preserveMessages?: boolean
}

type TicketDrawerCommentState = {
  creatingComment: boolean
  updatingCommentId: string | null
  deletingCommentId: string | null
  clearMutationMessages: () => void
  setMutationError: (message: string) => void
  setMutationNotice: (message: string) => void
  load: (projectId: string, ticketId: string, options?: LoadOptions) => Promise<void>
}

export async function handleCreateTicketComment({
  projectId,
  ticketId,
  drawerState,
  body,
}: {
  projectId?: string | null
  ticketId?: string | null
  drawerState: TicketDrawerCommentState
  body: string
}) {
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

export async function handleUpdateTicketComment({
  projectId,
  ticketId,
  drawerState,
  commentId,
  body,
}: {
  projectId?: string | null
  ticketId?: string | null
  drawerState: TicketDrawerCommentState
  commentId: string
  body: string
}) {
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

export async function handleDeleteTicketComment({
  projectId,
  ticketId,
  drawerState,
  commentId,
}: {
  projectId?: string | null
  ticketId?: string | null
  drawerState: TicketDrawerCommentState
  commentId: string
}) {
  if (!projectId || !ticketId) return false

  drawerState.deletingCommentId = commentId
  drawerState.clearMutationMessages()

  try {
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
