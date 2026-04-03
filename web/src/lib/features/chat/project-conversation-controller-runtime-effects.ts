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
  onError?: (message: string) => void
  persistTabs: () => void
  touchTabs: () => void
  connectTabStream: (tab: ProjectConversationTabState, conversationId: string) => void
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
  if (event.kind === 'session') {
    tab.conversationId = input.conversations.applySessionPayload(tab.conversationId, event.payload)
  }
  if ((event.kind === 'session' || event.kind === 'turn_done') && tab.conversationId) {
    void refreshTabWorkspaceDiff(tab, tab.conversationId, input.touchTabs)
  }
}

export async function reconcileTabAfterStreamClose(
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
    if (runtimeState === 'executing') {
      tab.phase = 'awaiting_reply'
      input.connectTabStream(tab, conversationId)
      return
    }

    await hydrateTabEntries(tab, conversationId, input.touchTabs)
    if (tab.streamId !== streamId || tab.conversationId !== conversationId) {
      return
    }

    if (runtimeState === 'interrupted' || projectConversationHasPendingInterrupt(tab.entries)) {
      tab.phase = 'awaiting_interrupt'
      input.connectTabStream(tab, conversationId)
    } else {
      tab.phase = 'idle'
    }

    void refreshTabWorkspaceDiff(tab, conversationId, input.touchTabs)
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
