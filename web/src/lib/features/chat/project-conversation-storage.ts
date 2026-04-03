const projectConversationStoragePrefix = 'openase.project-conversation'

export type PersistedProjectConversationTab = {
  conversationId: string
  providerId: string
  draft: string
}

export type PersistedProjectConversationTabs = {
  tabs: PersistedProjectConversationTab[]
  activeTabIndex: number
}

function storageKey(projectId: string) {
  return `${projectConversationStoragePrefix}.${projectId}`
}

export function storeProjectConversationTabs(
  projectId: string,
  value: PersistedProjectConversationTabs,
) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(storageKey(projectId), JSON.stringify(value))
  } catch {
    // Ignore localStorage failures.
  }
}

export function readProjectConversationTabs(projectId: string): PersistedProjectConversationTabs {
  if (typeof window === 'undefined') {
    return { tabs: [], activeTabIndex: 0 }
  }

  try {
    const raw = window.localStorage.getItem(storageKey(projectId))?.trim() ?? ''
    if (!raw) {
      return { tabs: [], activeTabIndex: 0 }
    }
    const parsed = JSON.parse(raw) as {
      tabs?: unknown
      activeTabIndex?: unknown
    }
    const tabs = Array.isArray(parsed.tabs)
      ? parsed.tabs
          .map((value) => {
            if (typeof value !== 'object' || value == null) {
              return null
            }
            const candidate = value as {
              conversationId?: unknown
              providerId?: unknown
              draft?: unknown
            }
            return {
              conversationId:
                typeof candidate.conversationId === 'string' ? candidate.conversationId.trim() : '',
              providerId:
                typeof candidate.providerId === 'string' ? candidate.providerId.trim() : '',
              draft: typeof candidate.draft === 'string' ? candidate.draft : '',
            } satisfies PersistedProjectConversationTab
          })
          .filter((value): value is PersistedProjectConversationTab => value != null)
      : []
    const activeTabIndex =
      typeof parsed.activeTabIndex === 'number' && Number.isInteger(parsed.activeTabIndex)
        ? Math.max(0, parsed.activeTabIndex)
        : 0
    return { tabs, activeTabIndex }
  } catch {
    return { tabs: [], activeTabIndex: 0 }
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}
