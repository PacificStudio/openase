import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketCommentRevision, TicketDetail, TicketTimelineItem } from '../types'
import TicketCommentsThread from './ticket-comments-thread.svelte'

const ticket: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-335',
  title: 'Add comment history',
  description: 'Ticket description',
  status: { id: 'todo', name: 'Todo', color: '#94a3b8' },
  priority: 'medium',
  type: 'feature',
  repoScopes: [],
  attemptCount: 0,
  costAmount: 0,
  budgetUsd: 100,
  dependencies: [],
  externalLinks: [],
  children: [],
  createdBy: 'user:tester',
  createdAt: '2026-03-27T12:00:00Z',
  updatedAt: '2026-03-27T12:00:00Z',
}

const timeline: TicketTimelineItem[] = [
  {
    id: 'description:ticket-1',
    ticketId: 'ticket-1',
    kind: 'description',
    actor: { name: 'tester', type: 'user' },
    title: 'Add comment history',
    bodyMarkdown: 'Ticket description',
    createdAt: '2026-03-27T12:00:00Z',
    updatedAt: '2026-03-27T12:00:00Z',
    isCollapsible: false,
    isDeleted: false,
    identifier: 'ASE-335',
  },
  {
    id: 'comment:comment-1',
    ticketId: 'ticket-1',
    kind: 'comment',
    commentId: 'comment-1',
    actor: { name: 'reviewer', type: 'user' },
    bodyMarkdown: 'Updated comment body',
    createdAt: '2026-03-27T12:05:00Z',
    updatedAt: '2026-03-27T12:10:00Z',
    editedAt: '2026-03-27T12:10:00Z',
    isCollapsible: true,
    isDeleted: false,
    editCount: 1,
    revisionCount: 2,
    lastEditedBy: 'user:reviewer',
  },
  {
    id: 'comment:comment-2',
    ticketId: 'ticket-1',
    kind: 'comment',
    commentId: 'comment-2',
    actor: { name: 'archiver', type: 'user' },
    bodyMarkdown: 'Comment deleted',
    createdAt: '2026-03-27T12:15:00Z',
    updatedAt: '2026-03-27T12:20:00Z',
    editedAt: '2026-03-27T12:20:00Z',
    isCollapsible: true,
    isDeleted: true,
    editCount: 1,
    revisionCount: 2,
    lastEditedBy: 'user:archiver',
  },
]

const revisions: TicketCommentRevision[] = [
  {
    id: 'revision-2',
    commentId: 'comment-1',
    revisionNumber: 2,
    bodyMarkdown: 'Updated comment body',
    editedBy: 'user:reviewer',
    editedAt: '2026-03-27T12:10:00Z',
  },
  {
    id: 'revision-1',
    commentId: 'comment-1',
    revisionNumber: 1,
    bodyMarkdown: 'Original comment body',
    editedBy: 'user:reviewer',
    editedAt: '2026-03-27T12:05:00Z',
  },
]

describe('TicketCommentsThread', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-27T12:20:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('shows edited metadata, opens comment history, and preserves deleted placeholders', async () => {
    const onLoadCommentHistory = vi.fn(async () => revisions)
    const { findByText, getByLabelText, getAllByText } = render(TicketCommentsThread, {
      props: {
        ticket,
        timeline,
        onLoadCommentHistory,
      },
    })

    expect(getAllByText('rev 2')).toHaveLength(2)
    expect(getByLabelText('View history for comment by reviewer')).toBeTruthy()
    expect(await findByText('edited 10m ago')).toBeTruthy()
    expect(await findByText('Comment deleted')).toBeTruthy()

    await fireEvent.click(getByLabelText('View history for comment by reviewer'))

    expect(onLoadCommentHistory).toHaveBeenCalledWith('comment-1')
    expect(await findByText('Revision 2')).toBeTruthy()
    expect(await findByText('Original comment body')).toBeTruthy()
  })

  it('collapses and re-expands a comment without mutating the backend data', async () => {
    const { findByText, getByLabelText, queryByText } = render(TicketCommentsThread, {
      props: {
        ticket,
        timeline,
      },
    })

    expect(await findByText('Updated comment body')).toBeTruthy()

    await fireEvent.click(getByLabelText('Collapse comment by reviewer'))

    expect(await findByText('Comment collapsed.')).toBeTruthy()
    expect(queryByText('Updated comment body')).toBeNull()

    await fireEvent.click(getByLabelText('Expand comment by reviewer'))

    expect(await findByText('Updated comment body')).toBeTruthy()
  })
})
