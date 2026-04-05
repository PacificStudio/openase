import { listProjectConversations, type ProjectConversation } from '$lib/api/chat'
import type { PersistedProjectConversationTab } from './project-conversation-storage'
import type { ProjectConversationTabState } from './project-conversation-controller-state'

export type FetchedConversationsResult = {
  allConversations: ProjectConversation[]
  currentProjectConversationIds: Set<string>
}

export async function fetchCrossProjectConversations(
  currentProjectId: string,
  persistedTabs: PersistedProjectConversationTab[],
): Promise<FetchedConversationsResult> {
  const projectIdsToFetch = new Set<string>([currentProjectId])
  for (const tab of persistedTabs) {
    if (tab.projectId) projectIdsToFetch.add(tab.projectId)
  }

  const allConversations: ProjectConversation[] = []
  const currentProjectConversationIds = new Set<string>()
  const fetchResults = await Promise.all(
    [...projectIdsToFetch].map(async (pid) => {
      try {
        const payload = await listProjectConversations({ projectId: pid })
        return { pid, conversations: payload.conversations }
      } catch {
        return { pid, conversations: [] as ProjectConversation[] }
      }
    }),
  )
  for (const result of fetchResults) {
    for (const conversation of result.conversations) {
      allConversations.push(conversation)
      if (result.pid === currentProjectId) {
        currentProjectConversationIds.add(conversation.id)
      }
    }
  }

  return { allConversations, currentProjectConversationIds }
}

export type RestoredTabItem = {
  tab: ProjectConversationTabState
  conversationId: string
  restored: boolean
}

export function mapPersistedToRestoredTabs(
  persistedTabs: PersistedProjectConversationTab[],
  conversationsByID: Map<string, ProjectConversation>,
  newTabState: (
    providerId?: string,
    restored?: boolean,
    projectId?: string,
    projectName?: string,
  ) => ProjectConversationTabState,
  fallbackProviderId: string,
): RestoredTabItem[] {
  return persistedTabs
    .map((persistedTab) => {
      const conversation = persistedTab.conversationId
        ? conversationsByID.get(persistedTab.conversationId)
        : null
      if (conversation != null) {
        const tab = newTabState(
          conversation.providerId,
          true,
          conversation.projectId || persistedTab.projectId,
          persistedTab.projectName,
        )
        tab.draft = persistedTab.draft
        return { tab, conversationId: conversation.id, restored: true }
      }
      if (persistedTab.conversationId.trim()) {
        return null
      }
      const tab = newTabState(
        persistedTab.providerId || fallbackProviderId,
        false,
        persistedTab.projectId,
        persistedTab.projectName,
      )
      tab.draft = persistedTab.draft
      return { tab, conversationId: '', restored: false }
    })
    .filter((item): item is RestoredTabItem => item != null)
}
