import { ApiError } from '$lib/api/client'
import { type TicketCommentRevisionRecord } from '$lib/api/contracts'
import {
  createTicketComment,
  deleteTicketComment,
  listTicketCommentRevisions,
  updateTicketComment,
} from '$lib/api/openase'
import type { TicketCommentRevision } from './types'

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
  refreshTimeline: (projectId: string, ticketId: string, options?: LoadOptions) => Promise<void>
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
    await drawerState.refreshTimeline(projectId, ticketId, {
      background: true,
      preserveMessages: true,
    })
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
    await drawerState.refreshTimeline(projectId, ticketId, {
      background: true,
      preserveMessages: true,
    })
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
    await drawerState.refreshTimeline(projectId, ticketId, {
      background: true,
      preserveMessages: true,
    })
    return true
  } catch (caughtError) {
    drawerState.setMutationError(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete comment.',
    )
    await drawerState.refreshTimeline(projectId, ticketId, {
      background: true,
      preserveMessages: true,
    })
    return false
  } finally {
    drawerState.deletingCommentId = null
  }
}

export async function loadTicketCommentHistory({
  ticketId,
  commentId,
}: {
  ticketId?: string | null
  commentId: string
}) {
  if (!ticketId) return []

  try {
    const response = await listTicketCommentRevisions(ticketId, commentId)
    return response.revisions
      .map(parseRawTicketCommentRevision)
      .filter((item): item is TicketCommentRevision => item !== null)
  } catch (caughtError) {
    throw new Error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to load comment history.',
    )
  }
}

function parseRawTicketCommentRevision(
  raw: TicketCommentRevisionRecord,
): TicketCommentRevision | null {
  if (
    !raw.id ||
    !raw.comment_id ||
    !raw.edited_at ||
    !raw.edited_by ||
    typeof raw.revision_number !== 'number'
  ) {
    return null
  }

  return {
    id: raw.id,
    commentId: raw.comment_id,
    revisionNumber: raw.revision_number,
    bodyMarkdown: raw.body_markdown ?? '',
    editedBy: raw.edited_by,
    editedAt: raw.edited_at,
    editReason: raw.edit_reason ?? undefined,
  }
}
