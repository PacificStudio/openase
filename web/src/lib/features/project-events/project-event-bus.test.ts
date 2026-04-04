import { afterEach, describe, expect, it, vi } from 'vitest'

import type { SSEFrame } from '$lib/api/sse'
import {
  isProjectDashboardRefreshEvent,
  projectEventAffectsTicketDetailReferences,
  projectEventReferencesTicket,
  readProjectDashboardRefreshSections,
  retainProjectEventBus,
  subscribeProjectEventBusState,
  subscribeProjectEvents,
  toProjectEventFrame,
} from './project-event-bus'

const { connectEventStream } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/sse', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/sse')>('$lib/api/sse')
  return {
    ...actual,
    connectEventStream,
  }
})

describe('projectEventBus', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shares one passive project connection across a representative project composition', () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const onTickets = vi.fn()
    const onDrawer = vi.fn()
    const onUpdates = vi.fn()
    const onState = vi.fn()

    const releaseShell = retainProjectEventBus('project-1', { onStateChange: onState })
    const unsubscribeTickets = subscribeProjectEvents('project-1', onTickets)
    const unsubscribeDrawer = subscribeProjectEvents('project-1', onDrawer)
    const unsubscribeUpdates = subscribeProjectEvents('project-1', onUpdates)

    expect(connectEventStream).toHaveBeenCalledTimes(1)
    expect(connectEventStream).toHaveBeenCalledWith(
      '/api/v1/projects/project-1/events/stream',
      expect.objectContaining({
        onEvent: expect.any(Function),
        onStateChange: expect.any(Function),
      }),
    )

    const options = connectEventStream.mock.calls[0]?.[1] as {
      onEvent: (frame: SSEFrame) => void
      onStateChange: (state: 'live' | 'idle' | 'connecting' | 'retrying') => void
    }
    options.onStateChange('live')
    options.onEvent({
      event: 'ticket.updated',
      data: JSON.stringify({
        topic: 'ticket.events',
        type: 'ticket.updated',
        payload: { ticket: { id: 'ticket-1' } },
        published_at: '2026-04-02T10:00:00Z',
      }),
    })

    expect(onState).toHaveBeenLastCalledWith('live')
    expect(onTickets).toHaveBeenCalledTimes(1)
    expect(onDrawer).toHaveBeenCalledTimes(1)
    expect(onUpdates).toHaveBeenCalledTimes(1)

    unsubscribeUpdates()
    unsubscribeDrawer()
    unsubscribeTickets()
    releaseShell()

    expect(disconnect).toHaveBeenCalledTimes(1)
  })

  it('tears down the old project bus and opens the new project exactly once on project switch', () => {
    const disconnectProjectOne = vi.fn()
    const disconnectProjectTwo = vi.fn()
    connectEventStream
      .mockReturnValueOnce(disconnectProjectOne)
      .mockReturnValueOnce(disconnectProjectTwo)

    const releaseProjectOne = retainProjectEventBus('project-1')
    releaseProjectOne()

    const releaseProjectTwo = retainProjectEventBus('project-2')

    expect(connectEventStream).toHaveBeenCalledTimes(2)
    expect(connectEventStream.mock.calls[0]?.[0]).toBe('/api/v1/projects/project-1/events/stream')
    expect(connectEventStream.mock.calls[1]?.[0]).toBe('/api/v1/projects/project-2/events/stream')
    expect(disconnectProjectOne).toHaveBeenCalledTimes(1)
    expect(disconnectProjectTwo).not.toHaveBeenCalled()

    releaseProjectTwo()
    expect(disconnectProjectTwo).toHaveBeenCalledTimes(1)
  })

  it('exposes typed ticket helpers at the bus boundary', () => {
    expect(
      projectEventReferencesTicket(
        {
          topic: 'ticket.run.events',
          payload: {
            ticket_id: 'ticket-1',
          },
        },
        'ticket-1',
      ),
    ).toBe(true)

    expect(
      projectEventReferencesTicket(
        {
          topic: 'ticket.run.events',
          payload: {
            entry: {
              ticket_id: 'ticket-1',
            },
          },
        },
        'ticket-1',
      ),
    ).toBe(true)

    expect(
      toProjectEventFrame({
        topic: 'ticket.run.events',
        type: 'ticket.run.trace',
        payload: { entry: { ticket_id: 'ticket-1', output: 'Planning' } },
        publishedAt: '2026-04-02T10:00:00Z',
      }),
    ).toEqual({
      event: 'ticket.run.trace',
      data: JSON.stringify({ entry: { ticket_id: 'ticket-1', output: 'Planning' } }),
    })

    expect(
      isProjectDashboardRefreshEvent({
        topic: 'project.dashboard.events',
        type: 'project.dashboard.refresh',
      }),
    ).toBe(true)

    expect(
      readProjectDashboardRefreshSections({
        payload: {
          dirty_sections: ['project', 'agents', 'tickets', 'agents', 'bogus'],
        },
      }),
    ).toEqual(['project', 'agents', 'tickets'])

    expect(
      projectEventAffectsTicketDetailReferences(
        {
          topic: 'ticket.events',
          type: 'ticket.updated',
          payload: {
            ticket: {
              id: 'ticket-2',
            },
          },
        },
        'ticket-1',
      ),
    ).toBe(true)

    expect(
      projectEventAffectsTicketDetailReferences(
        {
          topic: 'activity.events',
          type: 'project_repo_updated',
          payload: {
            event: {
              event_type: 'project_repo_updated',
            },
          },
        },
        'ticket-1',
      ),
    ).toBe(true)
  })

  it('lets non-owning listeners observe connection state without opening another transport', () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const releaseShell = retainProjectEventBus('project-1')
    const stateListener = vi.fn()
    const unsubscribeState = subscribeProjectEventBusState('project-1', stateListener)

    expect(connectEventStream).toHaveBeenCalledTimes(1)

    const options = connectEventStream.mock.calls[0]?.[1] as {
      onStateChange: (state: 'live' | 'idle' | 'connecting' | 'retrying') => void
    }
    options.onStateChange('retrying')

    expect(stateListener).toHaveBeenLastCalledWith('retrying')

    unsubscribeState()
    releaseShell()
  })
})
