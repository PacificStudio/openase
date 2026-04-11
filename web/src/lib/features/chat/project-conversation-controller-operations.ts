import {
  deleteProjectConversation,
  interruptProjectConversationTurn,
  respondProjectConversationInterrupt,
  type ProjectConversation,
} from '$lib/api/chat'
import { invalidateProjectConversationStream } from './project-conversation-controller-helpers'
import { restoreProjectConversationController } from './project-conversation-controller-restore-helpers'
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
  newTabState: (
    providerId?: string,
    restored?: boolean,
    projectId?: string,
    projectName?: string,
  ) => ProjectConversationTabState
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

  async function restore() {
    restoreOperationID += 1
    await restoreProjectConversationController({
      controllerInput: input.controllerInput,
      runtime,
      getRestoreOperationID: () => restoreOperationID,
      getPreferredProviderId: input.getPreferredProviderId,
      getTabs: input.getTabs,
      setTabs: input.setTabs,
      setConversations: input.setConversations,
      getConversations: input.getConversations,
      setActiveTabId: input.setActiveTabId,
      newTabState: input.newTabState,
      ensureTabExists: input.ensureTabExists,
      persistTabs: input.persistTabs,
    })
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
    const currentProjectId = input.controllerInput.getProjectId()
    const existingBlank = input
      .getTabs()
      .find(
        (tab) =>
          !tab.conversationId &&
          tab.entries.length === 0 &&
          tab.phase === 'idle' &&
          tab.draft.trim().length === 0 &&
          tab.projectId === currentProjectId,
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

  async function refreshWorkspaceDiff() {
    const activeTab = input.getActiveTab()
    const conversationId = activeTab?.conversationId ?? ''
    if (!activeTab || !conversationId) {
      return
    }
    await runtime.refreshWorkspaceDiff(activeTab, conversationId)
  }

  async function stopTurn() {
    const activeTab = input.getActiveTab()
    if (
      !activeTab?.conversationId ||
      (activeTab.phase !== 'awaiting_reply' &&
        activeTab.phase !== 'connecting_stream' &&
        activeTab.phase !== 'stopping_turn')
    ) {
      return
    }
    const previousPhase = activeTab.phase
    activeTab.phase = 'stopping_turn'
    try {
      await interruptProjectConversationTurn(activeTab.conversationId)
    } catch (caughtError) {
      activeTab.phase = previousPhase
      input.controllerInput.onError?.(
        caughtError instanceof Error ? caughtError.message : 'Failed to stop the current turn.',
      )
    }
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

  async function deleteConversation(conversationId: string, options: { force?: boolean } = {}) {
    if (!conversationId) {
      return false
    }

    await deleteProjectConversation(conversationId, options)

    for (const tab of input.getTabs()) {
      if (tab.conversationId === conversationId) {
        invalidateProjectConversationStream(tab)
      }
    }

    const remainingTabs = input.getTabs().filter((tab) => tab.conversationId !== conversationId)
    input.setTabs(remainingTabs)
    input.setConversations(
      input.getConversations().filter((conversation) => conversation.id !== conversationId),
    )
    if (!remainingTabs.some((tab) => tab.id === input.getActiveTabId())) {
      input.setActiveTabId(remainingTabs[0]?.id ?? '')
    }
    input.ensureTabExists()
    input.persistTabs()
    return true
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
    refreshWorkspaceDiff,
    resetConversation,
    stopTurn,
    respondInterrupt,
    deleteConversation,
    dispose,
  }
}
