import type { ProjectConversation } from '$lib/api/chat'
import { resetProjectConversationRuntime } from './project-conversation-actions'
import { createProjectConversationControllerConversations } from './project-conversation-controller-conversations'
import {
  handleTabStreamEvent,
  reconcileTabAfterReconnect,
  refreshTabWorkspaceDiff,
} from './project-conversation-controller-runtime-effects'
import {
  applyProjectConversationStreamEvent,
  beginProjectConversationOperation,
  connectProjectConversationStream,
  invalidateProjectConversationStream,
  isCurrentProjectConversationOperation,
} from './project-conversation-controller-helpers'
import { loadProjectConversation } from './project-conversation-runtime'
import { createProjectConversationRuntimeTabOps } from './project-conversation-controller-runtime-tab-ops'
import {
  mapPersistedEntries,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import {
  findProjectConversationTab,
  type CreateProjectConversationControllerInput,
  type ProjectConversationTabState,
} from './project-conversation-controller-state'

type ProjectConversationControllerRuntimeInput = {
  controllerInput: CreateProjectConversationControllerInput
  getProviderId: () => string
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
  getTabs: () => ProjectConversationTabState[]
  setTabs: (value: ProjectConversationTabState[]) => void
  setActiveTabId: (value: string) => void
  newTabState: (
    providerId?: string,
    restored?: boolean,
    projectId?: string,
    projectName?: string,
  ) => ProjectConversationTabState
  getActiveTab: () => ProjectConversationTabState | null
  ensureTabExists: () => void
  persistTabs: () => void
  touch: () => void
}

export function createProjectConversationControllerRuntime(
  input: ProjectConversationControllerRuntimeInput,
) {
  function getProjectId() {
    return input.controllerInput.getProjectId()
  }

  function touchTabs() {
    input.touch()
  }

  const isActiveTab = (tab: ProjectConversationTabState) => {
    return input.getActiveTab()?.id === tab.id
  }

  const conversations = createProjectConversationControllerConversations({
    getProjectId: input.controllerInput.getProjectId,
    getProviderId: input.getProviderId,
    getConversations: input.getConversations,
    setConversations: input.setConversations,
  })

  function connectTabStream(tab: ProjectConversationTabState, conversationId: string) {
    const projectId = tab.projectId || getProjectId()
    if (!projectId) {
      return Promise.resolve()
    }
    const streamTab = findProjectConversationTab(input.getTabs(), tab.id) ?? tab
    return connectProjectConversationStream({
      state: streamTab,
      projectId,
      conversationId,
      resolveState: () => findProjectConversationTab(input.getTabs(), streamTab.id),
      onStateChanged: touchTabs,
      onError: input.controllerInput.onError,
      onEvent: (event) => {
        const liveTab = findProjectConversationTab(input.getTabs(), streamTab.id)
        if (!liveTab) {
          return
        }
        if (isActiveTab(liveTab)) {
          applyProjectConversationStreamEvent({
            state: liveTab,
            event,
            onError: input.controllerInput.onError,
          })
        }
        handleTabStreamEvent(
          {
            conversations,
            isActiveTab,
            persistTabs: input.persistTabs,
            touchTabs,
            connectTabStream,
            onError: input.controllerInput.onError,
          },
          liveTab,
          event,
        )
        touchTabs()
      },
      onReconnect: (streamId) =>
        void reconcileTabAfterReconnect(
          {
            conversations,
            isActiveTab,
            persistTabs: input.persistTabs,
            touchTabs,
            connectTabStream,
            onError: input.controllerInput.onError,
          },
          findProjectConversationTab(input.getTabs(), streamTab.id) ?? streamTab,
          conversationId,
          streamId,
        ),
      onRetrying: () => {
        const liveTab = findProjectConversationTab(input.getTabs(), streamTab.id)
        if (!liveTab || !isActiveTab(liveTab) || liveTab.phase !== 'awaiting_reply') {
          return
        }
        liveTab.phase = 'connecting_stream'
        touchTabs()
      },
    })
  }

  async function loadTabConversation(
    tab: ProjectConversationTabState,
    conversationId: string,
    restored: boolean,
  ) {
    const previousPhase = tab.phase
    const currentOperationId = beginProjectConversationOperation(tab, 'restoring')
    try {
      await loadProjectConversation({
        conversationId,
        mapEntries: mapPersistedEntries,
        setConversationId: (nextId) => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            tab.conversationId = nextId
            tab.needsHydration = false
            tab.unread = false
            touchTabs()
          }
        },
        setEntries: (nextEntries) => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            const mappedEntries = nextEntries as ProjectConversationTranscriptEntry[]
            tab.entries = mappedEntries
            tab.entryCounter = mappedEntries.length
            tab.needsHydration = false
            tab.unread = false
            touchTabs()
          }
        },
        resetActiveAssistantEntry: () => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            tab.activeAssistantEntryId = ''
            touchTabs()
          }
        },
        connectStream: (nextConversationId) => connectTabStream(tab, nextConversationId),
      })
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
        tab.restored = restored
        if (tab.phase === 'restoring') {
          tab.phase = previousPhase === 'restoring' ? 'idle' : previousPhase
        }
      }
      touchTabs()
      void refreshTabWorkspaceDiff(tab, conversationId, touchTabs)
      input.persistTabs()
      return true
    } catch {
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
        tab.phase = 'idle'
        touchTabs()
      }
      return false
    }
  }

  const tabOps = createProjectConversationRuntimeTabOps({
    controllerInput: input.controllerInput,
    getProjectId,
    getProviderId: input.getProviderId,
    getConversations: input.getConversations,
    setConversations: input.setConversations,
    getTabs: input.getTabs,
    setTabs: input.setTabs,
    setActiveTabId: input.setActiveTabId,
    newTabState: input.newTabState,
    getActiveTab: input.getActiveTab,
    ensureTabExists: input.ensureTabExists,
    persistTabs: input.persistTabs,
    loadTabConversation,
    connectTabStream,
    sortProjectConversations: conversations.sortProjectConversations,
    touch: touchTabs,
    touchConversation: conversations.touchConversation,
  })

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
      activeTab.needsHydration = false
      activeTab.unread = false
      activeTab.phase = 'connecting_stream'
      connectTabStream(activeTab, activeConversationId)
      await refreshTabWorkspaceDiff(activeTab, activeConversationId, touchTabs)
    } else {
      activeTab.entries = []
      activeTab.workspaceDiff = null
      activeTab.workspaceDiffLoading = false
      activeTab.workspaceDiffError = ''
      activeTab.needsHydration = false
      activeTab.unread = false
    }
    if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) return
    activeTab.queuedTurns = []
    activeTab.phase = 'idle'
    activeTab.restored = false
    touchTabs()
    input.persistTabs()
  }

  return {
    sortProjectConversations: conversations.sortProjectConversations,
    loadTabConversation,
    async refreshWorkspaceDiff(tab: ProjectConversationTabState | null, conversationId: string) {
      if (!tab || !conversationId) {
        return
      }
      await refreshTabWorkspaceDiff(tab, conversationId, touchTabs)
    },
    restoreTabConversationMetadata: tabOps.restoreTabConversationMetadata,
    hydrateTabIfNeeded: tabOps.hydrateTabIfNeeded,
    openConversationInTab: tabOps.openConversationInTab,
    sendTurnInTab: tabOps.sendTurnInTab,
    resetConversation,
  }
}
