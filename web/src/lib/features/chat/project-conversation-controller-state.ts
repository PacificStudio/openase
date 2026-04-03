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
  draft: string
  queuedTurns: QueuedProjectTurn[]
}

export type ProjectConversationTabSummary = {
  id: string
  providerId: string
  conversationId: string
  phase: ProjectConversationPhase
  pending: boolean
  hasPendingInterrupt: boolean
  restored: boolean
  draft: string
  queuedTurns: QueuedProjectTurn[]
  entries: ProjectConversationTranscriptEntry[]
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

export function summarizeProjectConversationTab(
  tab: ProjectConversationTabState,
): ProjectConversationTabSummary {
  return {
    id: tab.id,
    providerId: tab.providerId,
    conversationId: tab.conversationId,
    phase: tab.phase,
    pending: isProjectConversationTabPending(tab),
    hasPendingInterrupt: projectConversationHasPendingInterrupt(tab.entries),
    restored: tab.restored,
    draft: tab.draft,
    queuedTurns: tab.queuedTurns,
    entries: tab.entries,
  }
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
