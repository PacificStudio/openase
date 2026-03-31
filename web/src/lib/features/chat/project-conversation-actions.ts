import { ApiError } from '$lib/api/client'
import {
  closeProjectConversationRuntime,
  executeProjectConversationActionProposal,
} from '$lib/api/chat'
import type { ChatActionExecutionResult } from './action-proposal-executor'
import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'

export async function resetProjectConversationRuntime(conversationId: string) {
  if (!conversationId) {
    return
  }

  await closeProjectConversationRuntime(conversationId).catch(() => {})
}

export async function confirmProjectConversationActionProposal(params: {
  conversationId: string
  entryId: string
  entries: ProjectConversationTranscriptEntry[]
  setEntries: (entries: ProjectConversationTranscriptEntry[]) => void
  onError?: (message: string) => void
}) {
  params.setEntries(
    params.entries.map((entry) =>
      entry.kind === 'action_proposal' && entry.id === params.entryId
        ? { ...entry, status: 'executing' }
        : entry,
    ),
  )

  try {
    const payload = await executeProjectConversationActionProposal(
      params.conversationId,
      params.entryId,
    )
    const results = payload.results as ChatActionExecutionResult[]
    params.setEntries(
      params.entries.map((entry) =>
        entry.kind === 'action_proposal' && entry.id === params.entryId
          ? { ...entry, status: 'confirmed', results }
          : entry,
      ),
    )
  } catch (caughtError) {
    params.onError?.(
      caughtError instanceof ApiError
        ? caughtError.detail
        : 'Failed to execute project conversation action proposal.',
    )
    params.setEntries(
      params.entries.map((entry) =>
        entry.kind === 'action_proposal' && entry.id === params.entryId
          ? { ...entry, status: 'pending' }
          : entry,
      ),
    )
  }
}
