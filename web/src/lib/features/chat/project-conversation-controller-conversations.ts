import type { ProjectConversation, ProjectConversationSessionPayload } from '$lib/api/chat'

type ProjectConversationControllerConversationsInput = {
  getProjectId: () => string
  getProviderId: () => string
  getConversations: () => ProjectConversation[]
  setConversations: (value: ProjectConversation[]) => void
}

export function createProjectConversationControllerConversations(
  input: ProjectConversationControllerConversationsInput,
) {
  function sortProjectConversations(items: ProjectConversation[]) {
    return [...items].sort((left, right) => {
      const comparison = conversationActivitySortKey(right).localeCompare(
        conversationActivitySortKey(left),
      )
      return comparison !== 0 ? comparison : right.id.localeCompare(left.id)
    })
  }

  function conversationActivitySortKey(conversation: ProjectConversation) {
    return conversation.lastActivityAt || conversation.updatedAt || conversation.createdAt || ''
  }

  function touchConversation(conversationId: string) {
    if (!conversationId) return
    const now = new Date().toISOString()
    input.setConversations(
      sortProjectConversations(
        input
          .getConversations()
          .map((conversation) =>
            conversation.id === conversationId
              ? { ...conversation, lastActivityAt: now }
              : conversation,
          ),
      ),
    )
  }

  function upsertConversation(conversation: ProjectConversation) {
    input.setConversations(
      sortProjectConversations([
        conversation,
        ...input.getConversations().filter((current) => current.id !== conversation.id),
      ]),
    )
  }

  function applySessionPayload(
    tabConversationId: string,
    payload: ProjectConversationSessionPayload,
  ) {
    const existing = input
      .getConversations()
      .find((conversation) => conversation.id === payload.conversationId)
    const now = new Date().toISOString()
    upsertConversation({
      id: payload.conversationId,
      projectId: existing?.projectId ?? input.getProjectId(),
      userId: existing?.userId ?? '',
      source: 'project_sidebar',
      providerId: existing?.providerId ?? input.getProviderId(),
      title: payload.title ?? existing?.title ?? '',
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
      rollingSummary: payload.rollingSummary ?? existing?.rollingSummary ?? '',
      lastActivityAt: now,
      createdAt: existing?.createdAt ?? now,
      updatedAt: now,
    })
    return !tabConversationId && payload.conversationId ? payload.conversationId : tabConversationId
  }

  return {
    sortProjectConversations,
    touchConversation,
    upsertConversation,
    applySessionPayload,
  }
}

export type ProjectConversationControllerConversations = ReturnType<
  typeof createProjectConversationControllerConversations
>
