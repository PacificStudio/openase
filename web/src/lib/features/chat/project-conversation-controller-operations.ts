import {
  listProjectConversations,
  respondProjectConversationInterrupt,
  type ProjectConversation,
} from '$lib/api/chat'
import { confirmProjectConversationActionProposal } from './project-conversation-actions'
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
  setProviderId: (value: string) => void
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  getActiveTabId: () => string
  setActiveTabId: (value: string) => void
  newTabState: (restored?: boolean) => ProjectConversationTabState
  getActiveTab: () => ProjectConversationTabState | null
  ensureTabExists: () => void
  ensureTabSelection: (preferredTabId?: string) => void
  persistTabs: () => void
}

export function createProjectConversationControllerOperations(input: ProjectConversationControllerOperationsInput) {
  let restoreOperationID = 0
  const runtime = createProjectConversationControllerRuntime(input)

  async function restore() {
    const projectId = input.controllerInput.getProjectId()
    const providerId = input.getProviderId()
    if (!projectId || !providerId) {
      input.ensureTabExists()
      return
    }

    restoreOperationID += 1
    const currentRestoreID = restoreOperationID
    disposeProjectConversationTabs(input.getTabs())

    try {
      const listPayload = await listProjectConversations({ projectId, providerId })
      if (currentRestoreID !== restoreOperationID) return

      input.setConversations(runtime.sortProjectConversations(listPayload.conversations))
      const availableConversationIDs = new Set(listPayload.conversations.map((conversation) => conversation.id))
      const persisted = readProjectConversationTabs(projectId, providerId)
      const restoredConversationIDs = persisted.conversationIds.filter((conversationId) =>
        availableConversationIDs.has(conversationId),
      )
      const fallbackConversationID =
        restoredConversationIDs.length === 0 ? (listPayload.conversations[0]?.id ?? '') : ''
      const preferredConversationID =
        (persisted.activeConversationId && availableConversationIDs.has(persisted.activeConversationId)
          ? persisted.activeConversationId
          : restoredConversationIDs[0]) || fallbackConversationID
      const initialConversationIDs = preferredConversationID ? [preferredConversationID] : []

      if (initialConversationIDs.length === 0) {
        const activeTab = input.getActiveTab()
        const reusableBlank =
          activeTab && !activeTab.conversationId && activeTab.entries.length === 0
            ? activeTab
            : (input.getTabs().find((tab) => !tab.conversationId && tab.entries.length === 0) ?? null)
        const blankTab = reusableBlank ?? input.newTabState(false)
        blankTab.restored = false
        blankTab.phase = 'idle'
        input.setTabs([blankTab])
        input.setActiveTabId(blankTab.id)
        input.persistTabs()
        return
      }

      input.setTabs(initialConversationIDs.map(() => input.newTabState(true)))
      const loadedTabIDs = new Set<string>()
      for (let index = 0; index < initialConversationIDs.length; index += 1) {
        if (currentRestoreID !== restoreOperationID) return
        const tab = input.getTabs()[index]
        const conversationId = initialConversationIDs[index]
        if (tab && (await runtime.loadTabConversation(tab, conversationId, restoredConversationIDs.includes(conversationId)))) {
          loadedTabIDs.add(tab.id)
        }
      }

      input.setTabs(input.getTabs().filter((tab) => loadedTabIDs.has(tab.id)))
      if (input.getTabs().length === 0) {
        input.ensureTabExists()
        input.persistTabs()
        return
      }
      input.ensureTabSelection(input.getTabs()[0]?.id ?? '')
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
    if (!nextProviderId || input.getProviderId() === nextProviderId || input.getTabs().some((tab) => tab.phase !== 'idle')) {
      return
    }
    disposeProjectConversationTabs(input.getTabs())
    input.setProviderId(nextProviderId)
    input.setConversations([])
    input.setTabs([input.newTabState(false)])
    input.setActiveTabId(input.getTabs()[0]?.id ?? '')
    input.persistTabs()
  }

  function createTab() {
    const existingBlank = input.getTabs().find(
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
    const tab = input.newTabState(false)
    input.setTabs([...input.getTabs(), tab])
    input.setActiveTabId(tab.id)
    input.persistTabs()
  }

  function selectTab(nextTabId: string) {
    if (!findProjectConversationTab(input.getTabs(), nextTabId)) return
    input.setActiveTabId(nextTabId)
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

  async function confirmActionProposal(entryId: string) {
    const activeTab = input.getActiveTab()
    if (!activeTab?.conversationId) return
    await confirmProjectConversationActionProposal({
      conversationId: activeTab.conversationId,
      entryId,
      entries: activeTab.entries,
      setEntries: (nextEntries) => {
        activeTab.entries = nextEntries
      },
      onError: input.controllerInput.onError,
    })
  }

  function cancelActionProposal(entryId: string) {
    const activeTab = input.getActiveTab()
    if (!activeTab) return
    activeTab.entries = activeTab.entries.map((entry) =>
      entry.kind === 'action_proposal' && entry.id === entryId && entry.status === 'pending'
        ? { ...entry, status: 'cancelled' }
        : entry,
    )
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
    confirmActionProposal,
    cancelActionProposal,
    respondInterrupt,
    dispose,
  }
}
