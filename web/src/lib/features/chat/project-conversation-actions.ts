import { ApiError } from '$lib/api/client'
import { closeProjectConversationRuntime } from '$lib/api/chat'

export async function resetProjectConversationRuntime(conversationId: string) {
  if (!conversationId) {
    return
  }

  await closeProjectConversationRuntime(conversationId).catch(
    (_caughtError: ApiError | Error) => {},
  )
}
