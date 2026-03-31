const projectConversationStoragePrefix = 'openase.project-conversation'

function storageKey(projectId: string, currentProviderId: string) {
  return `${projectConversationStoragePrefix}.${projectId}.${currentProviderId}`
}

export function storeProjectConversationId(
  projectId: string,
  currentProviderId: string,
  conversationId: string,
) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(storageKey(projectId, currentProviderId), conversationId)
  } catch {
    // Ignore localStorage failures.
  }
}

export function readProjectConversationId(projectId: string, currentProviderId: string) {
  if (typeof window === 'undefined') {
    return ''
  }

  try {
    return window.localStorage.getItem(storageKey(projectId, currentProviderId))?.trim() ?? ''
  } catch {
    return ''
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}
