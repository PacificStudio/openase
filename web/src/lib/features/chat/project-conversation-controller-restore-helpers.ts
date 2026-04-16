import { listProjectConversations, type ProjectConversation } from '$lib/api/chat'
import {
  disposeProjectConversationTabs,
  findProjectConversationTab,
  type CreateProjectConversationControllerInput,
  type ProjectConversationTabState,
} from './project-conversation-controller-state'
import type { PersistedProjectConversationTab } from './project-conversation-storage'
import {
  migrateLegacyProjectConversationTabs,
  readProjectConversationTabs,
} from './project-conversation-storage'

export type FetchedConversationsResult = {
  allConversations: ProjectConversation[]
  currentProjectConversationIds: Set<string>
  failedProjectIds: Set<string>
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
  const failedProjectIds = new Set<string>()
  const fetchResults = await Promise.all(
    [...projectIdsToFetch].map(async (pid) => {
      try {
        const payload = await listProjectConversations({ projectId: pid })
        return { pid, conversations: payload.conversations, ok: true }
      } catch {
        return { pid, conversations: [] as ProjectConversation[], ok: false }
      }
    }),
  )
  for (const result of fetchResults) {
    if (!result.ok) {
      failedProjectIds.add(result.pid)
    }
    for (const conversation of result.conversations) {
      allConversations.push(conversation)
      if (result.pid === currentProjectId) {
        currentProjectConversationIds.add(conversation.id)
      }
    }
  }

  return { allConversations, currentProjectConversationIds, failedProjectIds }
}

export type RestoredTabItem = {
  tab: ProjectConversationTabState
  conversationId: string
  restored: boolean
}

export function mapPersistedToRestoredTabs(
  persistedTabs: PersistedProjectConversationTab[],
  conversationsByID: Map<string, ProjectConversation>,
  failedProjectIds: Set<string>,
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
      if (
        persistedTab.conversationId.trim() &&
        persistedTab.projectId &&
        failedProjectIds.has(persistedTab.projectId)
      ) {
        const tab = newTabState(
          persistedTab.providerId || fallbackProviderId,
          true,
          persistedTab.projectId,
          persistedTab.projectName,
        )
        tab.conversationId = persistedTab.conversationId.trim()
        tab.needsHydration = true
        tab.draft = persistedTab.draft
        return { tab, conversationId: tab.conversationId, restored: true }
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

type RestoreControllerRuntime = {
  sortProjectConversations: (items: ProjectConversation[]) => ProjectConversation[]
  loadTabConversation: (
    tab: ProjectConversationTabState,
    conversationId: string,
    restored: boolean,
  ) => Promise<boolean>
  restoreTabConversationMetadata: (
    tab: ProjectConversationTabState,
    conversation: ProjectConversation,
    restored: boolean,
  ) => void
}

type RestoreProjectConversationControllerInput = {
  controllerInput: CreateProjectConversationControllerInput
  runtime: RestoreControllerRuntime
  getRestoreOperationID: () => number
  getPreferredProviderId: () => string
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  setConversations: (value: ProjectConversation[]) => void
  getConversations: () => ProjectConversation[]
  setActiveTabId: (value: string) => void
  newTabState: (
    providerId?: string,
    restored?: boolean,
    projectId?: string,
    projectName?: string,
  ) => ProjectConversationTabState
  ensureTabExists: () => void
  persistTabs: () => void
}

export async function restoreProjectConversationController(
  input: RestoreProjectConversationControllerInput,
): Promise<void> {
  const { projectId } = input.controllerInput.getProjectContext()
  if (!projectId) {
    input.ensureTabExists()
    return
  }

  const currentRestoreID = input.getRestoreOperationID()
  disposeProjectConversationTabs(input.getTabs())

  try {
    migrateLegacyProjectConversationTabs(projectId)
    const persisted = readProjectConversationTabs()

    const { allConversations, currentProjectConversationIds, failedProjectIds } =
      await fetchCrossProjectConversations(projectId, persisted.tabs)
    if (currentRestoreID !== input.getRestoreOperationID()) return

    input.setConversations(input.runtime.sortProjectConversations(allConversations))
    if (hasLocalTabActivity(input.getTabs())) {
      input.persistTabs()
      return
    }

    const conversationsByID = new Map(
      allConversations.map((conversation) => [conversation.id, conversation]),
    )
    const restoredTabs = mapPersistedToRestoredTabs(
      persisted.tabs,
      conversationsByID,
      failedProjectIds,
      input.newTabState,
      input.getPreferredProviderId(),
    )

    if (restoredTabs.length === 0) {
      const currentProjectConversations = allConversations.filter((conversation) =>
        currentProjectConversationIds.has(conversation.id),
      )
      const latestConversation =
        input.runtime.sortProjectConversations(currentProjectConversations)[0] ?? null
      if (latestConversation) {
        const restoredTab = input.newTabState(
          latestConversation.providerId,
          true,
          latestConversation.projectId,
          '',
        )
        input.setTabs([restoredTab])
        input.setActiveTabId(restoredTab.id)

        const liveRestoredTab = findProjectConversationTab(input.getTabs(), restoredTab.id)
        if (
          liveRestoredTab &&
          (await input.runtime.loadTabConversation(liveRestoredTab, latestConversation.id, true))
        ) {
          input.persistTabs()
          return
        }
      }

      const blankTab = input.newTabState(input.getPreferredProviderId(), false)
      input.setTabs([blankTab])
      input.setActiveTabId(blankTab.id)
      input.persistTabs()
      return
    }

    const nextTabs = restoredTabs.map((item) => item.tab)
    input.setTabs(nextTabs)
    const preferredTab =
      nextTabs[Math.min(persisted.activeTabIndex, nextTabs.length - 1)] ?? nextTabs[0] ?? null
    const preferredActiveTabId = preferredTab?.id ?? ''
    input.setActiveTabId(preferredActiveTabId)

    const loadedTabIDs = new Set(
      restoredTabs.filter((item) => item.conversationId === '').map((item) => item.tab.id),
    )
    for (let index = 0; index < restoredTabs.length; index += 1) {
      if (currentRestoreID !== input.getRestoreOperationID()) return
      const restored = restoredTabs[index]
      if (restored == null || restored.conversationId === '') {
        continue
      }
      const tab = findProjectConversationTab(input.getTabs(), restored.tab.id)
      const conversation = input
        .getConversations()
        .find((item) => item.id === restored.conversationId)
      if (!tab) {
        continue
      }

      if (tab.id === preferredActiveTabId) {
        if (await input.runtime.loadTabConversation(tab, restored.conversationId, restored.restored)) {
          loadedTabIDs.add(tab.id)
        } else {
          loadedTabIDs.add(tab.id)
        }
        continue
      }

      if (!conversation) {
        loadedTabIDs.add(tab.id)
        continue
      }

      input.runtime.restoreTabConversationMetadata(tab, conversation, restored.restored)
      loadedTabIDs.add(tab.id)
    }

    const filteredTabs = input.getTabs().filter((tab) => loadedTabIDs.has(tab.id))
    input.setTabs(filteredTabs)
    if (filteredTabs.length === 0) {
      input.ensureTabExists()
      input.persistTabs()
      return
    }

    input.setActiveTabId(
      preferredActiveTabId && filteredTabs.some((tab) => tab.id === preferredActiveTabId)
        ? preferredActiveTabId
        : (filteredTabs[0]?.id ?? ''),
    )
    input.persistTabs()
  } catch (caughtError) {
    input.ensureTabExists()
    input.controllerInput.onError?.(
      caughtError instanceof Error
        ? caughtError.message
        : 'Failed to restore project conversations.',
    )
  }
}

function hasLocalTabActivity(tabs: ProjectConversationTabState[]) {
  return tabs.some(
    (tab) =>
      tab.draft.trim().length > 0 ||
      tab.entries.length > 0 ||
      tab.queuedTurns.length > 0 ||
      tab.phase !== 'idle' ||
      tab.conversationId.trim().length > 0,
  )
}
