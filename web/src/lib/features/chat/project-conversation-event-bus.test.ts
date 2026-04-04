import { afterEach, describe, expect, it, vi } from 'vitest'

const { connectEventStream } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

import { watchProjectConversationMux } from './project-conversation-event-bus'

describe('watchProjectConversationMux', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shares one SSE connection across multiple conversation subscriptions in the same project', async () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const firstController = new AbortController()
    const secondController = new AbortController()
    const firstEvents: unknown[] = []
    const secondEvents: unknown[] = []

    const firstWatch = watchProjectConversationMux({
      projectId: 'project-1',
      conversationId: 'conversation-1',
      signal: firstController.signal,
      onEvent: (event) => firstEvents.push(event),
    })
    const secondWatch = watchProjectConversationMux({
      projectId: 'project-1',
      conversationId: 'conversation-2',
      signal: secondController.signal,
      onEvent: (event) => secondEvents.push(event),
    })

    expect(connectEventStream).toHaveBeenCalledTimes(1)
    const streamOptions = connectEventStream.mock.calls[0]?.[1]

    streamOptions?.onEvent({
      event: 'session',
      data: JSON.stringify({
        conversation_id: 'conversation-1',
        sent_at: '2026-04-04T12:00:00Z',
        payload: {
          conversation_id: 'conversation-1',
          runtime_state: 'ready',
        },
      }),
    })
    streamOptions?.onEvent({
      event: 'session',
      data: JSON.stringify({
        conversation_id: 'conversation-2',
        sent_at: '2026-04-04T12:00:01Z',
        payload: {
          conversation_id: 'conversation-2',
          runtime_state: 'executing',
        },
      }),
    })

    expect(firstEvents).toEqual([
      {
        kind: 'session',
        payload: {
          conversationId: 'conversation-1',
          runtimeState: 'ready',
          providerAnchorKind: undefined,
          providerAnchorId: undefined,
          providerTurnId: undefined,
          providerTurnSupported: undefined,
          providerStatus: undefined,
          providerActiveFlags: [],
        },
      },
    ])
    expect(secondEvents).toEqual([
      {
        kind: 'session',
        payload: {
          conversationId: 'conversation-2',
          runtimeState: 'executing',
          providerAnchorKind: undefined,
          providerAnchorId: undefined,
          providerTurnId: undefined,
          providerTurnSupported: undefined,
          providerStatus: undefined,
          providerActiveFlags: [],
        },
      },
    ])

    firstController.abort()
    secondController.abort()
    await firstWatch
    await secondWatch

    expect(disconnect).toHaveBeenCalledTimes(1)
  })

  it('replays cached session frames to late subscribers and signals reconnect after retries', async () => {
    const disconnect = vi.fn()
    connectEventStream.mockReturnValue(disconnect)

    const firstController = new AbortController()
    const reconnected = vi.fn()
    const seenByLateSubscriber: unknown[] = []

    const firstWatch = watchProjectConversationMux({
      projectId: 'project-1',
      conversationId: 'conversation-1',
      signal: firstController.signal,
      onEvent: vi.fn(),
      onReconnect: reconnected,
    })

    const streamOptions = connectEventStream.mock.calls[0]?.[1]
    streamOptions?.onStateChange('connecting')
    streamOptions?.onEvent({
      event: 'session',
      data: JSON.stringify({
        conversation_id: 'conversation-1',
        sent_at: '2026-04-04T12:00:00Z',
        payload: {
          conversation_id: 'conversation-1',
          runtime_state: 'ready',
        },
      }),
    })
    streamOptions?.onStateChange('live')

    const secondController = new AbortController()
    const secondWatch = watchProjectConversationMux({
      projectId: 'project-1',
      conversationId: 'conversation-1',
      signal: secondController.signal,
      onEvent: (event) => seenByLateSubscriber.push(event),
    })

    expect(seenByLateSubscriber).toEqual([
      {
        kind: 'session',
        payload: {
          conversationId: 'conversation-1',
          runtimeState: 'ready',
          providerAnchorKind: undefined,
          providerAnchorId: undefined,
          providerTurnId: undefined,
          providerTurnSupported: undefined,
          providerStatus: undefined,
          providerActiveFlags: [],
        },
      },
    ])

    streamOptions?.onStateChange('retrying')
    streamOptions?.onStateChange('live')
    expect(reconnected).toHaveBeenCalledTimes(1)

    firstController.abort()
    secondController.abort()
    await firstWatch
    await secondWatch
  })
})
