const globalStorageKey = 'openase.project-conversation.global'
const legacyStoragePrefix = 'openase.project-conversation'

export type PersistedProjectConversationTab = {
  projectId: string
  projectName: string
  conversationId: string
  providerId: string
  draft: string
}

export type PersistedProjectConversationTabs = {
  tabs: PersistedProjectConversationTab[]
  activeTabIndex: number
}

export function storeProjectConversationTabs(value: PersistedProjectConversationTabs) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(globalStorageKey, JSON.stringify(value))
  } catch {
    // Ignore localStorage failures.
  }
}

export function readProjectConversationTabs(): PersistedProjectConversationTabs {
  if (typeof window === 'undefined') {
    return { tabs: [], activeTabIndex: 0 }
  }

  try {
    const raw = window.localStorage.getItem(globalStorageKey)?.trim() ?? ''
    if (raw) {
      return parsePersistedTabs(raw)
    }
    return { tabs: [], activeTabIndex: 0 }
  } catch {
    return { tabs: [], activeTabIndex: 0 }
  }
}

export function migrateLegacyProjectConversationTabs(projectId: string): void {
  if (typeof window === 'undefined' || !projectId) return

  try {
    const legacyKey = `${legacyStoragePrefix}.${projectId}`
    const raw = window.localStorage.getItem(legacyKey)?.trim() ?? ''
    if (!raw) return

    const parsed = JSON.parse(raw) as { tabs?: unknown; activeTabIndex?: unknown }
    const tabs = Array.isArray(parsed.tabs)
      ? parsed.tabs
          .map((value) => {
            if (typeof value !== 'object' || value == null) return null
            const candidate = value as {
              conversationId?: unknown
              providerId?: unknown
              draft?: unknown
            }
            return {
              projectId,
              projectName: '',
              conversationId:
                typeof candidate.conversationId === 'string' ? candidate.conversationId.trim() : '',
              providerId:
                typeof candidate.providerId === 'string' ? candidate.providerId.trim() : '',
              draft: typeof candidate.draft === 'string' ? candidate.draft : '',
            } satisfies PersistedProjectConversationTab
          })
          .filter((value): value is PersistedProjectConversationTab => value != null)
      : []

    if (tabs.length === 0) {
      window.localStorage.removeItem(legacyKey)
      return
    }

    const existing = readProjectConversationTabs()
    const merged: PersistedProjectConversationTabs = {
      tabs: [...existing.tabs, ...tabs],
      activeTabIndex: existing.tabs.length > 0 ? existing.activeTabIndex : 0,
    }
    storeProjectConversationTabs(merged)
    window.localStorage.removeItem(legacyKey)
  } catch {
    // Ignore migration failures.
  }
}

function parsePersistedTabs(raw: string): PersistedProjectConversationTabs {
  const parsed = JSON.parse(raw) as { tabs?: unknown; activeTabIndex?: unknown }
  const tabs = Array.isArray(parsed.tabs)
    ? parsed.tabs
        .map((value) => {
          if (typeof value !== 'object' || value == null) return null
          const candidate = value as {
            projectId?: unknown
            projectName?: unknown
            conversationId?: unknown
            providerId?: unknown
            draft?: unknown
          }
          return {
            projectId: typeof candidate.projectId === 'string' ? candidate.projectId.trim() : '',
            projectName:
              typeof candidate.projectName === 'string' ? candidate.projectName.trim() : '',
            conversationId:
              typeof candidate.conversationId === 'string' ? candidate.conversationId.trim() : '',
            providerId: typeof candidate.providerId === 'string' ? candidate.providerId.trim() : '',
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
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}
