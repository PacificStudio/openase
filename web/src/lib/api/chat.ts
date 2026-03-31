import { ApiError } from './client'
import { consumeEventStream, type SSEFrame } from './sse'

const chatUserHeader = 'X-OpenASE-Chat-User'
const chatUserStorageKey = 'openase.ephemeral-chat-user-id'

let cachedChatUserId = ''

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
  turnsRemaining?: number
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
  actions: ChatActionProposalAction[]
}

export type ChatDiffLineOp = 'context' | 'add' | 'remove'

export type ChatDiffLine = {
  op: ChatDiffLineOp
  text: string
}

export type ChatDiffHunk = {
  oldStart: number
  oldLines: number
  newStart: number
  newLines: number
  lines: ChatDiffLine[]
}

export type ChatDiffPayload = {
  type: 'diff'
  file: string
  hunks: ChatDiffHunk[]
}

export type ChatTaskPayload = {
  type: string
  raw?: unknown
}

export type ChatMessagePayload =
  | ChatTextPayload
  | ChatDiffPayload
  | ChatActionProposalPayload
  | ChatTaskPayload

export type ChatActionMethod = 'POST' | 'PATCH' | 'PUT' | 'DELETE'

export type ChatActionProposalAction = {
  method: ChatActionMethod
  path: string
  body?: Record<string, unknown>
}

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
      [chatUserHeader]: resolveEphemeralChatUserId(),
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
    headers: {
      [chatUserHeader]: resolveEphemeralChatUserId(),
    },
    credentials: 'same-origin',
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

function resolveEphemeralChatUserId() {
  if (cachedChatUserId) {
    return cachedChatUserId
  }

  if (typeof window === 'undefined') {
    cachedChatUserId = 'anonymous-browser'
    return cachedChatUserId
  }

  try {
    const stored = window.localStorage.getItem(chatUserStorageKey)?.trim()
    if (stored) {
      cachedChatUserId = stored
      return cachedChatUserId
    }
  } catch {
    // Ignore storage access failures and fall back to an in-memory identifier.
  }

  const generated =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `chat-user-${Date.now().toString(36)}`
  cachedChatUserId = generated

  try {
    window.localStorage.setItem(chatUserStorageKey, generated)
  } catch {
    // Ignore storage write failures and keep the in-memory identifier.
  }

  return cachedChatUserId
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
        actions: readActionProposalActions(object),
      },
    }
  }

  if (type === 'diff') {
    return {
      kind: 'message',
      payload: {
        type,
        file: readRequiredString(object, 'file'),
        hunks: readDiffHunks(object),
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
    turnsRemaining: readOptionalNumber(object, 'turns_remaining'),
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

function readActionProposalActions(object: Record<string, unknown>): ChatActionProposalAction[] {
  const actions = object.actions
  if (!Array.isArray(actions)) {
    throw new Error('chat stream action_proposal actions must be an array')
  }

  return actions.map((action, index) => parseActionProposalAction(action, index))
}

function parseActionProposalAction(value: unknown, index: number): ChatActionProposalAction {
  const object = parseRequiredObject(value)
  const method = readRequiredString(object, 'method').toUpperCase()
  if (!isActionMethod(method)) {
    throw new Error(`chat stream action_proposal action ${index} method is unsupported`)
  }

  const body = object.body
  if (body !== undefined && (body == null || typeof body !== 'object' || Array.isArray(body))) {
    throw new Error(`chat stream action_proposal action ${index} body must be an object`)
  }

  return {
    method,
    path: readRequiredString(object, 'path'),
    body: body as Record<string, unknown> | undefined,
  }
}

function isActionMethod(value: string): value is ChatActionMethod {
  return value === 'POST' || value === 'PATCH' || value === 'PUT' || value === 'DELETE'
}

function readDiffHunks(object: Record<string, unknown>): ChatDiffHunk[] {
  const hunks = object.hunks
  if (!Array.isArray(hunks) || hunks.length === 0) {
    throw new Error('chat stream diff hunks must be a non-empty array')
  }

  return hunks.map((hunk, index) => parseDiffHunk(hunk, index))
}

function parseDiffHunk(value: unknown, index: number): ChatDiffHunk {
  const object = parseRequiredObject(value)
  const oldStart = readRequiredNumber(object, 'old_start')
  const oldLines = readRequiredNumber(object, 'old_lines')
  const newStart = readRequiredNumber(object, 'new_start')
  const newLines = readRequiredNumber(object, 'new_lines')
  const lines = readDiffLines(object, index)

  if (!Number.isInteger(oldStart) || oldStart < 1) {
    throw new Error(`chat stream diff hunk ${index} old_start must be a positive integer`)
  }
  if (!Number.isInteger(newStart) || newStart < 1) {
    throw new Error(`chat stream diff hunk ${index} new_start must be a positive integer`)
  }
  if (!Number.isInteger(oldLines) || oldLines < 0) {
    throw new Error(`chat stream diff hunk ${index} old_lines must be a non-negative integer`)
  }
  if (!Number.isInteger(newLines) || newLines < 0) {
    throw new Error(`chat stream diff hunk ${index} new_lines must be a non-negative integer`)
  }

  return {
    oldStart,
    oldLines,
    newStart,
    newLines,
    lines,
  }
}

function readDiffLines(object: Record<string, unknown>, index: number): ChatDiffLine[] {
  const lines = object.lines
  if (!Array.isArray(lines) || lines.length === 0) {
    throw new Error(`chat stream diff hunk ${index} lines must be a non-empty array`)
  }

  return lines.map((line, lineIndex) => parseDiffLine(line, index, lineIndex))
}

function parseDiffLine(value: unknown, hunkIndex: number, lineIndex: number): ChatDiffLine {
  const object = parseRequiredObject(value)
  const op = readRequiredString(object, 'op')
  if (!isDiffLineOp(op)) {
    throw new Error(`chat stream diff hunk ${hunkIndex} line ${lineIndex} op is unsupported`)
  }

  return {
    op,
    text: readRequiredString(object, 'text'),
  }
}

function isDiffLineOp(value: string): value is ChatDiffLineOp {
  return value === 'context' || value === 'add' || value === 'remove'
}
