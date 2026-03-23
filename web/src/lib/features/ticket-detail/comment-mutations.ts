import { ApiError } from '$lib/api/client'
import { createTicketComment, deleteTicketComment, updateTicketComment } from '$lib/api/openase'

type CommentMutationOptions = {
  mutate: () => Promise<unknown>
  reload: () => Promise<void>
  clearMessages: () => void
  setError: (message: string) => void
  setNotice: (message: string) => void
  successMessage: string
  fallbackError: string
  start?: () => void
  finish?: () => void
}

export async function runTicketCommentMutation({
  mutate,
  reload,
  clearMessages,
  setError,
  setNotice,
  successMessage,
  fallbackError,
  start,
  finish,
}: CommentMutationOptions) {
  start?.()
  clearMessages()

  try {
    await mutate()
    setNotice(successMessage)
    await reload()
  } catch (caughtError) {
    setError(caughtError instanceof ApiError ? caughtError.detail : fallbackError)
  } finally {
    finish?.()
  }
}

type CommentHandlerBindings = {
  getProjectId: () => string | null | undefined
  getTicketId: () => string | null | undefined
  reload: (projectId: string, ticketId: string) => Promise<void>
  clearMessages: () => void
  setError: (message: string) => void
  setNotice: (message: string) => void
  setCreatingComment: (value: boolean) => void
  setUpdatingCommentId: (value: string | null) => void
  setDeletingCommentId: (value: string | null) => void
}

export function createTicketCommentHandlers(bindings: CommentHandlerBindings) {
  return {
    async create(draft: { body: string }) {
      const projectId = bindings.getProjectId()
      const ticketId = bindings.getTicketId()
      if (!projectId || !ticketId) return

      await runTicketCommentMutation({
        start: () => {
          bindings.setCreatingComment(true)
        },
        finish: () => {
          bindings.setCreatingComment(false)
        },
        mutate: () => createTicketComment(ticketId, { body: draft.body }),
        reload: () => bindings.reload(projectId, ticketId),
        clearMessages: bindings.clearMessages,
        setError: bindings.setError,
        setNotice: bindings.setNotice,
        successMessage: 'Comment added.',
        fallbackError: 'Failed to add comment.',
      })
    },

    async update(commentId: string, draft: { body: string }) {
      const projectId = bindings.getProjectId()
      const ticketId = bindings.getTicketId()
      if (!projectId || !ticketId) return

      await runTicketCommentMutation({
        start: () => {
          bindings.setUpdatingCommentId(commentId)
        },
        finish: () => {
          bindings.setUpdatingCommentId(null)
        },
        mutate: () => updateTicketComment(ticketId, commentId, { body: draft.body }),
        reload: () => bindings.reload(projectId, ticketId),
        clearMessages: bindings.clearMessages,
        setError: bindings.setError,
        setNotice: bindings.setNotice,
        successMessage: 'Comment updated.',
        fallbackError: 'Failed to update comment.',
      })
    },

    async delete(commentId: string) {
      const projectId = bindings.getProjectId()
      const ticketId = bindings.getTicketId()
      if (!projectId || !ticketId) return

      await runTicketCommentMutation({
        start: () => {
          bindings.setDeletingCommentId(commentId)
        },
        finish: () => {
          bindings.setDeletingCommentId(null)
        },
        mutate: () => deleteTicketComment(ticketId, commentId),
        reload: () => bindings.reload(projectId, ticketId),
        clearMessages: bindings.clearMessages,
        setError: bindings.setError,
        setNotice: bindings.setNotice,
        successMessage: 'Comment deleted.',
        fallbackError: 'Failed to delete comment.',
      })
    },
  }
}
