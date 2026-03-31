import { ApiError } from '$lib/api/client'
import {
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  watchProjectConversation,
  type ProjectConversationEntry,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import { isAbortError } from './project-conversation-storage'

export function startProjectConversationStream(params: {
  conversationId: string
  abortController: AbortController | null
  requestId: number
  onEvent: (event: ProjectConversationStreamEvent) => void
  onError: (message: string) => void
}) {
  params.abortController?.abort()

  const controller = new AbortController()
  const nextRequestId = params.requestId + 1
  const stream = watchProjectConversation(params.conversationId, {
    signal: controller.signal,
    onEvent: (event) => {
      if (nextRequestId === params.requestId + 1) {
        params.onEvent(event)
      }
    },
  }).catch((caughtError) => {
    if (isAbortError(caughtError)) {
      return
    }
    params.onError(
      caughtError instanceof ApiError
        ? caughtError.detail
        : 'Project conversation stream disconnected.',
    )
  })

  return { controller, requestId: nextRequestId, stream }
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
  connectStream: (conversationId: string) => Promise<void>
}) {
  const payload = await listProjectConversationEntries(params.conversationId)
  params.setConversationId(params.conversationId)
  params.setEntries(params.mapEntries(payload.entries))
  params.resetActiveAssistantEntry()
  await params.connectStream(params.conversationId)
}
