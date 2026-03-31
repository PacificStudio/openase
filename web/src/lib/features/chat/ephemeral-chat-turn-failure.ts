import { ApiError } from '$lib/api/client'

export function describeTurnFailure(error: unknown) {
  if (error instanceof ApiError) {
    return error.detail
  }
  if (isTransientStreamFailure(error)) {
    return 'The chat stream was interrupted before the reply completed. This can happen during a redeploy, page reload, or network reset. Retry the request.'
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
    message.includes('failed to fetch')
  )
}
