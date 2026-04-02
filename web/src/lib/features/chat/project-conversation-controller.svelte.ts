import { type ProjectConversation } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectAIFocus } from './project-ai-focus'
import { projectConversationHasPendingInterrupt } from './project-conversation-controller-helpers'
import { createProjectConversationControllerOperations } from './project-conversation-controller-operations'
import {
  canQueueProjectConversationTurn,
  createProjectConversationTabState,
  ensureProjectConversationTabSelection,
  getActiveProjectConversationTab,
  persistProjectConversationTabs,
  summarizeProjectConversationTab,
  isProjectConversationTabPending,
  type CreateProjectConversationControllerInput,
  type ProjectConversationTabState,
} from './project-conversation-controller-state'
import {
  listEphemeralChatProviders,
  pickDefaultEphemeralChatProvider,
  shouldKeepEphemeralChatProvider,
} from './provider-options'

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

  function newTabState(restored = false): ProjectConversationTabState {
    nextTabID += 1
    return createProjectConversationTabState(nextTabID, restored)
  }

  function getActiveTab() {
    return getActiveProjectConversationTab(tabs, activeTabId)
  }

  function ensureTabSelection(preferredTabId = '') {
    activeTabId = ensureProjectConversationTabSelection(tabs, activeTabId, preferredTabId)
  }

  function canQueueOnTab(tab: ProjectConversationTabState | null) {
    return canQueueProjectConversationTurn({
      projectId: input.getProjectId(),
      providerId,
      tab,
    })
  }

  function nextQueuedTurn(turn: { message: string; focus: ProjectAIFocus | null }) {
    nextQueuedTurnID += 1
    return {
      id: `queued-turn-${nextQueuedTurnID}`,
      createdAt: new Date().toISOString(),
      ...turn,
    }
  }

  function persistTabs() {
    persistProjectConversationTabs({
      projectId: input.getProjectId(),
      providerId,
      tabs,
      activeConversationId: getActiveTab()?.conversationId ?? '',
    })
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

  const operations = createProjectConversationControllerOperations({
    controllerInput: input,
    getProviderId: () => providerId,
    setProviderId: (value) => (providerId = value),
    getConversations: () => conversations,
    setConversations: (value) => (conversations = value),
    getTabs: () => tabs,
    setTabs: (value) => (tabs = value),
    getActiveTabId: () => activeTabId,
    setActiveTabId: (value) => (activeTabId = value),
    newTabState,
    getActiveTab,
    ensureTabExists,
    ensureTabSelection,
    persistTabs,
  })

  ensureTabExists()

  return {
    get providers() {
      return providers
    },
    get conversations() {
      return conversations
    },
    get tabs() {
      return tabs.map((tab) => summarizeProjectConversationTab(tab))
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
      return activeTab ? isProjectConversationTabPending(activeTab) : false
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
      await operations.restore()
    },
    async selectProvider(nextProviderId: string) {
      await operations.selectProvider(nextProviderId)
    },
    createTab() {
      operations.createTab()
    },
    async openConversation(nextConversationId: string) {
      await operations.openConversationInTab(nextConversationId)
    },
    async selectConversation(nextConversationId: string) {
      if (!nextConversationId) {
        return
      }
      await operations.openConversationInTab(nextConversationId)
    },
    selectTab(nextTabId: string) {
      operations.selectTab(nextTabId)
    },
    closeTab(tabId: string) {
      operations.closeTab(tabId)
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

      const sent = await operations.sendTurnInTab(activeTab, nextQueued.message, nextQueued.focus)
      if (!sent) {
        return false
      }

      activeTab.queuedTurns = activeTab.queuedTurns.filter((turn) => turn.id !== nextQueued.id)
      return true
    },
    async sendTurn(message: string, focus?: ProjectAIFocus | null) {
      await operations.sendTurnInTab(getActiveTab(), message, focus)
    },
    async resetConversation() {
      await operations.resetConversation()
    },
    async confirmActionProposal(entryId: string) {
      await operations.confirmActionProposal(entryId)
    },
    cancelActionProposal(entryId: string) {
      operations.cancelActionProposal(entryId)
    },
    async respondInterrupt(inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) {
      await operations.respondInterrupt(inputValue)
    },
    dispose() {
      operations.dispose()
    },
  }
}
