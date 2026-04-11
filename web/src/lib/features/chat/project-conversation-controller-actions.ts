import type { ProjectConversation } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ProjectAIFocus } from './project-ai-focus'
import type {
  ProjectConversationTabState,
  QueuedProjectTurn,
} from './project-conversation-controller-state'
import { syncProjectConversationControllerProviders } from './project-conversation-controller-view'

export function enqueueProjectConversationTurn(input: {
  tab: ProjectConversationTabState | null
  message: string
  focus?: ProjectAIFocus | null
  canQueueOnTab: (tab: ProjectConversationTabState | null) => boolean
  createQueuedTurn: (turn: { message: string; focus: ProjectAIFocus | null }) => QueuedProjectTurn
}) {
  const trimmed = input.message.trim()
  if (!trimmed || !input.canQueueOnTab(input.tab) || !input.tab) {
    return false
  }

  input.tab.queuedTurns = [
    ...input.tab.queuedTurns,
    input.createQueuedTurn({
      message: trimmed,
      focus: input.focus ?? null,
    }),
  ]
  return true
}

export function cancelQueuedProjectConversationTurn(
  tab: ProjectConversationTabState | null,
  queueTurnId: string,
) {
  if (!tab) {
    return false
  }

  const nextQueuedTurns = tab.queuedTurns.filter((turn) => turn.id !== queueTurnId)
  if (nextQueuedTurns.length === tab.queuedTurns.length) {
    return false
  }

  tab.queuedTurns = nextQueuedTurns
  return true
}

export async function sendNextQueuedProjectConversationTurn(input: {
  tab: ProjectConversationTabState | null
  sendTurnInTab: (
    tab: ProjectConversationTabState,
    message: string,
    focus: ProjectAIFocus | null,
  ) => Promise<boolean>
}) {
  const tab = input.tab
  const nextQueued = tab?.queuedTurns[0]
  if (!tab || !nextQueued) {
    return false
  }

  tab.queuedTurns = tab.queuedTurns.filter((turn) => turn.id !== nextQueued.id)

  const sent = await input.sendTurnInTab(tab, nextQueued.message, nextQueued.focus)
  if (!sent) {
    if (!tab.queuedTurns.some((turn) => turn.id === nextQueued.id)) {
      tab.queuedTurns = [nextQueued, ...tab.queuedTurns]
    }
    return false
  }

  return true
}

type ProjectConversationControllerActionsInput = {
  getConversations: () => ProjectConversation[]
  getTabs: () => ProjectConversationTabState[]
  getActiveTab: () => ProjectConversationTabState | null
  setProviders: (providers: AgentProvider[]) => void
  setPreferredProviderId: (providerId: string) => void
  canQueueOnTab: (tab: ProjectConversationTabState | null) => boolean
  nextQueuedTurn: (turn: { message: string; focus: ProjectAIFocus | null }) => QueuedProjectTurn
  ensureTabExists: () => void
  touch: () => void
  operations: {
    restore: () => Promise<void>
    selectProvider: (nextProviderId: string) => Promise<void>
    createTab: () => void
    openConversationInTab: (nextConversationId: string) => Promise<void>
    selectTab: (nextTabId: string) => void
    closeTab: (tabId: string) => void
    sendTurnInTab: (
      tab: ProjectConversationTabState | null,
      message: string,
      focus?: ProjectAIFocus | null,
    ) => Promise<boolean>
    refreshWorkspaceDiff: () => Promise<void>
    resetConversation: () => Promise<void>
    stopTurn: () => Promise<void>
    respondInterrupt: (inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void>
    deleteConversation: (conversationId: string, options?: { force?: boolean }) => Promise<boolean>
    dispose: () => void
  }
}

export function createProjectConversationControllerActions(
  input: ProjectConversationControllerActionsInput,
) {
  return {
    setDraft(value: string) {
      const activeTab = input.getActiveTab()
      if (!activeTab) {
        return
      }
      activeTab.draft = value
      input.touch()
    },
    syncProviders(providers: AgentProvider[], defaultProviderId: string | null | undefined) {
      const nextState = syncProjectConversationControllerProviders({
        providers,
        defaultProviderId,
        conversations: input.getConversations(),
        tabs: input.getTabs(),
      })
      input.setProviders(nextState.providers)
      input.setPreferredProviderId(nextState.preferredProviderId)
      input.ensureTabExists()
      input.touch()
    },
    async restore() {
      await input.operations.restore()
    },
    async selectProvider(nextProviderId: string) {
      await input.operations.selectProvider(nextProviderId)
    },
    createTab() {
      input.operations.createTab()
    },
    async openConversation(nextConversationId: string) {
      await input.operations.openConversationInTab(nextConversationId)
    },
    async selectConversation(nextConversationId: string) {
      if (!nextConversationId) {
        return
      }
      await input.operations.openConversationInTab(nextConversationId)
    },
    selectTab(nextTabId: string) {
      input.operations.selectTab(nextTabId)
    },
    closeTab(tabId: string) {
      input.operations.closeTab(tabId)
    },
    enqueueTurn(message: string, focus?: ProjectAIFocus | null) {
      const queued = enqueueProjectConversationTurn({
        tab: input.getActiveTab(),
        message,
        focus,
        canQueueOnTab: input.canQueueOnTab,
        createQueuedTurn: input.nextQueuedTurn,
      })
      if (queued) input.touch()
      return queued
    },
    cancelQueuedTurn(queueTurnId: string) {
      const cancelled = cancelQueuedProjectConversationTurn(input.getActiveTab(), queueTurnId)
      if (cancelled) input.touch()
      return cancelled
    },
    async sendNextQueuedTurn() {
      const sent = await sendNextQueuedProjectConversationTurn({
        tab: input.getActiveTab(),
        sendTurnInTab: input.operations.sendTurnInTab,
      })
      if (sent) input.touch()
      return sent
    },
    async sendTurn(message: string, focus?: ProjectAIFocus | null) {
      await input.operations.sendTurnInTab(input.getActiveTab(), message, focus)
    },
    async refreshWorkspaceDiff() {
      await input.operations.refreshWorkspaceDiff()
    },
    async resetConversation() {
      await input.operations.resetConversation()
    },
    async stopTurn() {
      await input.operations.stopTurn()
    },
    async respondInterrupt(inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) {
      await input.operations.respondInterrupt(inputValue)
    },
    async deleteConversation(conversationId: string, options?: { force?: boolean }) {
      return input.operations.deleteConversation(conversationId, options)
    },
    dispose() {
      input.operations.dispose()
    },
  }
}
