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
  archived: false,
  repoScopes: [],
  attemptCount: 0,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
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

const attemptTimeline: TicketTimelineItem[] = [
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
    id: 'activity:attempt-1-claimed',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.claimed',
    title: 'agent.claimed',
    bodyText: 'Agent claimed the ticket.',
    createdAt: '2026-03-27T12:01:00Z',
    updatedAt: '2026-03-27T12:01:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-1' },
  },
  {
    id: 'activity:attempt-1-launching',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.launching',
    title: 'agent.launching',
    bodyText: 'Agent is launching.',
    createdAt: '2026-03-27T12:02:00Z',
    updatedAt: '2026-03-27T12:02:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-1' },
  },
  {
    id: 'activity:attempt-1-failed',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.failed',
    title: 'agent.failed',
    bodyText: 'Agent failed to launch.',
    createdAt: '2026-03-27T12:03:00Z',
    updatedAt: '2026-03-27T12:03:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-1' },
  },
  {
    id: 'activity:attempt-2-claimed',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.claimed',
    title: 'agent.claimed',
    bodyText: 'Agent claimed the ticket again.',
    createdAt: '2026-03-27T12:10:00Z',
    updatedAt: '2026-03-27T12:10:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-2' },
  },
  {
    id: 'activity:attempt-2-launching',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.launching',
    title: 'agent.launching',
    bodyText: 'Agent is launching again.',
    createdAt: '2026-03-27T12:11:00Z',
    updatedAt: '2026-03-27T12:11:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-2' },
  },
  {
    id: 'activity:attempt-2-ready',
    ticketId: 'ticket-1',
    kind: 'activity',
    actor: { name: 'workflow-seed', type: 'agent' },
    eventType: 'agent.ready',
    title: 'agent.ready',
    bodyText: 'Agent is ready.',
    createdAt: '2026-03-27T12:12:00Z',
    updatedAt: '2026-03-27T12:12:00Z',
    isCollapsible: true,
    isDeleted: false,
    metadata: { run_id: 'run-2' },
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
    const { findByText, getByLabelText } = render(TicketCommentsThread, {
      props: {
        ticket,
        timeline,
        onLoadCommentHistory,
      },
    })

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

    expect(await findByText('Collapsed.')).toBeTruthy()
    expect(queryByText('Updated comment body')).toBeNull()

    await fireEvent.click(getByLabelText('Expand comment by reviewer'))

    expect(await findByText('Updated comment body')).toBeTruthy()
  })

  it('collapses consecutive activity events into a group summary', async () => {
    const { findByText, queryByText, getByText } = render(TicketCommentsThread, {
      props: {
        ticket,
        timeline: attemptTimeline,
      },
    })

    // Group summary visible, individual events hidden
    expect(await findByText(/agent events/)).toBeTruthy()
    expect(queryByText('Agent claimed the ticket.')).toBeNull()
    expect(queryByText('Agent is ready.')).toBeNull()

    // Expand the group to reveal individual events
    await fireEvent.click(getByText(/agent events/))
    expect(await findByText('Agent claimed the ticket.')).toBeTruthy()
    expect(await findByText('Agent is ready.')).toBeTruthy()
  })
})
