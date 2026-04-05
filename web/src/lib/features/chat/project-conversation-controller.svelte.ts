import { type ProjectConversation } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectAIFocus } from './project-ai-focus'
import { projectConversationHasPendingInterrupt } from './project-conversation-controller-helpers'
import { createProjectConversationControllerOperations } from './project-conversation-controller-operations'
import { createProjectConversationControllerActions } from './project-conversation-controller-actions'
import {
  canQueueProjectConversationTurn,
  createProjectConversationTabState,
  ensureProjectConversationTabSelection,
  getActiveProjectConversationTab,
  persistProjectConversationTabs,
  isProjectConversationTabPending,
  type CreateProjectConversationControllerInput,
  type ProjectConversationTabState,
} from './project-conversation-controller-state'
import {
  buildProjectConversationControllerSnapshot,
  createQueuedProjectConversationTurn,
  ensureProjectConversationTabExists,
  queueProjectConversationControllerSnapshotNotification,
} from './project-conversation-controller-view'

export function createProjectConversationController(
  input: CreateProjectConversationControllerInput,
) {
  let providers = $state<AgentProvider[]>([])
  let preferredProviderId = $state('')
  let conversations = $state<ProjectConversation[]>([])
  let tabs = $state<ProjectConversationTabState[]>([])
  let activeTabId = $state('')
  let revision = $state(0)
  let nextTabID = 0
  let nextQueuedTurnID = 0
  let snapshotNotificationQueued = false

  function touch() {
    revision += 1
    queueProjectConversationControllerSnapshotNotification({
      queued: snapshotNotificationQueued,
      setQueued: (value) => (snapshotNotificationQueued = value),
      onStateChange: input.onStateChange,
      snapshot: () =>
        buildProjectConversationControllerSnapshot({
          controllerInput: input,
          providers,
          conversations,
          tabs,
          activeTabId,
          activeTab: getActiveTab(),
          providerId: getActiveProviderId(),
          canQueueOnTab,
        }),
    })
  }

  function newTabState(
    providerId = '',
    restored = false,
    projectId = '',
    projectName = '',
  ): ProjectConversationTabState {
    nextTabID += 1
    const ctx = input.getProjectContext()
    return createProjectConversationTabState(
      nextTabID,
      providerId,
      restored,
      projectId || ctx.projectId,
      projectName || ctx.projectName,
    )
  }

  function getActiveTab() {
    return getActiveProjectConversationTab(tabs, activeTabId)
  }

  function getActiveProviderId() {
    return getActiveTab()?.providerId ?? preferredProviderId
  }

  function ensureTabSelection(preferredTabId = '') {
    activeTabId = ensureProjectConversationTabSelection(tabs, activeTabId, preferredTabId)
  }

  function canQueueOnTab(tab: ProjectConversationTabState | null) {
    return canQueueProjectConversationTurn({ tab })
  }

  function snapshot() {
    return buildProjectConversationControllerSnapshot({
      controllerInput: input,
      providers,
      conversations,
      tabs,
      activeTabId,
      activeTab: getActiveTab(),
      providerId: getActiveProviderId(),
      canQueueOnTab,
    })
  }

  function nextQueuedTurn(turn: { message: string; focus: ProjectAIFocus | null }) {
    nextQueuedTurnID += 1
    return createQueuedProjectConversationTurn(nextQueuedTurnID, turn)
  }

  function ensureTabExists() {
    ensureProjectConversationTabExists({
      tabs,
      preferredProviderId,
      newTabState,
      ensureTabSelection,
      setTabs: (value) => (tabs = value),
      setActiveTabId: (value) => (activeTabId = value),
    })
  }

  const operations = createProjectConversationControllerOperations({
    controllerInput: input,
    getProviderId: getActiveProviderId,
    getPreferredProviderId: () => preferredProviderId,
    getConversations: () => conversations,
    setConversations: (value) => {
      conversations = value
      touch()
    },
    getTabs: () => tabs,
    setTabs: (value) => {
      tabs = value
      touch()
    },
    getActiveTabId: () => activeTabId,
    setActiveTabId: (value) => (activeTabId = value),
    newTabState,
    getActiveTab,
    ensureTabExists,
    ensureTabSelection,
    persistTabs: () =>
      persistProjectConversationTabs({
        tabs,
        activeTabId,
      }),
    touch,
  })
  const actions = createProjectConversationControllerActions({
    getConversations: () => conversations,
    getTabs: () => tabs,
    getActiveTab,
    setProviders: (value) => (providers = value),
    setPreferredProviderId: (value) => (preferredProviderId = value),
    canQueueOnTab,
    nextQueuedTurn,
    ensureTabExists,
    touch,
    operations,
  })

  ensureTabExists()

  return {
    snapshot,
    get providers() {
      return providers
    },
    get conversations() {
      return conversations
    },
    get tabs() {
      void revision
      return tabs
    },
    get activeTabId() {
      return activeTabId
    },
    get activeTab() {
      void revision
      return getActiveTab()
    },
    get providerId() {
      return getActiveProviderId()
    },
    get phase() {
      void revision
      return getActiveTab()?.phase ?? 'idle'
    },
    get selectedProvider() {
      return providers.find((provider) => provider.id === getActiveProviderId()) ?? null
    },
    get busy() {
      return (getActiveTab()?.phase ?? 'idle') !== 'idle'
    },
    get pending() {
      void revision
      const activeTab = getActiveTab()
      return activeTab ? isProjectConversationTabPending(activeTab) : false
    },
    get conversationId() {
      void revision
      return getActiveTab()?.conversationId ?? ''
    },
    get entries() {
      void revision
      return getActiveTab()?.entries ?? []
    },
    get draft() {
      void revision
      return getActiveTab()?.draft ?? ''
    },
    get queuedTurns() {
      void revision
      return getActiveTab()?.queuedTurns ?? []
    },
    get workspaceDiff() {
      void revision
      return getActiveTab()?.workspaceDiff ?? null
    },
    get workspaceDiffLoading() {
      void revision
      return getActiveTab()?.workspaceDiffLoading ?? false
    },
    get workspaceDiffError() {
      void revision
      return getActiveTab()?.workspaceDiffError ?? ''
    },
    get hasPendingInterrupt() {
      void revision
      const activeTab = getActiveTab()
      return activeTab ? projectConversationHasPendingInterrupt(activeTab.entries) : false
    },
    get inputDisabled() {
      void revision
      const activeTab = getActiveTab()
      return (
        !activeTab?.projectId ||
        !activeTab?.providerId ||
        activeTab == null ||
        projectConversationHasPendingInterrupt(activeTab.entries)
      )
    },
    get sendDisabled() {
      void revision
      const activeTab = getActiveTab()
      return (
        !activeTab?.projectId ||
        !activeTab?.providerId ||
        activeTab == null ||
        activeTab.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(activeTab.entries)
      )
    },
    get canQueueTurn() {
      void revision
      return canQueueOnTab(getActiveTab())
    },
    get providerSelectionDisabled() {
      void revision
      const activeTab = getActiveTab()
      return activeTab ? isProjectConversationTabPending(activeTab) : false
    },
    ...actions,
  }
}
