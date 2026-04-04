import { describe, expect, it, vi, afterEach } from 'vitest'
import { describeTurnFailure } from './ephemeral-chat-turn-failure'

describe('describeTurnFailure', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('explains when the browser is offline', () => {
    vi.stubGlobal('navigator', { onLine: false })

    expect(describeTurnFailure(new TypeError('Failed to fetch'))).toBe(
      'The chat request could not continue because this browser is offline. OpenASE never had a stable stream to finish the reply. Reconnect to the network and retry.',
    )
  })

  it('explains when the stream was cut after a partial reply arrived', () => {
    expect(
      describeTurnFailure(new TypeError('network error'), {
        streamStarted: true,
        partialReplyReceived: true,
      }),
    ).toBe(
      'The reply stopped mid-stream because the browser connection closed before OpenASE sent the final completion event. The partial reply was kept above. This usually means the OpenASE server restarted during the turn or the network connection reset. Retry the request.',
    )
  })

  it('explains when the stream never opened', () => {
    expect(describeTurnFailure(new TypeError('Failed to fetch'))).toBe(
      'The browser could not open the chat stream, so no live reply channel was established. This usually means OpenASE was unreachable, restarting, or the network connection reset before streaming began. Retry the request.',
    )
  })

  it('explains when the response body is missing', () => {
    expect(describeTurnFailure(new Error('chat stream response body is unavailable'))).toBe(
      'OpenASE accepted the chat request, but the browser did not receive a readable streaming response body. Retry the request.',
    )
  })
})
