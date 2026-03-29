import type { ChatActionProposalPayload, ChatMessagePayload } from '$lib/api/chat'
import type { ChatActionExecutionResult } from './action-proposal-executor'

export type EphemeralChatRole = 'user' | 'assistant' | 'system'

export type EphemeralChatTextEntry = {
  id: string
  role: EphemeralChatRole
  kind: 'text'
  content: string
  streaming: boolean
}

export type EphemeralChatActionProposalEntry = {
  id: string
  role: 'assistant'
  kind: 'action_proposal'
  proposal: ChatActionProposalPayload
  status: 'pending' | 'executing' | 'confirmed' | 'cancelled'
  results: ChatActionExecutionResult[]
}

export type EphemeralChatTranscriptEntry = EphemeralChatTextEntry | EphemeralChatActionProposalEntry

export function mapChatPayloadToTranscriptEntry(
  id: string,
  payload: ChatMessagePayload,
): EphemeralChatTranscriptEntry {
  if (isTextPayload(payload)) {
    return {
      id,
      role: 'assistant',
      kind: 'text',
      content: payload.content,
      streaming: false,
    }
  }

  if (isActionProposalPayload(payload)) {
    return {
      id,
      role: 'assistant',
      kind: 'action_proposal',
      proposal: payload,
      status: 'pending',
      results: [],
    }
  }

  return {
    id,
    role: 'system',
    kind: 'text',
    content: describeSystemMessage(payload.type),
    streaming: false,
  }
}

export function createTextTranscriptEntry(
  id: string,
  role: EphemeralChatRole,
  content: string,
  options?: {
    streaming?: boolean
  },
): EphemeralChatTextEntry {
  return {
    id,
    role,
    kind: 'text',
    content,
    streaming: options?.streaming ?? false,
  }
}

export function isActionProposalEntry(
  entry: EphemeralChatTranscriptEntry,
): entry is EphemeralChatActionProposalEntry {
  return entry.kind === 'action_proposal'
}

export function isTextTranscriptEntry(
  entry: EphemeralChatTranscriptEntry,
): entry is EphemeralChatTextEntry {
  return entry.kind === 'text'
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
