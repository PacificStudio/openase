import { ApiError } from './client'
import type { SkillFile } from './contracts'
import { consumeEventStream, type SSEFrame } from './sse'

const chatUserHeader = 'X-OpenASE-Chat-User'
const chatUserStorageKey = 'openase.ephemeral-chat-user-id'

let cachedChatUserId = ''

export type SkillRefinementRequest = {
  projectId: string
  skillId: string
  message: string
  providerId?: string
  files: Array<{
    path: string
    contentBase64: string
    mediaType?: string
    isExecutable?: boolean
  }>
}

export type SkillRefinementSessionPayload = {
  sessionId: string
  workspacePath: string
}

export type SkillRefinementStatusPhase =
  | 'editing'
  | 'testing'
  | 'retrying'
  | 'verified'
  | 'blocked'
  | 'unverified'

export type SkillRefinementStatusPayload = {
  sessionId: string
  phase: SkillRefinementStatusPhase
  attempt: number
  message: string
}

export type SkillRefinementResultPayload = {
  sessionId: string
  status: 'verified' | 'blocked' | 'unverified'
  workspacePath: string
  providerId: string
  providerName: string
  providerThreadId?: string
  providerTurnId?: string
  attempts: number
  transcriptSummary?: string
  commandOutputSummary?: string
  failureReason?: string
  candidateFiles: SkillFile[]
  candidateBundleHash?: string
}

export type SkillRefinementErrorPayload = {
  message: string
}

export type SkillRefinementStreamEvent =
  | { kind: 'session'; payload: SkillRefinementSessionPayload }
  | { kind: 'status'; payload: SkillRefinementStatusPayload }
  | { kind: 'result'; payload: SkillRefinementResultPayload }
  | { kind: 'error'; payload: SkillRefinementErrorPayload }

export async function streamSkillRefinement(
  request: SkillRefinementRequest,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: SkillRefinementStreamEvent) => void
  },
) {
  const response = await fetch(
    `/api/v1/skills/${encodeURIComponent(request.skillId)}/refinement-runs`,
    {
      method: 'POST',
      headers: {
        accept: 'text/event-stream',
        'Content-Type': 'application/json',
        [chatUserHeader]: resolveChatUserId(),
      },
      body: JSON.stringify({
        project_id: request.projectId,
        message: request.message,
        provider_id: request.providerId,
        files: request.files.map((file) => ({
          path: file.path,
          content_base64: file.contentBase64,
          media_type: file.mediaType,
          is_executable: file.isExecutable ?? false,
        })),
      }),
      credentials: 'same-origin',
      signal: handlers.signal,
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('skill refinement stream response body is unavailable')
  }

  await consumeEventStream(response.body, (frame) => {
    const event = parseSkillRefinementStreamEvent(frame)
    if (event) {
      handlers.onEvent(event)
    }
  })
}

export async function closeSkillRefinementSession(sessionId: string) {
  const response = await fetch(`/api/v1/skills/refinement-runs/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
    headers: {
      [chatUserHeader]: resolveChatUserId(),
    },
    credentials: 'same-origin',
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

function parseSkillRefinementStreamEvent(frame: SSEFrame): SkillRefinementStreamEvent | null {
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
          sessionId: readRequiredString(object, 'session_id'),
          workspacePath: readRequiredString(object, 'workspace_path'),
        },
      }
    }
    case 'status': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'status',
        payload: {
          sessionId: readRequiredString(object, 'session_id'),
          phase: readRequiredString(object, 'phase') as SkillRefinementStatusPhase,
          attempt: readRequiredNumber(object, 'attempt'),
          message: readRequiredString(object, 'message'),
        },
      }
    }
    case 'result': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'result',
        payload: {
          sessionId: readRequiredString(object, 'session_id'),
          status: readRequiredString(object, 'status') as SkillRefinementResultPayload['status'],
          workspacePath: readRequiredString(object, 'workspace_path'),
          providerId: readRequiredString(object, 'provider_id'),
          providerName: readRequiredString(object, 'provider_name'),
          providerThreadId: readOptionalString(object, 'provider_thread_id'),
          providerTurnId: readOptionalString(object, 'provider_turn_id'),
          attempts: readRequiredNumber(object, 'attempts'),
          transcriptSummary: readOptionalString(object, 'transcript_summary'),
          commandOutputSummary: readOptionalString(object, 'command_output_summary'),
          failureReason: readOptionalString(object, 'failure_reason'),
          candidateFiles: readSkillFiles(object, 'candidate_files'),
          candidateBundleHash: readOptionalString(object, 'candidate_bundle_hash'),
        },
      }
    }
    case 'error': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'error',
        payload: {
          message: readRequiredString(object, 'message'),
        },
      }
    }
    default:
      return null
  }
}

function resolveChatUserId() {
  if (typeof window === 'undefined') {
    return 'chat-user-server'
  }
  if (cachedChatUserId) {
    return cachedChatUserId
  }
  try {
    const stored = window.localStorage.getItem(chatUserStorageKey)
    if (stored && stored.trim() !== '') {
      cachedChatUserId = stored
      return cachedChatUserId
    }
  } catch {
    // Ignore storage read failures.
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

function parseJSONObject(raw: string): unknown | null {
  try {
    return JSON.parse(raw) as unknown
  } catch {
    return null
  }
}

function parseRequiredObject(value: unknown): Record<string, unknown> {
  if (value == null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('skill refinement stream payload must be an object')
  }
  return value as Record<string, unknown>
}

function readRequiredString(object: Record<string, unknown>, key: string): string {
  const value = object[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`skill refinement stream payload field ${key} must be a non-empty string`)
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
    throw new Error(`skill refinement stream payload field ${key} must be a number`)
  }
  return value
}

function readRequiredBoolean(object: Record<string, unknown>, key: string): boolean {
  const value = object[key]
  if (typeof value !== 'boolean') {
    throw new Error(`skill refinement stream payload field ${key} must be a boolean`)
  }
  return value
}

function readSkillFiles(object: Record<string, unknown>, key: string): SkillFile[] {
  const value = object[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value.map((item) => {
    const file = parseRequiredObject(item)
    return {
      path: readRequiredString(file, 'path'),
      file_kind: readRequiredString(file, 'file_kind') as SkillFile['file_kind'],
      media_type: readRequiredString(file, 'media_type'),
      encoding: readRequiredString(file, 'encoding') as SkillFile['encoding'],
      is_executable: readRequiredBoolean(file, 'is_executable'),
      size_bytes: readRequiredNumber(file, 'size_bytes'),
      sha256: readRequiredString(file, 'sha256'),
      content: readOptionalString(file, 'content'),
      content_base64: readOptionalString(file, 'content_base64'),
    }
  })
}
