import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  subscribeProjectEvents,
  subscribeProjectEventBusState,
  isTicketRunProjectEvent,
  projectEventAffectsTicketDetailReferences,
  projectEventReferencesTicket,
  toProjectEventFrame,
} = vi.hoisted(() => ({
  subscribeProjectEvents: vi.fn(),
  subscribeProjectEventBusState: vi.fn((_: string, listener: (state: string) => void) => {
    listener('live')
    return () => {}
  }),
  isTicketRunProjectEvent: vi.fn(
    (event: { topic?: string }) => event.topic === 'ticket.run.events',
  ),
  projectEventAffectsTicketDetailReferences: vi.fn(() => false),
  projectEventReferencesTicket: vi.fn(
    (
      event: {
        topic?: string
        payload?: {
          agent?: { current_ticket_id?: string }
          run?: { ticket_id?: string }
          entry?: { ticket_id?: string }
          ticket_id?: string
        }
      },
      ticketId: string,
    ) =>
      event.payload?.agent?.current_ticket_id === ticketId ||
      event.payload?.run?.ticket_id === ticketId ||
      event.payload?.entry?.ticket_id === ticketId ||
      event.payload?.ticket_id === ticketId,
  ),
  toProjectEventFrame: vi.fn((event: { type: string; payload: unknown }) => ({
    event: event.type,
    data: JSON.stringify(event.payload),
  })),
}))

vi.mock('$lib/features/project-events', () => ({
  subscribeProjectEvents,
  subscribeProjectEventBusState,
  isTicketRunProjectEvent,
  projectEventAffectsTicketDetailReferences,
  projectEventReferencesTicket,
  toProjectEventFrame,
}))

import { connectTicketDetailStreams } from './streams'

describe('connectTicketDetailStreams', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it.each(['agent.ready', 'agent.executing'])(
    'refreshes the drawer when %s references the active ticket',
    (eventType) => {
      let projectListener:
        | ((event: {
            topic: string
            type: string
            payload: { agent: { current_ticket_id: string } }
          }) => void)
        | undefined

      subscribeProjectEvents.mockImplementation((_projectId, listener) => {
        projectListener = listener
        return () => {}
      })

      const onRelevantEvent = vi.fn()
      const onRunFrame = vi.fn()

      connectTicketDetailStreams('project-1', 'ticket-1', {
        onRelevantEvent,
        onRunFrame,
      })

      projectListener?.({
        topic: 'agent.events',
        type: eventType,
        payload: {
          agent: {
            current_ticket_id: 'ticket-1',
          },
        },
      })

      expect(onRelevantEvent).toHaveBeenCalledTimes(1)
      expect(onRunFrame).not.toHaveBeenCalled()
    },
  )

  it('passes shared reconnect recovery notifications through to the drawer handlers', () => {
    let reconnectRecovery: ((recovery: { sequence: number }) => void) | undefined

    subscribeProjectEvents.mockImplementation((_projectId, _listener, options) => {
      reconnectRecovery = options?.onReconnectRecovery
      return () => {}
    })

    const onReconnectRecovery = vi.fn()

    connectTicketDetailStreams('project-1', 'ticket-1', {
      onRelevantEvent: vi.fn(),
      onRunFrame: vi.fn(),
      onReconnectRecovery,
    })

    reconnectRecovery?.({ sequence: 2 })

    expect(onReconnectRecovery).toHaveBeenCalledWith({ sequence: 2 })
  })
})
