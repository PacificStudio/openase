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

  it('authoritatively reloads timeline and references after a reconnect without clearing loaded runs', async () => {
    const initialReferenceData = buildReferenceData()
    const refreshedReferenceData = buildReferenceData({
      statuses: [
        { id: 'status-1', name: 'Todo', color: '#2563eb' },
        { id: 'status-2', name: 'In Progress', color: '#16a34a' },
      ],
      statusLookup: [
        { id: 'status-1', stage: 'unstarted', color: '#2563eb' },
        { id: 'status-2', stage: 'started', color: '#16a34a' },
      ],
      repoOptions: [
        { id: 'repo-1', name: 'openase', defaultBranch: 'main' },
        { id: 'repo-2', name: 'automation', defaultBranch: 'main' },
      ],
    })
    const initialContext = buildLiveContext()
    const refreshedContext = buildLiveContext({
      ticket: {
        ...initialContext.ticket,
        title: 'Recovered after reconnect',
        status: { id: 'status-2', name: 'In Progress', color: '#16a34a' },
      },
      timeline: [
        ...initialContext.timeline,
        {
          id: 'activity:event-recovered',
          ticketId: 'ticket-1',
          kind: 'activity',
          actor: { name: 'runtime', type: 'system' },
          eventType: 'ticket.updated',
          title: 'ticket.updated',
          bodyText: 'Recovered authoritative ticket detail after reconnect.',
          createdAt: '2026-03-29T10:06:00Z',
          updatedAt: '2026-03-29T10:06:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          metadata: {},
        },
      ],
    })

    const runDeps = createRunDeps()
    const state = createTicketDrawerState({
      fetchLiveContext: vi
        .fn<
          (
            projectId: string,
            ticketId: string,
            refs: TicketDetailProjectReferenceData,
          ) => Promise<TicketDetailLiveContext>
        >()
        .mockResolvedValueOnce(initialContext)
        .mockResolvedValueOnce(refreshedContext),
      fetchReferenceData: vi
        .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
        .mockResolvedValueOnce(initialReferenceData)
        .mockResolvedValueOnce(refreshedReferenceData),
      ...runDeps,
    })

    await state.load('project-1', 'ticket-1')
    await state.ensureRunsLoaded('project-1', 'ticket-1')

    const initialRuns = state.runs
    const initialRunBlocks = state.runBlocks

    state.invalidateReferences('project-1')
    await state.load('project-1', 'ticket-1', { background: true, preserveMessages: true })

    expect(state.ticket?.title).toBe('Recovered after reconnect')
    expect(state.ticket?.status.name).toBe('In Progress')
    expect(state.timeline).toEqual(refreshedContext.timeline)
    expect(state.statuses).toEqual(refreshedReferenceData.statuses)
    expect(state.repoOptions).toEqual(refreshedReferenceData.repoOptions)
    expect(state.runs).toBe(initialRuns)
    expect(state.runBlocks).toBe(initialRunBlocks)
    expect(runDeps.fetchRuns).toHaveBeenCalledTimes(1)
  })
})
