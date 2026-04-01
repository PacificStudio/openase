import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailContext } from './context'
import { createTicketDrawerState } from './drawer-state.svelte'

function buildContext(overrides: Partial<TicketDetailContext> = {}): TicketDetailContext {
  return {
    ticket: {
      id: 'ticket-1',
      identifier: 'ASE-336',
      title: 'Align Ticket Detail refresh wiring',
      description: 'Initial description',
      status: {
        id: 'status-1',
        name: 'Todo',
        color: '#2563eb',
      },
      priority: 'high',
      type: 'feature',
      repoScopes: [],
      attemptCount: 0,
      retryPaused: false,
      costTokensInput: 0,
      costTokensOutput: 0,
      costAmount: 0,
      budgetUsd: 0,
      dependencies: [],
      externalLinks: [],
      children: [],
      createdBy: 'codex',
      createdAt: '2026-03-29T10:00:00Z',
      updatedAt: '2026-03-29T10:00:00Z',
    },
    timeline: [
      {
        id: 'description:ticket-1',
        ticketId: 'ticket-1',
        kind: 'description',
        actor: { name: 'codex', type: 'user' },
        title: 'Align Ticket Detail refresh wiring',
        bodyMarkdown: 'Initial description',
        createdAt: '2026-03-29T10:00:00Z',
        updatedAt: '2026-03-29T10:00:00Z',
        editedAt: undefined,
        isCollapsible: false,
        isDeleted: false,
        identifier: 'ASE-336',
      },
    ],
    hooks: [],
    statuses: [
      {
        id: 'status-1',
        name: 'Todo',
        color: '#2563eb',
      },
    ],
    dependencyCandidates: [
      {
        id: 'ticket-2',
        identifier: 'ASE-337',
        title: 'Follow-up',
      },
    ],
    repoOptions: [
      {
        id: 'repo-1',
        name: 'openase',
        defaultBranch: 'main',
      },
    ],
    ...overrides,
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

describe('createTicketDrawerState', () => {
  it('refreshes only the live ticket timeline snapshot', async () => {
    const initialContext = buildContext()
    const refreshedContext = buildContext({
      ticket: {
        ...initialContext.ticket,
        title: 'Align Ticket Detail SSE refresh wiring',
      },
      timeline: [
        ...initialContext.timeline,
        {
          id: 'comment:comment-1',
          ticketId: 'ticket-1',
          kind: 'comment',
          commentId: 'comment-1',
          actor: { name: 'reviewer', type: 'user' },
          bodyMarkdown: 'Looks good.',
          createdAt: '2026-03-29T10:05:00Z',
          updatedAt: '2026-03-29T10:05:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          editCount: 0,
          revisionCount: 1,
          lastEditedBy: undefined,
        },
      ],
      hooks: [
        {
          id: 'hook-1',
          hookName: 'ticket.timeline.refresh',
          status: 'pass',
          timestamp: '2026-03-29T10:05:00Z',
        },
      ],
    })

    const fetchContext = vi
      .fn<(projectId: string, ticketId: string) => Promise<TicketDetailContext>>()
      .mockResolvedValueOnce(initialContext)
      .mockResolvedValueOnce(refreshedContext)

    const state = createTicketDrawerState({ fetchContext })

    await state.load('project-1', 'ticket-1')
    await state.refreshTimeline('project-1', 'ticket-1')

    expect(state.ticket?.title).toBe('Align Ticket Detail SSE refresh wiring')
    expect(state.timeline).toEqual(refreshedContext.timeline)
    expect(state.hooks).toEqual(refreshedContext.hooks)
    expect(state.statuses).toEqual(initialContext.statuses)
    expect(state.dependencyCandidates).toEqual(initialContext.dependencyCandidates)
    expect(state.repoOptions).toEqual(initialContext.repoOptions)
  })

  it('queues one follow-up refresh when another event arrives mid-refresh', async () => {
    const initialContext = buildContext()
    const interimContext = buildContext({
      timeline: [
        ...initialContext.timeline,
        {
          id: 'activity:event-1',
          ticketId: 'ticket-1',
          kind: 'activity',
          actor: { name: 'dispatcher', type: 'system' },
          eventType: 'agent_started',
          title: 'agent_started',
          bodyText: 'Agent started work.',
          createdAt: '2026-03-29T10:06:00Z',
          updatedAt: '2026-03-29T10:06:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          metadata: {},
        },
      ],
    })
    const finalContext = buildContext({
      timeline: [
        ...interimContext.timeline,
        {
          id: 'comment:comment-2',
          ticketId: 'ticket-1',
          kind: 'comment',
          commentId: 'comment-2',
          actor: { name: 'reviewer', type: 'user' },
          bodyMarkdown: 'History count updated.',
          createdAt: '2026-03-29T10:07:00Z',
          updatedAt: '2026-03-29T10:07:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          editCount: 0,
          revisionCount: 1,
          lastEditedBy: undefined,
        },
      ],
    })
    const deferredRefresh = createDeferred<TicketDetailContext>()

    const fetchContext = vi
      .fn<(projectId: string, ticketId: string) => Promise<TicketDetailContext>>()
      .mockResolvedValueOnce(initialContext)
      .mockReturnValueOnce(deferredRefresh.promise)
      .mockResolvedValueOnce(finalContext)

    const state = createTicketDrawerState({ fetchContext })
    await state.load('project-1', 'ticket-1')

    const firstRefresh = state.refreshTimeline('project-1', 'ticket-1')
    const secondRefresh = state.refreshTimeline('project-1', 'ticket-1')

    expect(fetchContext).toHaveBeenCalledTimes(2)

    deferredRefresh.resolve(interimContext)
    await Promise.all([firstRefresh, secondRefresh])

    expect(fetchContext).toHaveBeenCalledTimes(3)
    expect(state.timeline).toEqual(finalContext.timeline)
  })
})
