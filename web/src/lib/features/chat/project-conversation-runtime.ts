import { ApiError } from '$lib/api/client'
import {
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  type ProjectConversationEntry,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import { watchProjectConversationMux } from './project-conversation-event-bus'
import { isAbortError } from './project-conversation-storage'

export function startProjectConversationStream(params: {
  projectId: string
  conversationId: string
  abortController: AbortController | null
  onEvent: (event: ProjectConversationStreamEvent) => void
  onReconnect?: () => void
  onError: (message: string) => void
}) {
  params.abortController?.abort()

  const controller = new AbortController()
  const subscription = watchProjectConversationMux({
    projectId: params.projectId,
    conversationId: params.conversationId,
    signal: controller.signal,
    onEvent: params.onEvent,
    onReconnect: params.onReconnect,
  })
  const stream = subscription.stream.catch((caughtError) => {
    if (isAbortError(caughtError)) {
      return
    }
    params.onError(
      caughtError instanceof ApiError
        ? caughtError.detail
        : 'Project conversation stream disconnected.',
    )
  })

  return { controller, stream, connected: subscription.connected }
}

export async function restoreProjectConversation(params: {
  projectId: string
  providerId: string
  readConversationId: (projectId: string, providerId: string) => string
  storeConversationId: (projectId: string, providerId: string, conversationId: string) => void
  loadConversation: (conversationId: string) => Promise<void>
  clearConversation: () => void
}) {
  const storedConversationId = params.readConversationId(params.projectId, params.providerId)

  if (storedConversationId) {
    try {
      const conversation = await getProjectConversation(storedConversationId)
      if (conversation.conversation.providerId === params.providerId) {
        await params.loadConversation(storedConversationId)
        return
      }
    } catch {
      // Fall through to list lookup.
    }
  }

  const listPayload = await listProjectConversations({
    projectId: params.projectId,
    providerId: params.providerId,
  })
  const current = listPayload.conversations[0]
  if (!current) {
    params.clearConversation()
    return
  }

  params.storeConversationId(params.projectId, params.providerId, current.id)
  await params.loadConversation(current.id)
}

export async function loadProjectConversation(params: {
  conversationId: string
  mapEntries: (entries: ProjectConversationEntry[]) => unknown
  setConversationId: (conversationId: string) => void
  setEntries: (entries: unknown) => void
  resetActiveAssistantEntry: () => void
  connectStream: (conversationId: string) => void
}) {
  const payload = await listProjectConversationEntries(params.conversationId)
  params.setConversationId(params.conversationId)
  params.setEntries(params.mapEntries(payload.entries))
  params.resetActiveAssistantEntry()
  params.connectStream(params.conversationId)
}
