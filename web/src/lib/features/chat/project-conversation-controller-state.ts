import type { ProjectConversation, ProjectConversationWorkspaceDiff } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import { storeProjectConversationTabs } from './project-conversation-storage'
import {
  projectConversationHasPendingInterrupt,
  invalidateProjectConversationStream,
  type ProjectConversationControllerState,
  type ProjectConversationPhase,
} from './project-conversation-controller-helpers'
import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'
import type { ProjectAIFocus } from './project-ai-focus'

export type CreateProjectConversationControllerInput = {
  getProjectId: () => string
  onError?: (message: string) => void
  onStateChange?: (snapshot: ProjectConversationControllerSnapshot) => void
}

export type QueuedProjectTurn = {
  id: string
  message: string
  focus: ProjectAIFocus | null
  createdAt: string
}

export type ProjectConversationTabState = ProjectConversationControllerState & {
  id: string
  providerId: string
  restored: boolean
  needsHydration: boolean
  unread: boolean
  draft: string
  queuedTurns: QueuedProjectTurn[]
  readonly pending: boolean
  readonly hasPendingInterrupt: boolean
}

export type ProjectConversationControllerSnapshot = {
  providers: AgentProvider[]
  conversations: ProjectConversation[]
  tabs: ProjectConversationTabState[]
  activeTabId: string
  providerId: string
  providerSelectionDisabled: boolean
  entries: ProjectConversationTranscriptEntry[]
  conversationId: string
  workspaceDiff: ProjectConversationWorkspaceDiff | null
  workspaceDiffLoading: boolean
  workspaceDiffError: string
  pending: boolean
  queuedTurns: QueuedProjectTurn[]
  hasPendingInterrupt: boolean
  draft: string
  inputDisabled: boolean
  sendDisabled: boolean
  canQueueTurn: boolean
  phase: ProjectConversationPhase
}

export function createProjectConversationTabState(
  tabNumber: number,
  providerId = '',
  restored = false,
): ProjectConversationTabState {
  return {
    id: `tab-${tabNumber}`,
    providerId,
    restored,
    needsHydration: false,
    unread: false,
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
    get pending() {
      return isProjectConversationTabPending(this as ProjectConversationTabState)
    },
    get hasPendingInterrupt() {
      return projectConversationHasPendingInterrupt(this.entries)
    },
  }
}

export function findProjectConversationTab(tabs: ProjectConversationTabState[], tabId: string) {
  return tabs.find((tab) => tab.id === tabId) ?? null
}

export function getActiveProjectConversationTab(
  tabs: ProjectConversationTabState[],
  activeTabId: string,
) {
  return findProjectConversationTab(tabs, activeTabId) ?? tabs[0] ?? null
}

export function isProjectConversationTabPending(tab: ProjectConversationTabState) {
  return (
    tab.phase === 'creating_conversation' ||
    tab.phase === 'connecting_stream' ||
    tab.phase === 'submitting_turn' ||
    tab.phase === 'awaiting_reply'
  )
}

export function ensureProjectConversationTabSelection(
  tabs: ProjectConversationTabState[],
  activeTabId: string,
  preferredTabId = '',
) {
  if (preferredTabId && findProjectConversationTab(tabs, preferredTabId) != null) {
    return preferredTabId
  }
  if (findProjectConversationTab(tabs, activeTabId) != null) {
    return activeTabId
  }
  return tabs[0]?.id ?? ''
}

export function canQueueProjectConversationTurn(input: {
  projectId: string
  tab: ProjectConversationTabState | null
}) {
  return (
    !!input.projectId &&
    input.tab != null &&
    !!input.tab.providerId &&
    (isProjectConversationTabPending(input.tab) ||
      (input.tab.phase === 'idle' && input.tab.queuedTurns.length > 0)) &&
    !projectConversationHasPendingInterrupt(input.tab.entries)
  )
}

export function persistProjectConversationTabs(input: {
  projectId: string
  tabs: ProjectConversationTabState[]
  activeTabId: string
}) {
  if (!input.projectId) return
  storeProjectConversationTabs(input.projectId, {
    tabs: input.tabs.map((tab) => ({
      conversationId: tab.conversationId.trim(),
      providerId: tab.providerId.trim(),
      draft: tab.draft,
    })),
    activeTabIndex: Math.max(
      0,
      input.tabs.findIndex((tab) => tab.id === input.activeTabId),
    ),
  })
}

export function disposeProjectConversationTabs(tabs: ProjectConversationTabState[]) {
  for (const tab of tabs) {
    invalidateProjectConversationStream(tab)
    tab.phase = 'idle'
  }
}
