import {
  createProjectConversation,
  startProjectConversationTurn,
  type ProjectConversation,
  type ProjectConversationTurnRequest,
} from '$lib/api/chat'
import { projectConversationTabPhaseFromRuntimeState } from './project-conversation-controller-runtime-effects'
import {
  appendProjectConversationText,
  beginProjectConversationOperation,
  isCurrentProjectConversationOperation,
  projectConversationHasPendingInterrupt,
} from './project-conversation-controller-helpers'
import type { ProjectAIFocus } from './project-ai-focus'
import type {
  CreateProjectConversationControllerInput,
  ProjectConversationTabState,
} from './project-conversation-controller-state'

type ProjectConversationRuntimeTabOpsInput = {
  controllerInput: CreateProjectConversationControllerInput
  getProjectId: () => string
  getProviderId: () => string
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  setActiveTabId: (value: string) => void
  newTabState: (providerId?: string, restored?: boolean) => ProjectConversationTabState
  getActiveTab: () => ProjectConversationTabState | null
  ensureTabExists: () => void
  persistTabs: () => void
  loadTabConversation: (
    tab: ProjectConversationTabState,
    conversationId: string,
    restored: boolean,
  ) => Promise<boolean>
  connectTabStream: (tab: ProjectConversationTabState, conversationId: string) => Promise<void>
  sortProjectConversations: (value: ProjectConversation[]) => ProjectConversation[]
  touchConversation: (conversationId: string) => void
}

export function createProjectConversationRuntimeTabOps(
  input: ProjectConversationRuntimeTabOpsInput,
) {
  async function waitForStreamConnection(
    tab: ProjectConversationTabState,
    conversationId: string,
    timeoutMs = 3000,
  ) {
    await Promise.race([
      input.connectTabStream(tab, conversationId),
      new Promise<void>((resolve) => {
        window.setTimeout(resolve, timeoutMs)
      }),
    ])
  }

  function restoreTabConversationMetadata(
    tab: ProjectConversationTabState,
    conversation: ProjectConversation,
    restored: boolean,
  ) {
    tab.conversationId = conversation.id
    tab.providerId = conversation.providerId || tab.providerId
    tab.entries = []
    tab.entryCounter = 0
    tab.activeAssistantEntryId = ''
    tab.restored = restored
    tab.needsHydration = true
    tab.unread = false
    tab.phase = projectConversationTabPhaseFromRuntimeState(
      conversation.runtimePrincipal?.runtimeState,
      conversation.providerStatus,
      conversation.providerActiveFlags,
    )
    input.connectTabStream(tab, conversation.id)
  }

  async function hydrateTabIfNeeded(tab: ProjectConversationTabState | null) {
    if (!tab?.conversationId || !tab.needsHydration) {
      return
    }
    await input.loadTabConversation(tab, tab.conversationId, tab.restored)
  }

  async function openConversationInTab(nextConversationId: string) {
    if (!nextConversationId) return
    const existing = input.getTabs().find((tab) => tab.conversationId === nextConversationId)
    if (existing) {
      input.setActiveTabId(existing.id)
      existing.unread = false
      void hydrateTabIfNeeded(existing)
      input.persistTabs()
      return
    }

    const activeTab = input.getActiveTab()
    const conversation =
      input.getConversations().find((item) => item.id === nextConversationId) ?? null
    const target =
      activeTab &&
      !activeTab.conversationId &&
      activeTab.entries.length === 0 &&
      activeTab.phase === 'idle' &&
      activeTab.draft.trim().length === 0
        ? activeTab
        : input.newTabState(conversation?.providerId ?? input.getProviderId(), false)
    if (conversation?.providerId) {
      target.providerId = conversation.providerId
    }
    if (target !== activeTab) input.setTabs([...input.getTabs(), target])
    input.setActiveTabId(target.id)

    if (!(await input.loadTabConversation(target, nextConversationId, false))) {
      input.setTabs(input.getTabs().filter((tab) => tab.id !== target.id))
      input.ensureTabExists()
      input.controllerInput.onError?.('Failed to open project conversation.')
      input.persistTabs()
      return
    }

    input.touchConversation(nextConversationId)
  }

  async function sendTurnInTab(
    activeTab: ProjectConversationTabState | null,
    message: string,
    focus?: ProjectAIFocus | null,
  ) {
    const trimmed = message.trim()
    const projectId = input.getProjectId()
    const providerId = activeTab?.providerId?.trim() ?? ''
    if (
      !trimmed ||
      !projectId ||
      !providerId ||
      activeTab == null ||
      activeTab.phase !== 'idle' ||
      projectConversationHasPendingInterrupt(activeTab.entries)
    ) {
      return false
    }

    const currentOperationId = beginProjectConversationOperation(
      activeTab,
      activeTab.conversationId ? 'submitting_turn' : 'creating_conversation',
    )

    try {
      if (!activeTab.conversationId) {
        const createPayload = await createProjectConversation({ providerId, projectId })
        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false
        activeTab.conversationId = createPayload.conversation.id
        activeTab.restored = false
        activeTab.needsHydration = false
        activeTab.unread = false
        input.setConversations(
          input.sortProjectConversations([
            createPayload.conversation,
            ...input
              .getConversations()
              .filter((conversation) => conversation.id !== createPayload.conversation.id),
          ]),
        )
        input.persistTabs()
        activeTab.phase = 'connecting_stream'
        await waitForStreamConnection(activeTab, activeTab.conversationId)
      } else if (!activeTab.abortController) {
        activeTab.phase = 'connecting_stream'
        await waitForStreamConnection(activeTab, activeTab.conversationId)
      }

      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false

      appendProjectConversationText(activeTab, 'user', trimmed)
      activeTab.activeAssistantEntryId = ''
      activeTab.restored = false
      activeTab.needsHydration = false
      activeTab.unread = false
      activeTab.phase = 'submitting_turn'
      await startProjectConversationTurn(activeTab.conversationId, {
        message: trimmed,
        focus: focus ?? undefined,
      } satisfies ProjectConversationTurnRequest)
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false
      input.touchConversation(activeTab.conversationId)
      if (projectConversationHasPendingInterrupt(activeTab.entries)) {
        activeTab.phase = 'awaiting_interrupt'
      } else if (activeTab.phase === 'submitting_turn') {
        activeTab.phase = 'awaiting_reply'
      }
      input.persistTabs()
      return true
    } catch (caughtError) {
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false
      activeTab.phase = 'idle'
      input.controllerInput.onError?.(
        caughtError instanceof Error ? caughtError.message : 'Failed to send project message.',
      )
      return false
    }
  }

  return {
    restoreTabConversationMetadata,
    hydrateTabIfNeeded,
    openConversationInTab,
    sendTurnInTab,
  }
}
