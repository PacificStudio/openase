import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '$lib/api/client'
import {
  handleCreateTicketComment,
  handleDeleteTicketComment,
  handleUpdateTicketComment,
} from './drawer-comment-actions'

const {
  createTicketComment,
  deleteTicketComment,
  listTicketCommentRevisions,
  updateTicketComment,
} = vi.hoisted(() => ({
  createTicketComment: vi.fn(),
  deleteTicketComment: vi.fn(),
  listTicketCommentRevisions: vi.fn(),
  updateTicketComment: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createTicketComment,
  deleteTicketComment,
  listTicketCommentRevisions,
  updateTicketComment,
}))

function createDrawerState() {
  return {
    creatingComment: false,
    updatingCommentId: null as string | null,
    deletingCommentId: null as string | null,
    clearMutationMessages: vi.fn(),
    setMutationError: vi.fn(),
    setMutationNotice: vi.fn(),
    refreshTimeline: vi.fn().mockResolvedValue(undefined),
  }
}

describe('ticket detail comment actions', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('refreshes the unified timeline after creating a comment', async () => {
    createTicketComment.mockResolvedValue({
      comment: {
        id: 'comment-1',
      },
    })

    const drawerState = createDrawerState()
    const result = await handleCreateTicketComment({
      projectId: 'project-1',
      ticketId: 'ticket-1',
      drawerState,
      body: 'Ship it',
    })

    expect(result).toBe(true)
    expect(drawerState.setMutationNotice).toHaveBeenCalledWith('Comment added.')
    expect(drawerState.refreshTimeline).toHaveBeenCalledWith('project-1', 'ticket-1', {
      background: true,
      preserveMessages: true,
    })
  })

  it('refreshes the unified timeline after updating a comment', async () => {
    updateTicketComment.mockResolvedValue({
      comment: {
        id: 'comment-1',
      },
    })

    const drawerState = createDrawerState()
    const result = await handleUpdateTicketComment({
      projectId: 'project-1',
      ticketId: 'ticket-1',
      drawerState,
      commentId: 'comment-1',
      body: 'Edited body',
    })

    expect(result).toBe(true)
    expect(drawerState.setMutationNotice).toHaveBeenCalledWith('Comment updated.')
    expect(drawerState.refreshTimeline).toHaveBeenCalledWith('project-1', 'ticket-1', {
      background: true,
      preserveMessages: true,
    })
  })

  it('re-syncs the timeline after a delete failure so placeholders and history stay current', async () => {
    deleteTicketComment.mockRejectedValue(new ApiError(500, 'delete failed'))

    const drawerState = createDrawerState()
    const result = await handleDeleteTicketComment({
      projectId: 'project-1',
      ticketId: 'ticket-1',
      drawerState,
      commentId: 'comment-1',
    })

    expect(result).toBe(false)
    expect(drawerState.setMutationError).toHaveBeenCalledWith('delete failed')
    expect(drawerState.refreshTimeline).toHaveBeenCalledWith('project-1', 'ticket-1', {
      background: true,
      preserveMessages: true,
    })
  })
})
