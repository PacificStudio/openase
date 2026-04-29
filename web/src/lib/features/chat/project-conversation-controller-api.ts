import type { ProjectConversation } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectConversationTabState } from './project-conversation-controller-state'

type ProjectConversationControllerApiInput<TActions extends object> = {
  snapshot: () => unknown
  getProviders: () => AgentProvider[]
  getConversations: () => ProjectConversation[]
  getTabs: () => ProjectConversationTabState[]
  getActiveTabId: () => string
  getActiveTab: () => ProjectConversationTabState | null
  getProviderId: () => string
  getPhase: () => ProjectConversationTabState['phase'] | 'idle'
  getSelectedProvider: () => AgentProvider | null
  getBusy: () => boolean
  getPending: () => boolean
  getConversationId: () => string
  getEntries: () => ProjectConversationTabState['entries']
  getDraft: () => string
  getQueuedTurns: () => ProjectConversationTabState['queuedTurns']
  getWorkspaceDiff: () => ProjectConversationTabState['workspaceDiff']
  getWorkspaceDiffLoading: () => boolean
  getWorkspaceDiffError: () => string
  getHasPendingInterrupt: () => boolean
  getInputDisabled: () => boolean
  getSendDisabled: () => boolean
  getCanQueueTurn: () => boolean
  getProviderSelectionDisabled: () => boolean
  actions: TActions
}

export function createProjectConversationControllerApi<TActions extends object>(
  input: ProjectConversationControllerApiInput<TActions>,
) {
  return {
    snapshot: input.snapshot,
    get providers() {
      return input.getProviders()
    },
    get conversations() {
      return input.getConversations()
    },
    get tabs() {
      return input.getTabs()
    },
    get activeTabId() {
      return input.getActiveTabId()
    },
    get activeTab() {
      return input.getActiveTab()
    },
    get providerId() {
      return input.getProviderId()
    },
    get phase() {
      return input.getPhase()
    },
    get selectedProvider() {
      return input.getSelectedProvider()
    },
    get busy() {
      return input.getBusy()
    },
    get pending() {
      return input.getPending()
    },
    get conversationId() {
      return input.getConversationId()
    },
    get entries() {
      return input.getEntries()
    },
    get draft() {
      return input.getDraft()
    },
    get queuedTurns() {
      return input.getQueuedTurns()
    },
    get workspaceDiff() {
      return input.getWorkspaceDiff()
    },
    get workspaceDiffLoading() {
      return input.getWorkspaceDiffLoading()
    },
    get workspaceDiffError() {
      return input.getWorkspaceDiffError()
    },
    get hasPendingInterrupt() {
      return input.getHasPendingInterrupt()
    },
    get inputDisabled() {
      return input.getInputDisabled()
    },
    get sendDisabled() {
      return input.getSendDisabled()
    },
    get canQueueTurn() {
      return input.getCanQueueTurn()
    },
    get providerSelectionDisabled() {
      return input.getProviderSelectionDisabled()
    },
    ...input.actions,
  }
}
