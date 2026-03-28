import type { ChatMessagePayload } from '$lib/api/chat'

export type EphemeralChatRole = 'user' | 'assistant' | 'system'

export type EphemeralChatTranscriptEntry = {
  id: string
  role: EphemeralChatRole
  content: string
}

export function mapChatPayloadToTranscriptEntry(
  payload: ChatMessagePayload,
): Omit<EphemeralChatTranscriptEntry, 'id'> {
  if (isTextPayload(payload)) {
    return {
      role: 'assistant',
      content: payload.content,
    }
  }

  if (isActionProposalPayload(payload)) {
    return {
      role: 'system',
      content: `Action proposal: ${payload.summary ?? 'Awaiting confirmation.'}`,
    }
  }

  return {
    role: 'system',
    content: describeSystemMessage(payload.type),
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function describeSystemMessage(type: string) {
  switch (type) {
    case 'task_started':
      return 'Assistant started a background task.'
    case 'task_progress':
      return 'Assistant reported task progress.'
    case 'task_notification':
      return 'Assistant emitted a task notification.'
    default:
      return `Assistant emitted ${type}.`
  }
}

function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

function isActionProposalPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'action_proposal' }> {
  return payload.type === 'action_proposal'
}
