import { ApiError } from '$lib/api/client'

export type TurnFailureContext = {
  streamStarted?: boolean
  partialReplyReceived?: boolean
}

export function describeTurnFailure(error: unknown, context: TurnFailureContext = {}) {
  if (error instanceof ApiError) {
    return error.detail
  }

  if (isOffline()) {
    return 'The chat request could not continue because this browser is offline. OpenASE never had a stable stream to finish the reply. Reconnect to the network and retry.'
  }

  if (isMissingResponseBodyFailure(error)) {
    return 'OpenASE accepted the chat request, but the browser did not receive a readable streaming response body. Retry the request.'
  }

  if (isTransientStreamFailure(error) && context.streamStarted) {
    return context.partialReplyReceived
      ? 'The reply stopped mid-stream because the browser connection closed before OpenASE sent the final completion event. The partial reply was kept above. This usually means the OpenASE server restarted during the turn or the network connection reset. Retry the request.'
      : 'The chat stream opened, but the connection closed before OpenASE sent the final completion event. This usually means the OpenASE server restarted during the turn or the network connection reset. Retry the request.'
  }

  if (isTransientStreamFailure(error)) {
    return 'The browser could not open the chat stream, so no live reply channel was established. This usually means OpenASE was unreachable, restarting, or the network connection reset before streaming began. Retry the request.'
  }
  return 'Ephemeral chat request failed.'
}

function isTransientStreamFailure(error: unknown) {
  if (error instanceof TypeError) {
    return true
  }
  if (!(error instanceof Error)) {
    return false
  }

  const message = error.message.toLowerCase()
  return (
    message.includes('network error') ||
    message.includes('incomplete chunked') ||
    message.includes('failed to fetch') ||
    message.includes('fetch failed') ||
    message.includes('load failed') ||
    message.includes('network connection was lost') ||
    message.includes('terminated')
  )
}

function isMissingResponseBodyFailure(error: unknown) {
  return (
    error instanceof Error &&
    error.message.toLowerCase() === 'chat stream response body is unavailable'
  )
}

function isOffline() {
  return typeof navigator !== 'undefined' && navigator.onLine === false
}
