import { afterEach, describe, expect, it, vi } from 'vitest'

const { watchProjectConversationMuxStream } = vi.hoisted(() => ({
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  watchProjectConversationMuxStream,
}))

import { watchProjectConversationMux } from './project-conversation-event-bus'

describe('watchProjectConversationMux', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shares one SSE connection across multiple conversation subscriptions in the same project', async () => {
    let firstCall:
      | {
          onOpen?: () => void
          onFrame: (frame: { conversationId: string; sentAt: string; event: unknown }) => void
        }
      | undefined
    watchProjectConversationMuxStream.mockImplementation((_projectId, handlers) => {
      firstCall = handlers
      return new Promise<void>((resolve, reject) => {
        handlers.signal?.addEventListener(
          'abort',
          () => reject(new DOMException('Aborted', 'AbortError')),
          { once: true },
        )
      })
    })

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

    expect(watchProjectConversationMuxStream).toHaveBeenCalledTimes(1)
    firstCall?.onOpen?.()
    await Promise.all([firstWatch.connected, secondWatch.connected])

    firstCall?.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-04T12:00:00Z',
      event: {
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
    })
    firstCall?.onFrame({
      conversationId: 'conversation-2',
      sentAt: '2026-04-04T12:00:01Z',
      event: {
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
    await firstWatch.stream
    await secondWatch.stream
  })

  it('replays cached session frames to late subscribers and signals reconnect after retries', async () => {
    vi.useFakeTimers()

    const calls: Array<{
      handlers: {
        signal?: AbortSignal
        onOpen?: () => void
        onFrame: (frame: { conversationId: string; sentAt: string; event: unknown }) => void
      }
      resolve: () => void
      reject: (error: unknown) => void
    }> = []
    watchProjectConversationMuxStream.mockImplementation((_projectId, handlers) => {
      return new Promise<void>((resolve, reject) => {
        const call = { handlers, resolve, reject }
        calls.push(call)
        handlers.signal?.addEventListener(
          'abort',
          () => reject(new DOMException('Aborted', 'AbortError')),
          { once: true },
        )
      })
    })

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

    calls[0]?.handlers.onOpen?.()
    calls[0]?.handlers.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-04T12:00:00Z',
      event: {
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
    })

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

    calls[0]?.resolve()
    await Promise.resolve()
    await vi.advanceTimersByTimeAsync(2000)
    expect(watchProjectConversationMuxStream).toHaveBeenCalledTimes(2)

    calls[1]?.handlers.onOpen?.()
    expect(reconnected).toHaveBeenCalledTimes(1)
    await Promise.all([firstWatch.connected, secondWatch.connected])

    firstController.abort()
    secondController.abort()
    await firstWatch.stream
    await secondWatch.stream

    vi.useRealTimers()
  })

  it('retries when the mux stream goes idle after connecting', async () => {
    vi.useFakeTimers()

    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    watchProjectConversationMuxStream.mockImplementation((_projectId, handlers) => {
      handlers.onOpen?.()
      return new Promise<void>((_resolve, reject) => {
        const timeoutId = window.setTimeout(() => {
          reject(new Error('stream went idle for 10000ms'))
        }, 10000)
        handlers.signal?.addEventListener(
          'abort',
          () => {
            window.clearTimeout(timeoutId)
            reject(new DOMException('Aborted', 'AbortError'))
          },
          { once: true },
        )
      })
    })

    const controller = new AbortController()
    const onReconnect = vi.fn()
    const watch = watchProjectConversationMux({
      projectId: 'project-1',
      conversationId: 'conversation-1',
      signal: controller.signal,
      onEvent: vi.fn(),
      onReconnect,
    })

    await watch.connected
    expect(watchProjectConversationMuxStream).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(10000)
    await vi.advanceTimersByTimeAsync(2000)
    expect(watchProjectConversationMuxStream).toHaveBeenCalledTimes(2)
    expect(onReconnect).toHaveBeenCalledTimes(1)
    expect(consoleError).toHaveBeenCalledWith(
      'Project conversation mux bus error:',
      expect.any(Error),
    )

    controller.abort()
    await watch.stream

    consoleError.mockRestore()
    vi.useRealTimers()
  })
})
