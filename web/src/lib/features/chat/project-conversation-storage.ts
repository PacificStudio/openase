const projectConversationStoragePrefix = 'openase.project-conversation'

export type PersistedProjectConversationTabs = {
  conversationIds: string[]
  activeConversationId: string
}

function storageKey(projectId: string, currentProviderId: string) {
  return `${projectConversationStoragePrefix}.${projectId}.${currentProviderId}`
}

export function storeProjectConversationTabs(
  projectId: string,
  currentProviderId: string,
  value: PersistedProjectConversationTabs,
) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(storageKey(projectId, currentProviderId), JSON.stringify(value))
  } catch {
    // Ignore localStorage failures.
  }
}

export function readProjectConversationTabs(
  projectId: string,
  currentProviderId: string,
): PersistedProjectConversationTabs {
  if (typeof window === 'undefined') {
    return { conversationIds: [], activeConversationId: '' }
  }

  try {
    const raw = window.localStorage.getItem(storageKey(projectId, currentProviderId))?.trim() ?? ''
    if (!raw) {
      return { conversationIds: [], activeConversationId: '' }
    }
    const parsed = JSON.parse(raw) as {
      conversationIds?: unknown
      activeConversationId?: unknown
    }
    const conversationIds = Array.isArray(parsed.conversationIds)
      ? parsed.conversationIds
          .map((value) => (typeof value === 'string' ? value.trim() : ''))
          .filter((value) => value.length > 0)
      : []
    const activeConversationId =
      typeof parsed.activeConversationId === 'string' ? parsed.activeConversationId.trim() : ''
    return { conversationIds, activeConversationId }
  } catch {
    return { conversationIds: [], activeConversationId: '' }
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}
