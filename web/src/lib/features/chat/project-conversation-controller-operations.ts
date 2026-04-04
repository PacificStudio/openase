import {
  listProjectConversations,
  respondProjectConversationInterrupt,
  type ProjectConversation,
} from '$lib/api/chat'
import { invalidateProjectConversationStream } from './project-conversation-controller-helpers'
import { readProjectConversationTabs } from './project-conversation-storage'
import { createProjectConversationControllerRuntime } from './project-conversation-controller-runtime'
import {
  disposeProjectConversationTabs,
  findProjectConversationTab,
  type CreateProjectConversationControllerInput,
  type ProjectConversationTabState,
} from './project-conversation-controller-state'

type ProjectConversationControllerOperationsInput = {
  controllerInput: CreateProjectConversationControllerInput
  getProviderId: () => string
  getPreferredProviderId: () => string
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  getActiveTabId: () => string
  setActiveTabId: (value: string) => void
  newTabState: (providerId?: string, restored?: boolean) => ProjectConversationTabState
  getActiveTab: () => ProjectConversationTabState | null
  ensureTabExists: () => void
  ensureTabSelection: (preferredTabId?: string) => void
  persistTabs: () => void
  touch: () => void
}

export function createProjectConversationControllerOperations(
  input: ProjectConversationControllerOperationsInput,
) {
  let restoreOperationID = 0
  const runtime = createProjectConversationControllerRuntime(input)

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

  async function restore() {
    const projectId = input.controllerInput.getProjectId()
    if (!projectId) {
      input.ensureTabExists()
      return
    }

    restoreOperationID += 1
    const currentRestoreID = restoreOperationID
    disposeProjectConversationTabs(input.getTabs())

    try {
      const listPayload = await listProjectConversations({ projectId })
      if (currentRestoreID !== restoreOperationID) return

      input.setConversations(runtime.sortProjectConversations(listPayload.conversations))
      if (hasLocalTabActivity(input.getTabs())) {
        input.persistTabs()
        return
      }
      const conversationsByID = new Map(
        listPayload.conversations.map((conversation) => [conversation.id, conversation]),
      )
      const persisted = readProjectConversationTabs(projectId)
      const restoredTabs = persisted.tabs
        .map((persistedTab) => {
          const conversation = persistedTab.conversationId
            ? conversationsByID.get(persistedTab.conversationId)
            : null
          if (conversation != null) {
            const tab = input.newTabState(conversation.providerId, true)
            tab.draft = persistedTab.draft
            return { tab, conversationId: conversation.id, restored: true }
          }
          if (persistedTab.conversationId.trim()) {
            return null
          }
          const tab = input.newTabState(
            persistedTab.providerId || input.getPreferredProviderId(),
            false,
          )
          tab.draft = persistedTab.draft
          return { tab, conversationId: '', restored: false }
        })
        .filter(
          (
            item,
          ): item is {
            tab: ProjectConversationTabState
            conversationId: string
            restored: boolean
          } => item != null,
        )

      if (restoredTabs.length === 0) {
        const latestConversation = input.getConversations()[0] ?? null
        if (latestConversation) {
          const restoredTab = input.newTabState(latestConversation.providerId, true)
          input.setTabs([restoredTab])
          input.setActiveTabId(restoredTab.id)

          const liveRestoredTab = findProjectConversationTab(input.getTabs(), restoredTab.id)
          if (
            liveRestoredTab &&
            (await runtime.loadTabConversation(liveRestoredTab, latestConversation.id, true))
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
        if (currentRestoreID !== restoreOperationID) return
        const restored = restoredTabs[index]
        if (restored == null || restored.conversationId === '') {
          continue
        }
        const tab = findProjectConversationTab(input.getTabs(), restored.tab.id)
        const conversationId = restored.conversationId
        const conversation = input.getConversations().find((item) => item.id === conversationId)
        if (!tab || !conversation) {
          continue
        }

        if (tab.id === preferredActiveTabId) {
          if (await runtime.loadTabConversation(tab, conversationId, restored.restored)) {
            loadedTabIDs.add(tab.id)
          }
          continue
        }

        runtime.restoreTabConversationMetadata(tab, conversation, restored.restored)
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

  async function selectProvider(nextProviderId: string) {
    if (!nextProviderId) {
      return
    }
    const activeTab = input.getActiveTab()
    if (!activeTab || activeTab.phase !== 'idle') {
      return
    }
    if (activeTab.providerId === nextProviderId) {
      return
    }
    if (
      !activeTab.conversationId &&
      activeTab.entries.length === 0 &&
      activeTab.queuedTurns.length === 0
    ) {
      activeTab.providerId = nextProviderId
      activeTab.restored = false
      input.persistTabs()
      return
    }

    const nextTab = input.newTabState(nextProviderId, false)
    input.setTabs([...input.getTabs(), nextTab])
    input.setActiveTabId(nextTab.id)
    input.persistTabs()
  }

  function createTab() {
    const existingBlank = input
      .getTabs()
      .find(
        (tab) =>
          !tab.conversationId &&
          tab.entries.length === 0 &&
          tab.phase === 'idle' &&
          tab.draft.trim().length === 0,
      )
    if (existingBlank) {
      input.setActiveTabId(existingBlank.id)
      input.persistTabs()
      return
    }
    const tab = input.newTabState(input.getProviderId() || input.getPreferredProviderId(), false)
    input.setTabs([...input.getTabs(), tab])
    input.setActiveTabId(tab.id)
    input.persistTabs()
  }

  function selectTab(nextTabId: string) {
    const tab = findProjectConversationTab(input.getTabs(), nextTabId)
    if (!tab) return
    input.setActiveTabId(nextTabId)
    tab.unread = false
    void runtime.hydrateTabIfNeeded(tab)
    input.persistTabs()
  }

  function closeTab(tabId: string) {
    const tab = findProjectConversationTab(input.getTabs(), tabId)
    if (!tab) return
    invalidateProjectConversationStream(tab)
    const remainingTabs = input.getTabs().filter((item) => item.id !== tabId)
    input.setTabs(remainingTabs)
    if (input.getActiveTabId() === tabId) input.setActiveTabId(remainingTabs[0]?.id ?? '')
    input.ensureTabExists()
    input.persistTabs()
  }

  async function resetConversation() {
    await runtime.resetConversation()
  }

  async function respondInterrupt(inputValue: {
    interruptId: string
    decision?: string
    answer?: Record<string, unknown>
  }) {
    const activeTab = input.getActiveTab()
    if (!activeTab?.conversationId || activeTab.phase !== 'awaiting_interrupt') return
    activeTab.phase = 'awaiting_reply'
    try {
      await respondProjectConversationInterrupt(activeTab.conversationId, inputValue.interruptId, {
        decision: inputValue.decision,
        answer: inputValue.answer,
      })
    } catch (caughtError) {
      activeTab.phase = 'awaiting_interrupt'
      input.controllerInput.onError?.(
        caughtError instanceof Error ? caughtError.message : 'Failed to answer interrupt.',
      )
    }
  }

  function dispose() {
    disposeProjectConversationTabs(input.getTabs())
  }

  return {
    openConversationInTab: runtime.openConversationInTab,
    sendTurnInTab: runtime.sendTurnInTab,
    restore,
    selectProvider,
    createTab,
    selectTab,
    closeTab,
    resetConversation,
    respondInterrupt,
    dispose,
  }
}
