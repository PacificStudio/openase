import { ApiError, buildRequestHeaders } from './client'
import type { SkillFile } from './contracts'
import { consumeEventStream, type SSEFrame } from './sse'

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

export type SkillRefinementMessagePayload = Record<string, unknown> & {
  type: string
  content?: string
  raw?: Record<string, unknown>
}

export type SkillRefinementInterruptPayload = {
  requestId: string
  kind: string
  options: Array<{ id: string; label: string }>
  payload: Record<string, unknown>
}

export type SkillRefinementThreadStatusPayload = {
  threadId: string
  status: string
  activeFlags: string[]
  entryId?: string
}

export type SkillRefinementSessionStatePayload = {
  status: string
  activeFlags: string[]
  detail?: string
  raw?: Record<string, unknown>
  entryId?: string
}

export type SkillRefinementPlanUpdatedPayload = {
  threadId: string
  turnId: string
  explanation?: string
  plan: Array<{ step: string; status: string }>
  entryId?: string
}

export type SkillRefinementDiffUpdatedPayload = {
  threadId: string
  turnId: string
  diff: string
  entryId?: string
}

export type SkillRefinementReasoningUpdatedPayload = {
  threadId: string
  turnId: string
  itemId: string
  kind: string
  delta?: string
  summaryIndex?: number
  contentIndex?: number
  entryId?: string
}

export type SkillRefinementThreadCompactedPayload = {
  threadId: string
  turnId: string
  entryId?: string
}

export type SkillRefinementSessionAnchorPayload = {
  providerThreadId?: string
  providerTurnId?: string
  providerAnchorId?: string
  providerAnchorKind?: string
  providerTurnSupported?: boolean
}

export type SkillRefinementStreamEvent =
  | { kind: 'session'; payload: SkillRefinementSessionPayload }
  | { kind: 'status'; payload: SkillRefinementStatusPayload }
  | { kind: 'message'; payload: SkillRefinementMessagePayload }
  | { kind: 'interrupt_requested'; payload: SkillRefinementInterruptPayload }
  | { kind: 'thread_status'; payload: SkillRefinementThreadStatusPayload }
  | { kind: 'session_state'; payload: SkillRefinementSessionStatePayload }
  | { kind: 'plan_updated'; payload: SkillRefinementPlanUpdatedPayload }
  | { kind: 'diff_updated'; payload: SkillRefinementDiffUpdatedPayload }
  | { kind: 'reasoning_updated'; payload: SkillRefinementReasoningUpdatedPayload }
  | { kind: 'thread_compacted'; payload: SkillRefinementThreadCompactedPayload }
  | { kind: 'session_anchor'; payload: SkillRefinementSessionAnchorPayload }
  | { kind: 'result'; payload: SkillRefinementResultPayload }
  | { kind: 'error'; payload: SkillRefinementErrorPayload }

export async function streamSkillRefinement(
  request: SkillRefinementRequest,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: SkillRefinementStreamEvent) => void
  },
) {
  const headers = buildRequestHeaders('POST', {
    accept: 'text/event-stream',
    'Content-Type': 'application/json',
  })
  const skillId = encodeURIComponent(request.skillId)
  const response = await fetch(`/api/v1/skills/${skillId}/refinement-runs`, {
    method: 'POST',
    headers,
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
  })

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
  const headers = buildRequestHeaders('DELETE')
  const response = await fetch(`/api/v1/skills/refinement-runs/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
    headers,
    credentials: 'same-origin',
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

export function parseSkillRefinementStreamEvent(
  frame: SSEFrame,
): SkillRefinementStreamEvent | null {
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
    case 'message': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'message',
        payload: {
          ...object,
          type: readRequiredString(object, 'type'),
          content: readOptionalString(object, 'content'),
          raw: readOptionalObject(object, 'raw'),
        },
      }
    }
    case 'interrupt_requested': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupt_requested',
        payload: {
          requestId: readRequiredString(object, 'request_id'),
          kind: readRequiredString(object, 'kind'),
          options: readDecisionOptions(object, 'options'),
          payload: readOptionalObject(object, 'payload') ?? {},
        },
      }
    }
    case 'thread_status': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'thread_status',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          status: readRequiredString(object, 'status'),
          activeFlags: readStringArray(object, 'active_flags'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'session_state': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'session_state',
        payload: {
          status: readRequiredString(object, 'status'),
          activeFlags: readStringArray(object, 'active_flags'),
          detail: readOptionalString(object, 'detail'),
          raw: readOptionalObject(object, 'raw'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'plan_updated': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'plan_updated',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          explanation: readOptionalString(object, 'explanation'),
          plan: readPlanItems(object, 'plan'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'diff_updated': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'diff_updated',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          diff: readRequiredString(object, 'diff'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'reasoning_updated': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'reasoning_updated',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          itemId: readRequiredString(object, 'item_id'),
          kind: readRequiredString(object, 'kind'),
          delta: readOptionalString(object, 'delta'),
          summaryIndex: readOptionalNumber(object, 'summary_index'),
          contentIndex: readOptionalNumber(object, 'content_index'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'thread_compacted': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'thread_compacted',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'session_anchor': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'session_anchor',
        payload: {
          providerThreadId: readOptionalStringField(
            object,
            'provider_thread_id',
            'ProviderThreadID',
          ),
          providerTurnId: readOptionalStringField(object, 'provider_turn_id', 'LastTurnID'),
          providerAnchorId: readOptionalStringField(
            object,
            'provider_anchor_id',
            'ProviderAnchorID',
          ),
          providerAnchorKind: readOptionalStringField(
            object,
            'provider_anchor_kind',
            'ProviderAnchorKind',
          ),
          providerTurnSupported: readOptionalBooleanField(
            object,
            'provider_turn_supported',
            'ProviderTurnSupported',
          ),
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

function readOptionalNumber(object: Record<string, unknown>, key: string): number | undefined {
  const value = object[key]
  return typeof value === 'number' && !Number.isNaN(value) ? value : undefined
}

function readRequiredBoolean(object: Record<string, unknown>, key: string): boolean {
  const value = object[key]
  if (typeof value !== 'boolean') {
    throw new Error(`skill refinement stream payload field ${key} must be a boolean`)
  }
  return value
}

function readOptionalObject(
  object: Record<string, unknown>,
  key: string,
): Record<string, unknown> | undefined {
  const value = object[key]
  if (value == null || typeof value !== 'object' || Array.isArray(value)) {
    return undefined
  }
  return value as Record<string, unknown>
}

function readStringArray(object: Record<string, unknown>, key: string): string[] {
  const value = object[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value
    .map((item) => (typeof item === 'string' ? item.trim() : ''))
    .filter((item) => item !== '')
}

function readDecisionOptions(
  object: Record<string, unknown>,
  key: string,
): Array<{ id: string; label: string }> {
  const value = object[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value
    .map((item) => {
      const record = parseRequiredObject(item)
      const id = readRequiredString(record, 'id')
      const label = readRequiredString(record, 'label')
      return { id, label }
    })
    .filter(Boolean)
}

function readPlanItems(
  object: Record<string, unknown>,
  key: string,
): Array<{ step: string; status: string }> {
  const value = object[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value
    .map((item) => {
      const record = parseRequiredObject(item)
      return {
        step: readRequiredString(record, 'step'),
        status: readRequiredString(record, 'status'),
      }
    })
    .filter(Boolean)
}

function readOptionalStringField(object: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = readOptionalString(object, key)
    if (value) {
      return value
    }
  }
  return undefined
}

function readOptionalBooleanField(object: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = object[key]
    if (typeof value === 'boolean') {
      return value
    }
  }
  return undefined
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
