import {
  createProjectConversation,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  type ProjectConversation,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import {
  confirmProjectConversationActionProposal,
  resetProjectConversationRuntime,
} from './project-conversation-actions'
import {
  appendProjectConversationText,
  beginProjectConversationOperation,
  connectProjectConversationStream,
  invalidateProjectConversationStream,
  isCurrentProjectConversationOperation,
  projectConversationHasPendingInterrupt,
  setProjectConversationIdleIfCurrent,
  type ProjectConversationControllerState,
  type ProjectConversationPhase,
} from './project-conversation-controller-helpers'
import {
  mapPersistedEntries,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import { loadProjectConversation } from './project-conversation-runtime'
import {
  readProjectConversationTabs,
  storeProjectConversationTabs,
} from './project-conversation-storage'
import {
  listEphemeralChatProviders,
  pickDefaultEphemeralChatProvider,
  shouldKeepEphemeralChatProvider,
} from './provider-options'

type CreateProjectConversationControllerInput = {
  getProjectId: () => string
  onError?: (message: string) => void
}

type ProjectConversationTabState = ProjectConversationControllerState & {
  id: string
  restored: boolean
}

type ProjectConversationTabSummary = {
  id: string
  conversationId: string
  phase: ProjectConversationPhase
  pending: boolean
  hasPendingInterrupt: boolean
  restored: boolean
  entries: ProjectConversationTranscriptEntry[]
}

export function createProjectConversationController(
  input: CreateProjectConversationControllerInput,
) {
  let providers = $state<AgentProvider[]>([])
  let providerId = $state('')
  let conversations = $state<ProjectConversation[]>([])
  let tabs = $state<ProjectConversationTabState[]>([])
  let activeTabId = $state('')
  let nextTabID = 0
  let restoreOperationID = 0

  function newTabState(restored = false): ProjectConversationTabState {
    nextTabID += 1
    return {
      id: `tab-${nextTabID}`,
      restored,
      phase: 'idle',
      conversationId: '',
      entries: [],
      activeAssistantEntryId: '',
      abortController: null,
      entryCounter: 0,
      operationId: 0,
      streamId: 0,
    }
  }

  function findTab(tabId: string) {
    return tabs.find((tab) => tab.id === tabId) ?? null
  }

  function getActiveTab() {
    return findTab(activeTabId) ?? tabs[0] ?? null
  }

  function summarizeTab(tab: ProjectConversationTabState): ProjectConversationTabSummary {
    return {
      id: tab.id,
      conversationId: tab.conversationId,
      phase: tab.phase,
      pending: isTabPending(tab),
      hasPendingInterrupt: projectConversationHasPendingInterrupt(tab.entries),
      restored: tab.restored,
      entries: tab.entries,
    }
  }

  function isTabPending(tab: ProjectConversationTabState) {
    return (
      tab.phase === 'creating_conversation' ||
      tab.phase === 'connecting_stream' ||
      tab.phase === 'submitting_turn' ||
      tab.phase === 'awaiting_reply'
    )
  }

  function ensureTabSelection(preferredTabId = '') {
    if (preferredTabId && findTab(preferredTabId) != null) {
      activeTabId = preferredTabId
      return
    }
    if (findTab(activeTabId) != null) {
      return
    }
    activeTabId = tabs[0]?.id ?? ''
  }

  function persistTabs() {
    const projectId = input.getProjectId()
    if (!projectId || !providerId) {
      return
    }
    const activeConversationId = getActiveTab()?.conversationId ?? ''
    storeProjectConversationTabs(projectId, providerId, {
      conversationIds: tabs
        .map((tab) => tab.conversationId.trim())
        .filter((conversationId) => conversationId.length > 0),
      activeConversationId,
    })
  }

  function connectTabStream(tab: ProjectConversationTabState, nextConversationId: string) {
    connectProjectConversationStream({
      state: tab,
      conversationId: nextConversationId,
      onError: input.onError,
    })
  }

  function touchConversation(conversationId: string) {
    if (!conversationId) {
      return
    }
    const now = new Date().toISOString()
    conversations = [
      ...conversations
        .map((conversation) =>
          conversation.id === conversationId
            ? { ...conversation, lastActivityAt: now }
            : conversation,
        )
        .sort((left, right) => right.lastActivityAt.localeCompare(left.lastActivityAt)),
    ]
  }

  function upsertConversation(conversation: ProjectConversation) {
    conversations = [
      conversation,
      ...conversations.filter((current) => current.id !== conversation.id),
    ]
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
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            tab.conversationId = nextId
          }
        },
        setEntries: (nextEntries) => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            const mappedEntries = nextEntries as ProjectConversationTranscriptEntry[]
            tab.entries = mappedEntries
            tab.entryCounter = mappedEntries.length
          }
        },
        resetActiveAssistantEntry: () => {
          if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
            tab.activeAssistantEntryId = ''
          }
        },
        connectStream: (nextConversationId) => connectTabStream(tab, nextConversationId),
      })
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
        tab.restored = restored
      }
      setProjectConversationIdleIfCurrent(tab, currentOperationId)
      persistTabs()
      return true
    } catch {
      if (isCurrentProjectConversationOperation(tab, currentOperationId)) {
        tab.phase = 'idle'
      }
      return false
    }
  }

  function disposeAllTabs() {
    for (const tab of tabs) {
      invalidateProjectConversationStream(tab)
      tab.phase = 'idle'
    }
  }

  function ensureTabExists() {
    if (tabs.length > 0) {
      ensureTabSelection()
      return
    }
    const tab = newTabState(false)
    tabs = [tab]
    activeTabId = tab.id
  }

  async function openConversationInTab(nextConversationId: string) {
    if (!nextConversationId) {
      return
    }
    const existing = tabs.find((tab) => tab.conversationId === nextConversationId)
    if (existing) {
      activeTabId = existing.id
      persistTabs()
      return
    }

    const activeTab = getActiveTab()
    const target =
      activeTab &&
      !activeTab.conversationId &&
      activeTab.entries.length === 0 &&
      activeTab.phase === 'idle'
        ? activeTab
        : newTabState(false)
    if (target !== activeTab) {
      tabs = [...tabs, target]
    }
    activeTabId = target.id

    const loaded = await loadTabConversation(target, nextConversationId, false)
    if (!loaded) {
      tabs = tabs.filter((tab) => tab.id !== target.id)
      ensureTabExists()
      input.onError?.('Failed to open project conversation.')
      persistTabs()
      return
    }

    touchConversation(nextConversationId)
  }

  ensureTabExists()

  return {
    get providers() {
      return providers
    },
    get conversations() {
      return conversations
    },
    get tabs() {
      return tabs.map((tab) => summarizeTab(tab))
    },
    get activeTabId() {
      return activeTabId
    },
    get providerId() {
      return providerId
    },
    get phase() {
      return getActiveTab()?.phase ?? 'idle'
    },
    get selectedProvider() {
      return providers.find((provider) => provider.id === providerId) ?? null
    },
    get busy() {
      return (getActiveTab()?.phase ?? 'idle') !== 'idle'
    },
    get pending() {
      const activeTab = getActiveTab()
      return activeTab ? isTabPending(activeTab) : false
    },
    get conversationId() {
      return getActiveTab()?.conversationId ?? ''
    },
    get entries() {
      return getActiveTab()?.entries ?? []
    },
    get hasPendingInterrupt() {
      const activeTab = getActiveTab()
      return activeTab ? projectConversationHasPendingInterrupt(activeTab.entries) : false
    },
    get inputDisabled() {
      const activeTab = getActiveTab()
      return (
        !input.getProjectId() ||
        !providerId ||
        activeTab == null ||
        activeTab.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(activeTab.entries)
      )
    },
    get providerSelectionDisabled() {
      return tabs.some((tab) => tab.phase !== 'idle')
    },
    syncProviders(nextProviders: AgentProvider[], defaultProviderId: string | null | undefined) {
      providers = listEphemeralChatProviders(nextProviders)
      if (shouldKeepEphemeralChatProvider(providers, providerId)) {
        return
      }
      providerId = pickDefaultEphemeralChatProvider(providers, defaultProviderId)
    },
    async restore() {
      const projectId = input.getProjectId()
      if (!projectId || !providerId) {
        ensureTabExists()
        return
      }

      restoreOperationID += 1
      const currentRestoreID = restoreOperationID
      disposeAllTabs()

      try {
        const listPayload = await listProjectConversations({ projectId, providerId })
        if (currentRestoreID !== restoreOperationID) {
          return
        }

        conversations = listPayload.conversations
        const availableConversationIDs = new Set(
          listPayload.conversations.map((conversation) => conversation.id),
        )
        const persisted = readProjectConversationTabs(projectId, providerId)
        const restoredConversationIDs = persisted.conversationIds.filter((conversationId) =>
          availableConversationIDs.has(conversationId),
        )
        const fallbackConversationID =
          restoredConversationIDs.length === 0 ? (listPayload.conversations[0]?.id ?? '') : ''
        const initialConversationIDs =
          restoredConversationIDs.length > 0
            ? restoredConversationIDs
            : fallbackConversationID
              ? [fallbackConversationID]
              : []

        tabs = initialConversationIDs.map(() => newTabState(true))
        if (tabs.length === 0) {
          ensureTabExists()
          persistTabs()
          return
        }

        const loadedTabIDs = new Set<string>()
        for (let index = 0; index < initialConversationIDs.length; index += 1) {
          if (currentRestoreID !== restoreOperationID) {
            return
          }
          const tab = tabs[index]
          const conversationId = initialConversationIDs[index]
          const restored = restoredConversationIDs.includes(conversationId)
          if (await loadTabConversation(tab, conversationId, restored)) {
            loadedTabIDs.add(tab.id)
          }
        }

        tabs = tabs.filter((tab) => loadedTabIDs.has(tab.id))
        if (tabs.length === 0) {
          ensureTabExists()
          persistTabs()
          return
        }

        const preferredConversationId = persisted.activeConversationId
        const preferredTabId =
          tabs.find((tab) => tab.conversationId === preferredConversationId)?.id ?? tabs[0].id
        ensureTabSelection(preferredTabId)
        persistTabs()
      } catch (caughtError) {
        ensureTabExists()
        input.onError?.(
          caughtError instanceof Error
            ? caughtError.message
            : 'Failed to restore project conversations.',
        )
      }
    },
    async selectProvider(nextProviderId: string) {
      if (
        !nextProviderId ||
        providerId === nextProviderId ||
        tabs.some((tab) => tab.phase !== 'idle')
      ) {
        return
      }
      disposeAllTabs()
      providerId = nextProviderId
      conversations = []
      tabs = [newTabState(false)]
      activeTabId = tabs[0].id
      persistTabs()
    },
    createTab() {
      const existingBlank = tabs.find(
        (tab) => !tab.conversationId && tab.entries.length === 0 && tab.phase === 'idle',
      )
      if (existingBlank) {
        activeTabId = existingBlank.id
        persistTabs()
        return
      }

      const tab = newTabState(false)
      tabs = [...tabs, tab]
      activeTabId = tab.id
      persistTabs()
    },
    async openConversation(nextConversationId: string) {
      await openConversationInTab(nextConversationId)
    },
    async selectConversation(nextConversationId: string) {
      if (!nextConversationId) {
        return
      }
      await openConversationInTab(nextConversationId)
    },
    selectTab(nextTabId: string) {
      if (!findTab(nextTabId)) {
        return
      }
      activeTabId = nextTabId
      persistTabs()
    },
    closeTab(tabId: string) {
      const tab = findTab(tabId)
      if (!tab) {
        return
      }

      invalidateProjectConversationStream(tab)
      const remainingTabs = tabs.filter((item) => item.id !== tabId)
      tabs = remainingTabs
      if (activeTabId === tabId) {
        activeTabId = remainingTabs[0]?.id ?? ''
      }
      ensureTabExists()
      persistTabs()
    },
    async sendTurn(message: string) {
      const trimmed = message.trim()
      const projectId = input.getProjectId()
      const activeTab = getActiveTab()
      if (
        !trimmed ||
        !projectId ||
        !providerId ||
        activeTab == null ||
        activeTab.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(activeTab.entries)
      ) {
        return
      }
      const currentOperationId = beginProjectConversationOperation(
        activeTab,
        activeTab.conversationId ? 'submitting_turn' : 'creating_conversation',
      )

      try {
        if (!activeTab.conversationId) {
          const createPayload = await createProjectConversation({ providerId, projectId })
          if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
            return
          }
          activeTab.conversationId = createPayload.conversation.id
          activeTab.restored = false
          upsertConversation(createPayload.conversation)
          persistTabs()
          activeTab.phase = 'connecting_stream'
          connectTabStream(activeTab, activeTab.conversationId)
        } else if (!activeTab.abortController) {
          activeTab.phase = 'connecting_stream'
          connectTabStream(activeTab, activeTab.conversationId)
        }

        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
          return
        }

        appendProjectConversationText(activeTab, 'user', trimmed)
        activeTab.activeAssistantEntryId = ''
        activeTab.restored = false
        activeTab.phase = 'submitting_turn'
        await startProjectConversationTurn(activeTab.conversationId, trimmed)
        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
          return
        }
        touchConversation(activeTab.conversationId)
        if (projectConversationHasPendingInterrupt(activeTab.entries)) {
          activeTab.phase = 'awaiting_interrupt'
        } else if (activeTab.phase === 'submitting_turn') {
          activeTab.phase = 'awaiting_reply'
        }
        persistTabs()
      } catch (caughtError) {
        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
          return
        }
        activeTab.phase = 'idle'
        input.onError?.(
          caughtError instanceof Error ? caughtError.message : 'Failed to send project message.',
        )
      }
    },
    async resetConversation() {
      const activeTab = getActiveTab()
      if (!activeTab) {
        return
      }

      const currentOperationId = beginProjectConversationOperation(activeTab, 'resetting')
      const activeConversationId = activeTab.conversationId
      invalidateProjectConversationStream(activeTab)
      if (activeConversationId) {
        await resetProjectConversationRuntime(activeConversationId)
      }
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
        return
      }

      activeTab.conversationId = ''
      activeTab.entries = []
      activeTab.activeAssistantEntryId = ''
      activeTab.phase = 'idle'
      activeTab.restored = false
      persistTabs()
    },
    async confirmActionProposal(entryId: string) {
      const activeTab = getActiveTab()
      if (!activeTab?.conversationId) {
        return
      }
      await confirmProjectConversationActionProposal({
        conversationId: activeTab.conversationId,
        entryId,
        entries: activeTab.entries,
        setEntries: (nextEntries) => {
          activeTab.entries = nextEntries
        },
        onError: input.onError,
      })
    },
    async respondInterrupt(inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) {
      const activeTab = getActiveTab()
      if (!activeTab?.conversationId || activeTab.phase !== 'awaiting_interrupt') {
        return
      }
      activeTab.phase = 'awaiting_reply'
      try {
        await respondProjectConversationInterrupt(
          activeTab.conversationId,
          inputValue.interruptId,
          {
            decision: inputValue.decision,
            answer: inputValue.answer,
          },
        )
      } catch (caughtError) {
        activeTab.phase = 'awaiting_interrupt'
        input.onError?.(
          caughtError instanceof Error ? caughtError.message : 'Failed to answer interrupt.',
        )
      }
    },
    dispose() {
      disposeAllTabs()
    },
  }
}
