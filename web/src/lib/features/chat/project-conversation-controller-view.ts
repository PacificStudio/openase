import type { ProjectConversation } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectAIFocus } from './project-ai-focus'
import { projectConversationHasPendingInterrupt } from './project-conversation-controller-helpers'
import {
  isProjectConversationTabPending,
  type CreateProjectConversationControllerInput,
  type ProjectConversationControllerSnapshot,
  type ProjectConversationTabState,
  type QueuedProjectTurn,
} from './project-conversation-controller-state'
import {
  listEphemeralChatProviders,
  pickDefaultEphemeralChatProvider,
  shouldKeepEphemeralChatProvider,
} from './provider-options'

type BuildProjectConversationControllerSnapshotInput = {
  controllerInput: CreateProjectConversationControllerInput
  providers: AgentProvider[]
  conversations: ProjectConversation[]
  tabs: ProjectConversationTabState[]
  activeTabId: string
  activeTab: ProjectConversationTabState | null
  providerId: string
  canQueueOnTab: (tab: ProjectConversationTabState | null) => boolean
}

export function buildProjectConversationControllerSnapshot(
  input: BuildProjectConversationControllerSnapshotInput,
): ProjectConversationControllerSnapshot {
  const activeTab = input.activeTab
  return {
    providers: input.providers,
    conversations: input.conversations,
    tabs: input.tabs,
    activeTabId: input.activeTabId,
    providerId: input.providerId,
    providerSelectionDisabled: activeTab ? isProjectConversationTabPending(activeTab) : false,
    entries: activeTab?.entries ?? [],
    conversationId: activeTab?.conversationId ?? '',
    workspaceDiff: activeTab?.workspaceDiff ?? null,
    workspaceDiffLoading: activeTab?.workspaceDiffLoading ?? false,
    workspaceDiffError: activeTab?.workspaceDiffError ?? '',
    pending: activeTab ? isProjectConversationTabPending(activeTab) : false,
    queuedTurns: activeTab?.queuedTurns ?? [],
    hasPendingInterrupt: activeTab
      ? projectConversationHasPendingInterrupt(activeTab.entries)
      : false,
    draft: activeTab?.draft ?? '',
    inputDisabled:
      !activeTab?.projectId ||
      !activeTab?.providerId ||
      activeTab == null ||
      projectConversationHasPendingInterrupt(activeTab.entries),
    sendDisabled:
      !activeTab?.projectId ||
      !activeTab?.providerId ||
      activeTab == null ||
      activeTab.phase !== 'idle' ||
      projectConversationHasPendingInterrupt(activeTab.entries),
    canQueueTurn: input.canQueueOnTab(activeTab),
    phase: activeTab?.phase ?? 'idle',
  }
}

export function queueProjectConversationControllerSnapshotNotification(input: {
  queued: boolean
  setQueued: (value: boolean) => void
  onStateChange?: (snapshot: ProjectConversationControllerSnapshot) => void
  snapshot: () => ProjectConversationControllerSnapshot
}) {
  if (input.queued) {
    return
  }

  input.setQueued(true)
  queueMicrotask(() => {
    input.setQueued(false)
    input.onStateChange?.(input.snapshot())
  })
}

type SyncProjectConversationControllerProvidersInput = {
  providers: AgentProvider[]
  defaultProviderId: string | null | undefined
  conversations: ProjectConversation[]
  tabs: ProjectConversationTabState[]
}

export function syncProjectConversationControllerProviders(
  input: SyncProjectConversationControllerProvidersInput,
) {
  const nextProviders = listEphemeralChatProviders(input.providers)
  const preferredProviderId = pickDefaultEphemeralChatProvider(
    nextProviders,
    input.defaultProviderId,
  )

  for (const tab of input.tabs) {
    if (tab.conversationId) {
      if (!tab.providerId) {
        tab.providerId =
          input.conversations.find((conversation) => conversation.id === tab.conversationId)
            ?.providerId ?? preferredProviderId
      }
      continue
    }

    if (!shouldKeepEphemeralChatProvider(nextProviders, tab.providerId)) {
      tab.providerId = preferredProviderId
    }
  }

  return {
    providers: nextProviders,
    preferredProviderId,
  }
}

export function createQueuedProjectConversationTurn(
  nextQueuedTurnId: number,
  turn: { message: string; focus: ProjectAIFocus | null },
): QueuedProjectTurn {
  return {
    id: `queued-turn-${nextQueuedTurnId}`,
    createdAt: new Date().toISOString(),
    ...turn,
  }
}

export function ensureProjectConversationTabExists(input: {
  tabs: ProjectConversationTabState[]
  preferredProviderId: string
  newTabState: (providerId?: string, restored?: boolean) => ProjectConversationTabState
  ensureTabSelection: (preferredTabId?: string) => void
  setTabs: (tabs: ProjectConversationTabState[]) => void
  setActiveTabId: (value: string) => void
}) {
  if (input.tabs.length > 0) {
    input.ensureTabSelection()
    return
  }

  const tab = input.newTabState(input.preferredProviderId, false)
  input.setTabs([tab])
  input.setActiveTabId(tab.id)
}
