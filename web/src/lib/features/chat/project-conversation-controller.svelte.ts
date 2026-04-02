import {
  createProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  type ProjectConversation,
  type ProjectConversationSessionPayload,
  type ProjectConversationStreamEvent,
  type ProjectConversationTurnRequest,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectAIFocus } from './project-ai-focus'
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

type QueuedProjectTurn = {
  id: string
  message: string
  focus: ProjectAIFocus | null
  createdAt: string
}

type ProjectConversationTabState = ProjectConversationControllerState & {
  id: string
  restored: boolean
  draft: string
  queuedTurns: QueuedProjectTurn[]
}

type ProjectConversationTabSummary = {
  id: string
  conversationId: string
  phase: ProjectConversationPhase
  pending: boolean
  hasPendingInterrupt: boolean
  restored: boolean
  draft: string
  queuedTurns: QueuedProjectTurn[]
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
  let nextQueuedTurnID = 0
  let restoreOperationID = 0

  function newTabState(restored = false): ProjectConversationTabState {
    nextTabID += 1
    return {
      id: `tab-${nextTabID}`,
      restored,
      draft: '',
      queuedTurns: [],
      phase: 'idle',
      conversationId: '',
      entries: [],
      workspaceDiff: null,
      workspaceDiffLoading: false,
      workspaceDiffError: '',
      workspaceDiffRequestId: 0,
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
      draft: tab.draft,
      queuedTurns: tab.queuedTurns,
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

  function canQueueOnTab(tab: ProjectConversationTabState | null) {
    return (
      !!input.getProjectId() &&
      !!providerId &&
      tab != null &&
      (isTabPending(tab) || (tab.phase === 'idle' && tab.queuedTurns.length > 0)) &&
      !projectConversationHasPendingInterrupt(tab.entries)
    )
  }

  function nextQueuedTurn(turn: Omit<QueuedProjectTurn, 'id' | 'createdAt'>): QueuedProjectTurn {
    nextQueuedTurnID += 1
    return {
      id: `queued-turn-${nextQueuedTurnID}`,
      createdAt: new Date().toISOString(),
      ...turn,
    }
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
      if (
        currentRequestId !== tab.workspaceDiffRequestId ||
        tab.conversationId !== conversationId
      ) {
        return
      }
      tab.workspaceDiff = payload.workspaceDiff
    } catch (caughtError) {
      if (
        currentRequestId !== tab.workspaceDiffRequestId ||
        tab.conversationId !== conversationId
      ) {
        return
      }
      tab.workspaceDiff = null
      tab.workspaceDiffError =
        caughtError instanceof Error
          ? caughtError.message
          : 'Failed to load Project AI workspace changes.'
    } finally {
      if (
        currentRequestId === tab.workspaceDiffRequestId &&
        tab.conversationId === conversationId
      ) {
        tab.workspaceDiffLoading = false
      }
    }
  }

  function handleTabStreamEvent(
    tab: ProjectConversationTabState,
    event: ProjectConversationStreamEvent,
  ) {
    if (event.kind === 'session') {
      applySessionPayload(tab, event.payload)
    }
    if ((event.kind === 'session' || event.kind === 'turn_done') && tab.conversationId) {
      void refreshTabWorkspaceDiff(tab, tab.conversationId)
    }
  }

  function touchConversation(conversationId: string) {
    if (!conversationId) {
      return
    }
    const now = new Date().toISOString()
    conversations = sortProjectConversations(
      conversations.map((conversation) =>
        conversation.id === conversationId
          ? { ...conversation, lastActivityAt: now }
          : conversation,
      ),
    )
  }

  function upsertConversation(conversation: ProjectConversation) {
    conversations = sortProjectConversations([
      conversation,
      ...conversations.filter((current) => current.id !== conversation.id),
    ])
  }

  function applySessionPayload(
    tab: ProjectConversationTabState,
    payload: ProjectConversationSessionPayload,
  ) {
    const existing = conversations.find(
      (conversation) => conversation.id === payload.conversationId,
    )
    const now = new Date().toISOString()

    upsertConversation({
      id: payload.conversationId,
      projectId: existing?.projectId ?? input.getProjectId(),
      userId: existing?.userId ?? '',
      source: 'project_sidebar',
      providerId: existing?.providerId ?? providerId,
      providerAnchorKind: payload.providerAnchorKind ?? existing?.providerAnchorKind,
      providerAnchorId: payload.providerAnchorId ?? existing?.providerAnchorId,
      providerTurnId: payload.providerTurnId ?? existing?.providerTurnId,
      providerTurnSupported: payload.providerTurnSupported ?? existing?.providerTurnSupported,
      providerStatus: payload.providerStatus ?? existing?.providerStatus,
      providerActiveFlags:
        payload.providerActiveFlags.length > 0
          ? [...payload.providerActiveFlags]
          : [...(existing?.providerActiveFlags ?? [])],
      status: existing?.status ?? '',
      rollingSummary: existing?.rollingSummary ?? '',
      lastActivityAt: now,
      createdAt: existing?.createdAt ?? now,
      updatedAt: now,
    })

    if (!tab.conversationId && payload.conversationId) {
      tab.conversationId = payload.conversationId
    }
  }

  function sortProjectConversations(items: ProjectConversation[]) {
    return [...items].sort((left, right) => {
      const comparison = conversationActivitySortKey(right).localeCompare(
        conversationActivitySortKey(left),
      )
      if (comparison !== 0) {
        return comparison
      }
      return right.id.localeCompare(left.id)
    })
  }

  function conversationActivitySortKey(conversation: ProjectConversation) {
    return conversation.lastActivityAt || conversation.updatedAt || conversation.createdAt || ''
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
      await refreshTabWorkspaceDiff(tab, conversationId)
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
      activeTab.phase === 'idle' &&
      activeTab.draft.trim().length === 0
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

  async function sendTurnInTab(
    activeTab: ProjectConversationTabState | null,
    message: string,
    focus?: ProjectAIFocus | null,
  ) {
    const trimmed = message.trim()
    const projectId = input.getProjectId()
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
        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
          return false
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
        return false
      }

      appendProjectConversationText(activeTab, 'user', trimmed)
      activeTab.activeAssistantEntryId = ''
      activeTab.restored = false
      activeTab.phase = 'submitting_turn'
      const request: ProjectConversationTurnRequest = {
        message: trimmed,
        focus: focus ?? undefined,
      }
      await startProjectConversationTurn(activeTab.conversationId, request)
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
        return false
      }
      touchConversation(activeTab.conversationId)
      if (projectConversationHasPendingInterrupt(activeTab.entries)) {
        activeTab.phase = 'awaiting_interrupt'
      } else if (activeTab.phase === 'submitting_turn') {
        activeTab.phase = 'awaiting_reply'
      }
      persistTabs()
      return true
    } catch (caughtError) {
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
        return false
      }
      activeTab.phase = 'idle'
      input.onError?.(
        caughtError instanceof Error ? caughtError.message : 'Failed to send project message.',
      )
      return false
    }
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
    get draft() {
      return getActiveTab()?.draft ?? ''
    },
    get queuedTurns() {
      return getActiveTab()?.queuedTurns ?? []
    },
    get workspaceDiff() {
      return getActiveTab()?.workspaceDiff ?? null
    },
    get workspaceDiffLoading() {
      return getActiveTab()?.workspaceDiffLoading ?? false
    },
    get workspaceDiffError() {
      return getActiveTab()?.workspaceDiffError ?? ''
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
        projectConversationHasPendingInterrupt(activeTab.entries)
      )
    },
    get sendDisabled() {
      const activeTab = getActiveTab()
      return (
        !input.getProjectId() ||
        !providerId ||
        activeTab == null ||
        activeTab.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(activeTab.entries)
      )
    },
    get canQueueTurn() {
      return canQueueOnTab(getActiveTab())
    },
    get providerSelectionDisabled() {
      return tabs.some((tab) => tab.phase !== 'idle')
    },
    setDraft(value: string) {
      const activeTab = getActiveTab()
      if (!activeTab) {
        return
      }
      activeTab.draft = value
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

        conversations = sortProjectConversations(listPayload.conversations)
        const availableConversationIDs = new Set(
          listPayload.conversations.map((conversation) => conversation.id),
        )
        const persisted = readProjectConversationTabs(projectId, providerId)
        const restoredConversationIDs = persisted.conversationIds.filter((conversationId) =>
          availableConversationIDs.has(conversationId),
        )
        const fallbackConversationID =
          restoredConversationIDs.length === 0 ? (listPayload.conversations[0]?.id ?? '') : ''
        const preferredConversationID =
          (persisted.activeConversationId &&
          availableConversationIDs.has(persisted.activeConversationId)
            ? persisted.activeConversationId
            : restoredConversationIDs[0]) || fallbackConversationID
        const initialConversationIDs = preferredConversationID ? [preferredConversationID] : []

        if (initialConversationIDs.length === 0) {
          const activeTab = getActiveTab()
          const reusableBlank =
            activeTab && !activeTab.conversationId && activeTab.entries.length === 0
              ? activeTab
              : (tabs.find((tab) => !tab.conversationId && tab.entries.length === 0) ?? null)
          const blankTab = reusableBlank ?? newTabState(false)
          blankTab.restored = false
          blankTab.phase = 'idle'
          tabs = [blankTab]
          activeTabId = blankTab.id
          persistTabs()
          return
        }

        tabs = initialConversationIDs.map(() => newTabState(true))

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

        ensureTabSelection(tabs[0].id)
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
        (tab) =>
          !tab.conversationId &&
          tab.entries.length === 0 &&
          tab.phase === 'idle' &&
          tab.draft.trim().length === 0,
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
    enqueueTurn(message: string, focus?: ProjectAIFocus | null) {
      const activeTab = getActiveTab()
      const trimmed = message.trim()
      if (!trimmed || !canQueueOnTab(activeTab)) {
        return false
      }

      activeTab.queuedTurns = [
        ...activeTab.queuedTurns,
        nextQueuedTurn({
          message: trimmed,
          focus: focus ?? null,
        }),
      ]
      return true
    },
    cancelQueuedTurn(queueTurnId: string) {
      const activeTab = getActiveTab()
      if (!activeTab) {
        return false
      }

      const nextQueuedTurns = activeTab.queuedTurns.filter((turn) => turn.id !== queueTurnId)
      if (nextQueuedTurns.length === activeTab.queuedTurns.length) {
        return false
      }

      activeTab.queuedTurns = nextQueuedTurns
      return true
    },
    async sendNextQueuedTurn() {
      const activeTab = getActiveTab()
      const nextQueued = activeTab?.queuedTurns[0]
      if (!activeTab || !nextQueued) {
        return false
      }

      const sent = await sendTurnInTab(activeTab, nextQueued.message, nextQueued.focus)
      if (!sent) {
        return false
      }

      activeTab.queuedTurns = activeTab.queuedTurns.filter((turn) => turn.id !== nextQueued.id)
      return true
    },
    async sendTurn(message: string, focus?: ProjectAIFocus | null) {
      await sendTurnInTab(getActiveTab(), message, focus)
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
        if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
          return
        }
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
      if (!isCurrentProjectConversationOperation(activeTab, currentOperationId)) {
        return
      }
      activeTab.queuedTurns = []
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
    cancelActionProposal(entryId: string) {
      const activeTab = getActiveTab()
      if (!activeTab) {
        return
      }
      activeTab.entries = activeTab.entries.map((entry) =>
        entry.kind === 'action_proposal' && entry.id === entryId && entry.status === 'pending'
          ? { ...entry, status: 'cancelled' }
          : entry,
      )
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
