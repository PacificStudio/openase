import {
  createProjectConversation,
  getProjectConversationWorkspaceDiff,
  startProjectConversationTurn,
  type ProjectConversation,
  type ProjectConversationStreamEvent,
  type ProjectConversationTurnRequest,
} from '$lib/api/chat'
import { resetProjectConversationRuntime } from './project-conversation-actions'
import { createProjectConversationControllerConversations } from './project-conversation-controller-conversations'
import {
  appendProjectConversationText,
  beginProjectConversationOperation,
  connectProjectConversationStream,
  invalidateProjectConversationStream,
  isCurrentProjectConversationOperation,
  projectConversationHasPendingInterrupt,
  setProjectConversationIdleIfCurrent,
} from './project-conversation-controller-helpers'
import { loadProjectConversation } from './project-conversation-runtime'
import { mapPersistedEntries, type ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'
import type { ProjectAIFocus } from './project-ai-focus'
import type { CreateProjectConversationControllerInput, ProjectConversationTabState } from './project-conversation-controller-state'

type ProjectConversationControllerRuntimeInput = {
  controllerInput: CreateProjectConversationControllerInput
  getProviderId: () => string
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  setActiveTabId: (value: string) => void
  newTabState: (restored?: boolean) => ProjectConversationTabState
  getActiveTab: () => ProjectConversationTabState | null
  ensureTabExists: () => void
  persistTabs: () => void
}

export function createProjectConversationControllerRuntime(input: ProjectConversationControllerRuntimeInput) {
  const conversations = createProjectConversationControllerConversations({
    getProjectId: input.controllerInput.getProjectId,
    getProviderId: input.getProviderId,
    getConversations: input.getConversations,
    setConversations: input.setConversations,
  })

  function connectTabStream(tab: ProjectConversationTabState, conversationId: string) {
    connectProjectConversationStream({
      state: tab,
      conversationId,
      onError: input.controllerInput.onError,
      onEvent: (event) => handleTabStreamEvent(tab, event),
    })
  }

  async function refreshTabWorkspaceDiff(tab: ProjectConversationTabState, conversationId: string) {
    if (!conversationId) {
      tab.workspaceDiff = null
      tab.workspaceDiffLoading = false
      tab.workspaceDiffError = ''
      return
    }

    tab.workspaceDiffRequestId += 1
    const currentRequestId = tab.workspaceDiffRequestId
    tab.workspaceDiffLoading = true
    tab.workspaceDiffError = ''
    try {
      const payload = await getProjectConversationWorkspaceDiff(conversationId)
      if (currentRequestId !== tab.workspaceDiffRequestId || tab.conversationId !== conversationId) return
      tab.workspaceDiff = payload.workspaceDiff
    } catch (caughtError) {
      if (currentRequestId !== tab.workspaceDiffRequestId || tab.conversationId !== conversationId) return
      tab.workspaceDiff = null
      tab.workspaceDiffError =
        caughtError instanceof Error
          ? caughtError.message
          : 'Failed to load Project AI workspace changes.'
    } finally {
      if (currentRequestId === tab.workspaceDiffRequestId && tab.conversationId === conversationId) {
        tab.workspaceDiffLoading = false
      }
    }
  }

  function handleTabStreamEvent(tab: ProjectConversationTabState, event: ProjectConversationStreamEvent) {
    if (event.kind === 'session') {
      tab.conversationId = conversations.applySessionPayload(tab.conversationId, event.payload)
    }
    if ((event.kind === 'session' || event.kind === 'turn_done') && tab.conversationId) {
      void refreshTabWorkspaceDiff(tab, tab.conversationId)
    }
  }

  async function loadTabConversation(
    tab: ProjectConversationTabState,
    conversationId: string,
    restored: boolean,
  ) {
    const currentOperationId = beginProjectConversationOperation(tab, 'restoring')
    try {
      await loadProjectConversation({
        conversationId,
        mapEntries: mapPersistedEntries,
        setConversationId: (nextId) => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) tab.conversationId = nextId
        },
        setEntries: (nextEntries) => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            const mappedEntries = nextEntries as ProjectConversationTranscriptEntry[]
            tab.entries = mappedEntries
            tab.entryCounter = mappedEntries.length
          }
        },
        resetActiveAssistantEntry: () => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) tab.activeAssistantEntryId = ''
        },
        connectStream: (nextConversationId) => connectTabStream(tab, nextConversationId),
      })
      await refreshTabWorkspaceDiff(tab, conversationId)
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) tab.restored = restored
      setProjectConversationIdleIfCurrent(tab, currentOperationId)
      input.persistTabs()
      return true
    } catch {
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) tab.phase = 'idle'
      return false
    }
  }

  async function openConversationInTab(nextConversationId: string) {
    if (!nextConversationId) return
    const existing = input.getTabs().find((tab) => tab.conversationId === nextConversationId)
    if (existing) {
      input.setActiveTabId(existing.id)
      input.persistTabs()
      return
    }

    const activeTab = input.getActiveTab()
    const target =
      activeTab &&
      !activeTab.conversationId &&
      activeTab.entries.length === 0 &&
      activeTab.phase === 'idle' &&
      activeTab.draft.trim().length === 0
        ? activeTab
        : input.newTabState(false)
    if (target !== activeTab) input.setTabs([...input.getTabs(), target])
    input.setActiveTabId(target.id)

    if (!(await loadTabConversation(target, nextConversationId, false))) {
      input.setTabs(input.getTabs().filter((tab) => tab.id !== target.id))
      input.ensureTabExists()
      input.controllerInput.onError?.('Failed to open project conversation.')
      input.persistTabs()
      return
    }

    conversations.touchConversation(nextConversationId)
  }

  async function sendTurnInTab(
    activeTab: ProjectConversationTabState | null,
    message: string,
    focus?: ProjectAIFocus | null,
  ) {
    const trimmed = message.trim()
    const projectId = input.controllerInput.getProjectId()
    const providerId = input.getProviderId()
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
        input.setConversations(
          conversations.sortProjectConversations([
            createPayload.conversation,
            ...input.getConversations().filter((conversation) => conversation.id !== createPayload.conversation.id),
          ]),
        )
        input.persistTabs()
        activeTab.phase = 'connecting_stream'
        connectTabStream(activeTab, activeTab.conversationId)
      } else if (!activeTab.abortController) {
        activeTab.phase = 'connecting_stream'
        connectTabStream(activeTab, activeTab.conversationId)
      }

      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false

      appendProjectConversationText(activeTab, 'user', trimmed)
      activeTab.activeAssistantEntryId = ''
      activeTab.restored = false
      activeTab.phase = 'submitting_turn'
      await startProjectConversationTurn(activeTab.conversationId, {
        message: trimmed,
        focus: focus ?? undefined,
      } satisfies ProjectConversationTurnRequest)
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return false
      conversations.touchConversation(activeTab.conversationId)
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

  async function resetConversation() {
    const activeTab = input.getActiveTab()
    if (!activeTab) return
    const currentOperationId = beginProjectConversationOperation(activeTab, 'resetting')
    const activeConversationId = activeTab.conversationId
    invalidateProjectConversationStream(activeTab)
    if (activeConversationId) {
      await resetProjectConversationRuntime(activeConversationId)
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return
      activeTab.activeAssistantEntryId = ''
      activeTab.phase = 'connecting_stream'
      connectTabStream(activeTab, activeConversationId)
      await refreshTabWorkspaceDiff(activeTab, activeConversationId)
    } else {
      activeTab.entries = []
      activeTab.workspaceDiff = null
      activeTab.workspaceDiffLoading = false
      activeTab.workspaceDiffError = ''
    }
    if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return
    activeTab.queuedTurns = []
    activeTab.phase = 'idle'
    activeTab.restored = false
    input.persistTabs()
  }

  return {
    sortProjectConversations: conversations.sortProjectConversations,
    loadTabConversation,
    openConversationInTab,
    sendTurnInTab,
    resetConversation,
  }
}
