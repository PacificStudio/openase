import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import {
  buildLiveContext,
  buildReferenceData,
  createDeferred,
  createRunDeps,
} from './drawer-state.test-fixtures'
import { createTicketDrawerState } from './drawer-state.svelte'

describe('createTicketDrawerState loading refresh handling', () => {
  it('replays a timeline refresh requested during the initial loading window', async () => {
    const initialReferenceData = buildReferenceData()
    const initialContext = buildLiveContext()
    const deferredInitialContext = createDeferred<TicketDetailLiveContext>()
    const refreshedContext = buildLiveContext({
      ticket: {
        ...initialContext.ticket,
        title: 'Runtime updated after queued event refresh',
      },
      timeline: [
        ...initialContext.timeline,
        {
          id: 'activity:event-queued',
          ticketId: 'ticket-1',
          kind: 'activity',
          actor: { name: 'runtime', type: 'system' },
          eventType: 'agent.executing',
          title: 'agent.executing',
          bodyText: 'Agent started executing after the drawer subscribed.',
          createdAt: '2026-03-29T10:06:00Z',
          updatedAt: '2026-03-29T10:06:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          metadata: {},
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
      .mockReturnValueOnce(deferredInitialContext.promise)
      .mockResolvedValueOnce(refreshedContext)
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(initialReferenceData)

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    const loadPromise = state.load('project-1', 'ticket-1')
    await Promise.resolve()

    await state.refreshTimeline('project-1', 'ticket-1')
    deferredInitialContext.resolve(initialContext)

    await loadPromise

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
    expect(state.ticket?.title).toBe('Runtime updated after queued event refresh')
    expect(state.timeline).toEqual(refreshedContext.timeline)
  })
})
