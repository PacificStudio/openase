import { ApiError } from './client'
import { consumeEventStream, type SSEFrame } from './sse'
import type { ProjectAIFocus } from '$lib/features/chat/project-ai-focus'

const chatUserHeader = 'X-OpenASE-Chat-User'
const chatUserStorageKey = 'openase.ephemeral-chat-user-id'

let cachedChatUserId = ''

export type ChatSource = 'harness_editor' | 'skill_editor' | 'project_sidebar' | 'ticket_detail'

export type ChatTurnRequest = {
  message: string
  source: ChatSource
  providerId?: string
  sessionId?: string
  context: {
    projectId: string
    workflowId?: string
    ticketId?: string
    harnessDraft?: string
    skillId?: string
    skillFilePath?: string
    skillFileDraft?: string
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
  entryId?: string
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
  entryId?: string
  file: string
  hunks: ChatDiffHunk[]
}

export type ChatBundleDiffFile = {
  file: string
  hunks: ChatDiffHunk[]
}

export type ChatBundleDiffPayload = {
  type: 'bundle_diff'
  entryId?: string
  files: ChatBundleDiffFile[]
}

export type ChatTaskPayload = {
  type: string
  raw?: unknown
}

export type ChatMessagePayload =
  | ChatTextPayload
  | ChatDiffPayload
  | ChatBundleDiffPayload
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

export type ProjectConversation = {
  id: string
  projectId: string
  userId: string
  source: 'project_sidebar'
  providerId: string
  status: string
  rollingSummary: string
  lastActivityAt: string
  createdAt: string
  updatedAt: string
}

export type ProjectConversationEntry = {
  id: string
  conversationId: string
  turnId?: string
  seq: number
  kind: string
  payload: Record<string, unknown>
  createdAt: string
}

export type ProjectConversationInterruptOption = {
  id: string
  label: string
}

export type ProjectConversationInterruptRequestedPayload = {
  interruptId: string
  provider: string
  kind: string
  options: ProjectConversationInterruptOption[]
  payload: Record<string, unknown>
}

export type ProjectConversationInterruptResolvedPayload = {
  interruptId: string
  decision?: string
}

export type ProjectConversationTurnRequest = {
  message: string
  focus?: ProjectAIFocus | null
}

export type ProjectConversationTurnDonePayload = {
  conversationId: string
  turnId: string
  costUSD?: number
}

export type ProjectConversationSessionPayload = {
  conversationId: string
  runtimeState: string
}

export type ProjectConversationStreamEvent =
  | { kind: 'session'; payload: ProjectConversationSessionPayload }
  | { kind: 'message'; payload: ChatMessagePayload }
  | { kind: 'interrupt_requested'; payload: ProjectConversationInterruptRequestedPayload }
  | { kind: 'interrupt_resolved'; payload: ProjectConversationInterruptResolvedPayload }
  | { kind: 'turn_done'; payload: ProjectConversationTurnDonePayload }
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
        harness_draft: request.context.harnessDraft,
        skill_id: request.context.skillId,
        skill_file_path: request.context.skillFilePath,
        skill_file_draft: request.context.skillFileDraft,
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

export function createProjectConversation(request: { providerId: string; projectId: string }) {
  return fetchJSON<{ conversation: ProjectConversation }>('/api/v1/chat/conversations', {
    method: 'POST',
    body: {
      source: 'project_sidebar',
      provider_id: request.providerId,
      context: { project_id: request.projectId },
    },
  })
}

export function listProjectConversations(request: { projectId: string; providerId?: string }) {
  return fetchJSON<{ conversations: ProjectConversation[] }>('/api/v1/chat/conversations', {
    params: {
      project_id: request.projectId,
      provider_id: request.providerId,
    },
  })
}

export function getProjectConversation(conversationId: string) {
  return fetchJSON<{ conversation: ProjectConversation }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}`,
  )
}

export function listProjectConversationEntries(conversationId: string) {
  return fetchJSON<{ entries: ProjectConversationEntry[] }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/entries`,
  )
}

export function startProjectConversationTurn(
  conversationId: string,
  request: ProjectConversationTurnRequest,
) {
  return fetchJSON<{ turn: { id: string; turn_index: number; status: string } }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/turns`,
    {
      method: 'POST',
      body: {
        message: request.message,
        focus: serializeProjectConversationFocus(request.focus),
      },
    },
  )
}

function serializeProjectConversationFocus(focus: ProjectAIFocus | null | undefined) {
  if (!focus) {
    return undefined
  }

  switch (focus.kind) {
    case 'workflow':
      return {
        kind: focus.kind,
        workflow_id: focus.workflowId,
        workflow_name: focus.workflowName,
        workflow_type: focus.workflowType,
        harness_path: focus.harnessPath,
        is_active: focus.isActive,
        selected_area: focus.selectedArea,
        has_dirty_draft: focus.hasDirtyDraft,
      }
    case 'skill':
      return {
        kind: focus.kind,
        skill_id: focus.skillId,
        skill_name: focus.skillName,
        selected_file_path: focus.selectedFilePath,
        bound_workflow_names: focus.boundWorkflowNames,
        has_dirty_draft: focus.hasDirtyDraft,
      }
    case 'ticket':
      return {
        kind: focus.kind,
        ticket_id: focus.ticketId,
        ticket_identifier: focus.ticketIdentifier,
        ticket_title: focus.ticketTitle,
        ticket_status: focus.ticketStatus,
        selected_area: focus.selectedArea,
      }
    case 'machine':
      return {
        kind: focus.kind,
        machine_id: focus.machineId,
        machine_name: focus.machineName,
        machine_host: focus.machineHost,
        machine_status: focus.machineStatus,
        selected_area: focus.selectedArea,
        health_summary: focus.healthSummary,
      }
  }
}

export async function watchProjectConversation(
  conversationId: string,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: ProjectConversationStreamEvent) => void
  },
) {
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/stream`,
    {
      method: 'GET',
      headers: {
        accept: 'text/event-stream',
        [chatUserHeader]: resolveEphemeralChatUserId(),
      },
      credentials: 'same-origin',
      signal: handlers.signal,
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('project conversation stream response body is unavailable')
  }

  await consumeEventStream(response.body, (frame) => {
    const event = parseProjectConversationStreamEvent(frame)
    if (event) {
      handlers.onEvent(event)
    }
  })
}

export function respondProjectConversationInterrupt(
  conversationId: string,
  interruptId: string,
  body: {
    decision?: string | null
    answer?: Record<string, unknown> | null
  },
) {
  return fetchJSON<{ interrupt: Record<string, unknown> }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/interrupts/${encodeURIComponent(interruptId)}/respond`,
    {
      method: 'POST',
      body: {
        decision: body.decision ?? undefined,
        answer: body.answer ?? undefined,
      },
    },
  )
}

export function executeProjectConversationActionProposal(conversationId: string, entryId: string) {
  return fetchJSON<{ result_entry: ProjectConversationEntry; results: Record<string, unknown>[] }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/action-proposals/${encodeURIComponent(entryId)}/execute`,
    { method: 'POST' },
  )
}

export async function closeProjectConversationRuntime(conversationId: string) {
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/runtime`,
    {
      method: 'DELETE',
      headers: {
        [chatUserHeader]: resolveEphemeralChatUserId(),
      },
      credentials: 'same-origin',
    },
  )

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
      return { kind: 'message', payload: parseMessagePayload(payload) }
    case 'done':
      return { kind: 'done', payload: parseDonePayload(payload) }
    case 'error':
      return { kind: 'error', payload: parseErrorPayload(payload) }
    default:
      return null
  }
}

function parseProjectConversationStreamEvent(
  frame: SSEFrame,
): ProjectConversationStreamEvent | null {
  const payload = parseJSONObject(frame.data)
  if (payload == null) {
    return null
  }

  switch (frame.event) {
    case 'session': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'session',
        payload: {
          conversationId: readRequiredString(object, 'conversation_id'),
          runtimeState: readRequiredString(object, 'runtime_state'),
        },
      }
    }
    case 'message':
      return { kind: 'message', payload: parseMessagePayload(payload) }
    case 'interrupt_requested': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupt_requested',
        payload: {
          interruptId: readRequiredString(object, 'interrupt_id'),
          provider: readRequiredString(object, 'provider'),
          kind: readRequiredString(object, 'kind'),
          options: readInterruptOptions(object),
          payload: readOptionalObject(object, 'payload') ?? {},
        },
      }
    }
    case 'interrupt_resolved': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupt_resolved',
        payload: {
          interruptId: readRequiredString(object, 'interrupt_id'),
          decision: readOptionalString(object, 'decision'),
        },
      }
    }
    case 'turn_done': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'turn_done',
        payload: {
          conversationId: readRequiredString(object, 'conversation_id'),
          turnId: readRequiredString(object, 'turn_id'),
          costUSD: readOptionalNumber(object, 'cost_usd'),
        },
      }
    }
    case 'error':
      return { kind: 'error', payload: parseErrorPayload(payload) }
    default:
      return null
  }
}

function parseMessagePayload(payload: unknown): ChatMessagePayload {
  const object = parseRequiredObject(payload)
  const type = readRequiredString(object, 'type')

  if (type === 'text') {
    return {
      type,
      content: readRequiredString(object, 'content'),
    }
  }

  if (type === 'action_proposal') {
    return {
      type,
      entryId: readOptionalString(object, 'entry_id'),
      summary: readOptionalString(object, 'summary'),
      actions: readActionProposalActions(object),
    }
  }

  if (type === 'diff') {
    return {
      type,
      entryId: readOptionalString(object, 'entry_id'),
      file: readRequiredString(object, 'file'),
      hunks: readDiffHunks(object),
    }
  }

  if (type === 'bundle_diff') {
    return {
      type,
      entryId: readOptionalString(object, 'entry_id'),
      files: readBundleDiffFiles(object),
    }
  }

  return {
    type,
    raw: readOptionalObject(object, 'raw') ?? object,
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

function readOptionalObject(
  object: Record<string, unknown>,
  key: string,
): Record<string, unknown> | undefined {
  const value = object[key]
  return value != null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function readInterruptOptions(
  object: Record<string, unknown>,
): ProjectConversationInterruptOption[] {
  const value = object.options
  if (!Array.isArray(value)) {
    return []
  }

  return value.map((item) => {
    const option = parseRequiredObject(item)
    return {
      id: readRequiredString(option, 'id'),
      label: readRequiredString(option, 'label'),
    }
  })
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

function readBundleDiffFiles(object: Record<string, unknown>): ChatBundleDiffFile[] {
  const files = object.files
  if (!Array.isArray(files) || files.length === 0) {
    throw new Error('chat stream bundle_diff files must be a non-empty array')
  }

  const seen = new Set<string>()
  return files.map((item, index) => {
    const fileObject = parseRequiredObject(item)
    const file = readRequiredString(fileObject, 'file')
    if (seen.has(file)) {
      throw new Error(`chat stream bundle_diff file ${index} is duplicated`)
    }
    seen.add(file)
    return {
      file,
      hunks: readDiffHunks(fileObject),
    }
  })
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

async function fetchJSON<T>(
  path: string,
  options?: {
    method?: 'GET' | 'POST' | 'DELETE'
    params?: Record<string, string | undefined>
    body?: unknown
  },
) {
  const url = new URL(path, window.location.origin)
  for (const [key, value] of Object.entries(options?.params ?? {})) {
    if (value) {
      url.searchParams.set(key, value)
    }
  }

  const response = await fetch(url.toString(), {
    method: options?.method ?? 'GET',
    headers: {
      'Content-Type': 'application/json',
      [chatUserHeader]: resolveEphemeralChatUserId(),
    },
    body: options?.body ? JSON.stringify(options.body) : undefined,
    credentials: 'same-origin',
  })
  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}
