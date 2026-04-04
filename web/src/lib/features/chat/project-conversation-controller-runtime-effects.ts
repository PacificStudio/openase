import {
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import { projectConversationHasPendingInterrupt } from './project-conversation-controller-helpers'
import { mapPersistedEntries } from './project-conversation-transcript-state'
import type { ProjectConversationControllerConversations } from './project-conversation-controller-conversations'
import type { ProjectConversationTabState } from './project-conversation-controller-state'

type RuntimeEffectsInput = {
  conversations: ProjectConversationControllerConversations
  isActiveTab: (tab: ProjectConversationTabState) => boolean
  onError?: (message: string) => void
  persistTabs: () => void
  touchTabs: () => void
  connectTabStream: (tab: ProjectConversationTabState, conversationId: string) => void
}

export function projectConversationTabPhaseFromRuntimeState(
  runtimeState: string | undefined,
): ProjectConversationTabState['phase'] {
  switch ((runtimeState ?? '').trim()) {
    case 'executing':
      return 'awaiting_reply'
    case 'interrupted':
      return 'awaiting_interrupt'
    default:
      return 'idle'
  }
}

export async function hydrateTabEntries(
  tab: ProjectConversationTabState,
  conversationId: string,
  onChanged?: () => void,
) {
  const payload = await listProjectConversationEntries(conversationId)
  const mappedEntries = mapPersistedEntries(payload.entries)
  tab.entries = mappedEntries
  tab.entryCounter = mappedEntries.length
  tab.activeAssistantEntryId = ''
  onChanged?.()
}

export async function refreshTabWorkspaceDiff(
  tab: ProjectConversationTabState,
  conversationId: string,
  onChanged?: () => void,
) {
  if (!conversationId) {
    tab.workspaceDiff = null
    tab.workspaceDiffLoading = false
    tab.workspaceDiffError = ''
    onChanged?.()
    return
  }

  tab.workspaceDiffRequestId += 1
  const currentRequestId = tab.workspaceDiffRequestId
  tab.workspaceDiffLoading = true
  tab.workspaceDiffError = ''
  try {
    const payload = await getProjectConversationWorkspaceDiff(conversationId)
    if (currentRequestId !== tab.workspaceDiffRequestId || tab.conversationId !== conversationId)
      return
    tab.workspaceDiff = payload.workspaceDiff
    onChanged?.()
  } catch (caughtError) {
    if (currentRequestId !== tab.workspaceDiffRequestId || tab.conversationId !== conversationId)
      return
    tab.workspaceDiff = null
    tab.workspaceDiffError =
      caughtError instanceof Error
        ? caughtError.message
        : 'Failed to load Project AI workspace changes.'
    onChanged?.()
  } finally {
    if (currentRequestId === tab.workspaceDiffRequestId && tab.conversationId === conversationId) {
      tab.workspaceDiffLoading = false
      onChanged?.()
    }
  }
}

export function handleTabStreamEvent(
  input: RuntimeEffectsInput,
  tab: ProjectConversationTabState,
  event: ProjectConversationStreamEvent,
) {
  const isActive = input.isActiveTab(tab)
  if (event.kind === 'session') {
    tab.conversationId = input.conversations.applySessionPayload(tab.conversationId, event.payload)
    tab.phase = projectConversationTabPhaseFromRuntimeState(event.payload.runtimeState)
    if (isActive) {
      tab.needsHydration = false
      tab.unread = false
    }
  }
  if (isActive && (event.kind === 'session' || event.kind === 'turn_done') && tab.conversationId) {
    void refreshTabWorkspaceDiff(tab, tab.conversationId, input.touchTabs)
  }

  if (isActive) {
    tab.needsHydration = false
    tab.unread = false
    return
  }

  if (tab.conversationId) {
    input.conversations.touchConversation(tab.conversationId)
  }

  switch (event.kind) {
    case 'session':
      return
    case 'interrupt_requested':
      tab.phase = 'awaiting_interrupt'
      break
    case 'interrupt_resolved':
      tab.phase = 'awaiting_reply'
      break
    case 'turn_done':
    case 'error':
      tab.phase = 'idle'
      break
    default:
      if (tab.phase !== 'awaiting_interrupt') {
        tab.phase = 'awaiting_reply'
      }
      break
  }

  tab.needsHydration = true
  tab.unread = true
}

export async function reconcileTabAfterReconnect(
  input: RuntimeEffectsInput,
  tab: ProjectConversationTabState,
  conversationId: string,
  streamId: number,
) {
  if (tab.streamId !== streamId || tab.conversationId !== conversationId) {
    return
  }

  try {
    const payload = await getProjectConversation(conversationId)
    if (tab.streamId !== streamId || tab.conversationId !== conversationId) {
      return
    }

    input.conversations.upsertConversation(payload.conversation)
    if (payload.conversation.providerId) {
      tab.providerId = payload.conversation.providerId
    }

    const runtimeState = payload.conversation.runtimePrincipal?.runtimeState?.trim() ?? ''
    if (runtimeState === 'executing' && !input.isActiveTab(tab)) {
      tab.phase = 'awaiting_reply'
      tab.needsHydration = true
      return
    }

    if (input.isActiveTab(tab)) {
      await hydrateTabEntries(tab, conversationId, input.touchTabs)
      if (tab.streamId !== streamId || tab.conversationId !== conversationId) {
        return
      }
      tab.needsHydration = false
      tab.unread = false
    } else {
      tab.needsHydration = true
      tab.unread = true
    }

    if (runtimeState === 'interrupted') {
      tab.phase = 'awaiting_interrupt'
    } else if (runtimeState === 'executing') {
      tab.phase = 'awaiting_reply'
    } else if (input.isActiveTab(tab) && projectConversationHasPendingInterrupt(tab.entries)) {
      tab.phase = 'awaiting_interrupt'
    } else {
      tab.phase = 'idle'
    }

    if (input.isActiveTab(tab)) {
      void refreshTabWorkspaceDiff(tab, conversationId, input.touchTabs)
    }
    input.persistTabs()
  } catch (caughtError) {
    if (tab.streamId !== streamId || tab.conversationId !== conversationId) {
      return
    }
    if (tab.phase !== 'awaiting_interrupt') {
      tab.phase = 'idle'
    }
    input.onError?.(
      caughtError instanceof Error
        ? caughtError.message
        : 'Failed to reconcile project conversation state.',
    )
  }
}
