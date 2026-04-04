import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { connectEventStream, consumeEventStream } from './sse'

describe('sse', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('retries when a stream stays idle for 10 seconds', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      body: new ReadableStream<Uint8Array>({ start() {} }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const onError = vi.fn()
    const onStateChange = vi.fn()
    const disconnect = connectEventStream('/api/v1/projects/project-1/events/stream', {
      onEvent: vi.fn(),
      onError,
      onStateChange,
      retryDelayMs: 1000,
    })

    await vi.advanceTimersByTimeAsync(10000)
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1000)

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(onError).toHaveBeenCalledWith(expect.any(Error))
    expect(onStateChange).toHaveBeenCalledWith('retrying')

    disconnect()
  })

  it('treats keepalive comments as stream activity', async () => {
    const encoder = new TextEncoder()
    const frames: string[] = []
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        window.setTimeout(() => {
          controller.enqueue(encoder.encode(': keepalive\n\n'))
        }, 5000)
        window.setTimeout(() => {
          controller.enqueue(
            encoder.encode('event: ticket.updated\ndata: {"ticket":{"id":"ticket-1"}}\n\n'),
          )
        }, 9000)
        window.setTimeout(() => {
          controller.close()
        }, 9500)
      },
    })

    const done = consumeEventStream(
      stream,
      (frame) => {
        frames.push(frame.event)
      },
      { activityTimeoutMs: 10000 },
    )

    await vi.advanceTimersByTimeAsync(9500)
    await expect(done).resolves.toBeUndefined()
    expect(frames).toEqual(['ticket.updated'])
  })

  it('fails consumeEventStream when no bytes arrive before the activity timeout', async () => {
    const stream = new ReadableStream<Uint8Array>({ start() {} })

    const done = consumeEventStream(stream, vi.fn(), { activityTimeoutMs: 10000 })
    const expectation = expect(done).rejects.toThrow('stream went idle for 10000ms')

    await vi.advanceTimersByTimeAsync(10000)
    await expectation
  })
})
