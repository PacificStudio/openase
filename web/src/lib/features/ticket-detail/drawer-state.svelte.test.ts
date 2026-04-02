import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import {
  buildLiveContext,
  buildReferenceData,
  createDeferred,
  createRunDeps,
} from './drawer-state.test-fixtures'
import { createTicketDrawerState } from './drawer-state.svelte'

describe('createTicketDrawerState', () => {
  it('refreshes only the live ticket timeline snapshot', async () => {
    const initialReferenceData = buildReferenceData()
    const initialContext = buildLiveContext()
    const refreshedContext = buildLiveContext({
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

    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValueOnce(initialContext)
      .mockResolvedValueOnce(refreshedContext)
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(initialReferenceData)

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    await state.refreshTimeline('project-1', 'ticket-1')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
    expect(state.ticket?.title).toBe('Align Ticket Detail SSE refresh wiring')
    expect(state.timeline).toEqual(refreshedContext.timeline)
    expect(state.hooks).toEqual(refreshedContext.hooks)
    expect(state.statuses).toEqual(initialReferenceData.statuses)
    expect(state.dependencyCandidates).toEqual(initialReferenceData.dependencyCandidatesByTicketId)
    expect(state.repoOptions).toEqual(initialReferenceData.repoOptions)
  })

  it('queues one follow-up refresh when another event arrives mid-refresh', async () => {
    const initialReferenceData = buildReferenceData()
    const initialContext = buildLiveContext()
    const interimContext = buildLiveContext({
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
    const finalContext = buildLiveContext({
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
    const deferredRefresh = createDeferred<TicketDetailLiveContext>()

    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValueOnce(initialContext)
      .mockReturnValueOnce(deferredRefresh.promise)
      .mockResolvedValueOnce(finalContext)
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(initialReferenceData)

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })
    await state.load('project-1', 'ticket-1')

    const firstRefresh = state.refreshTimeline('project-1', 'ticket-1')
    const secondRefresh = state.refreshTimeline('project-1', 'ticket-1')

    await Promise.resolve()

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)

    deferredRefresh.resolve(interimContext)
    await Promise.all([firstRefresh, secondRefresh])

    expect(fetchLiveContext).toHaveBeenCalledTimes(3)
    expect(state.timeline).toEqual(finalContext.timeline)
  })

  it('reuses cached project references when opening another ticket in the same project', async () => {
    const referenceData = buildReferenceData({
      dependencyCandidatesByTicketId: [
        { id: 'ticket-1', identifier: 'ASE-336', title: 'Align Ticket Detail refresh wiring' },
        { id: 'ticket-2', identifier: 'ASE-337', title: 'Follow-up' },
        { id: 'ticket-3', identifier: 'ASE-338', title: 'Another ticket' },
      ],
    })
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(referenceData)
    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValueOnce(buildLiveContext())
      .mockResolvedValueOnce(
        buildLiveContext({
          ticket: {
            ...buildLiveContext().ticket,
            id: 'ticket-2',
            identifier: 'ASE-337',
            title: 'Follow-up',
          },
        }),
      )

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    await state.load('project-1', 'ticket-2')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
    expect(state.ticket?.id).toBe('ticket-2')
    expect(state.dependencyCandidates).toEqual([
      { id: 'ticket-1', identifier: 'ASE-336', title: 'Align Ticket Detail refresh wiring' },
      { id: 'ticket-3', identifier: 'ASE-338', title: 'Another ticket' },
    ])
  })

  it('keeps cached project references across drawer reset for the same project', async () => {
    const referenceData = buildReferenceData()
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(referenceData)
    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValue(buildLiveContext())

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    state.reset()
    await state.load('project-1', 'ticket-1')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
  })
})
