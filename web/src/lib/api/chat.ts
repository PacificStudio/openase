import { ApiError } from './client'
import { consumeEventStream, type SSEFrame } from './sse'

export type ChatSource = 'harness_editor' | 'project_sidebar' | 'ticket_detail'

export type ChatTurnRequest = {
  message: string
  source: ChatSource
  providerId?: string
  sessionId?: string
  context: {
    projectId: string
    workflowId?: string
    ticketId?: string
  }
}

export type ChatTextPayload = {
  type: 'text'
  content: string
}

export type ChatDonePayload = {
  sessionId: string
  turnsUsed: number
  turnsRemaining: number
  costUSD?: number
}

export type ChatSessionPayload = {
  sessionId: string
}

export type ChatErrorPayload = {
  message: string
}

export type ChatActionProposalPayload = {
  type: 'action_proposal'
  summary?: string
  actions?: unknown[]
}

export type ChatTaskPayload = {
  type: string
  raw?: unknown
}

export type ChatMessagePayload = ChatTextPayload | ChatActionProposalPayload | ChatTaskPayload

export type ChatStreamEvent =
  | { kind: 'session'; payload: ChatSessionPayload }
  | { kind: 'message'; payload: ChatMessagePayload }
  | { kind: 'done'; payload: ChatDonePayload }
  | { kind: 'error'; payload: ChatErrorPayload }

export async function streamChatTurn(
  request: ChatTurnRequest,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: ChatStreamEvent) => void
  },
) {
  const response = await fetch('/api/v1/chat', {
    method: 'POST',
    headers: {
      accept: 'text/event-stream',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: request.message,
      source: request.source,
      provider_id: request.providerId,
      session_id: request.sessionId,
      context: {
        project_id: request.context.projectId,
        workflow_id: request.context.workflowId,
        ticket_id: request.context.ticketId,
      },
    }),
    credentials: 'same-origin',
    signal: handlers.signal,
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('chat stream response body is unavailable')
  }

  await consumeEventStream(response.body, (frame) => {
    const event = parseChatStreamEvent(frame)
    if (event) {
      handlers.onEvent(event)
    }
  })
}

export async function closeChatSession(sessionId: string) {
  const response = await fetch(`/api/v1/chat/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
    credentials: 'same-origin',
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

function parseChatStreamEvent(frame: SSEFrame): ChatStreamEvent | null {
  const payload = parseJSONObject(frame.data)
  if (payload == null) {
    return null
  }

  switch (frame.event) {
    case 'session':
      return { kind: 'session', payload: parseSessionPayload(payload) }
    case 'message':
      return parseMessageEvent(payload)
    case 'done':
      return { kind: 'done', payload: parseDonePayload(payload) }
    case 'error':
      return { kind: 'error', payload: parseErrorPayload(payload) }
    default:
      return null
  }
}

function parseMessageEvent(payload: unknown): ChatStreamEvent {
  const object = parseRequiredObject(payload)
  const type = readRequiredString(object, 'type')

  if (type === 'text') {
    return {
      kind: 'message',
      payload: {
        type,
        content: readRequiredString(object, 'content'),
      },
    }
  }

  if (type === 'action_proposal') {
    return {
      kind: 'message',
      payload: {
        type,
        summary: readOptionalString(object, 'summary'),
        actions: Array.isArray(object.actions) ? object.actions : undefined,
      },
    }
  }

  return {
    kind: 'message',
    payload: {
      type,
      raw: object.raw,
    },
  }
}

function parseSessionPayload(payload: unknown): ChatSessionPayload {
  const object = parseRequiredObject(payload)
  return {
    sessionId: readRequiredString(object, 'session_id'),
  }
}

function parseDonePayload(payload: unknown): ChatDonePayload {
  const object = parseRequiredObject(payload)
  return {
    sessionId: readRequiredString(object, 'session_id'),
    turnsUsed: readRequiredNumber(object, 'turns_used'),
    turnsRemaining: readRequiredNumber(object, 'turns_remaining'),
    costUSD: readOptionalNumber(object, 'cost_usd'),
  }
}

function parseErrorPayload(payload: unknown): ChatErrorPayload {
  const object = parseRequiredObject(payload)
  return {
    message: readRequiredString(object, 'message'),
  }
}

function parseJSONObject(raw: string): unknown | null {
  try {
    return JSON.parse(raw) as unknown
  } catch {
    return null
  }
}

function parseRequiredObject(value: unknown): Record<string, unknown> {
  if (value == null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('chat stream payload must be an object')
  }

  return value as Record<string, unknown>
}

function readRequiredString(object: Record<string, unknown>, key: string): string {
  const value = object[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`chat stream payload field ${key} must be a non-empty string`)
  }
  return value
}

function readOptionalString(object: Record<string, unknown>, key: string): string | undefined {
  const value = object[key]
  return typeof value === 'string' && value.trim() !== '' ? value : undefined
}

function readRequiredNumber(object: Record<string, unknown>, key: string): number {
  const value = object[key]
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`chat stream payload field ${key} must be a number`)
  }
  return value
}

function readOptionalNumber(object: Record<string, unknown>, key: string): number | undefined {
  const value = object[key]
  return typeof value === 'number' && !Number.isNaN(value) ? value : undefined
}
