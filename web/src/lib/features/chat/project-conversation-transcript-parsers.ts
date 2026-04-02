import type { ChatDiffHunk, ChatDiffPayload } from '$lib/api/chat'
import {
  createProjectConversationTaskStatusEntry,
  type ProjectConversationDiffEntry,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'

export function createProjectConversationDiffEntry(params: {
  id: string
  payload: Record<string, unknown> | ChatDiffPayload
}) {
  return {
    id: params.id,
    kind: 'diff',
    role: 'assistant',
    diff: normalizeDiffPayload(params.payload, params.id),
  } satisfies ProjectConversationDiffEntry
}

export function mapProjectConversationTaskEntry(params: {
  id: string
  turnId?: string
  type: string
  raw?: unknown
}): ProjectConversationTranscriptEntry | null {
  const raw = asRecord(params.raw)

  if (params.type === 'task_notification') {
    const tool = readString(raw, 'tool')
    if (tool) {
      return {
        id: params.id,
        kind: 'tool_call',
        role: 'system',
        turnId: params.turnId,
        tool,
        arguments: raw?.arguments,
      }
    }
  }

  if (params.type === 'task_progress') {
    const stream = readString(raw, 'stream')
    const content = readString(raw, 'text')
    if (stream === 'command' && content) {
      return {
        id: params.id,
        kind: 'command_output',
        role: 'system',
        turnId: params.turnId,
        stream,
        command: readString(raw, 'command') || undefined,
        phase: readString(raw, 'phase') || undefined,
        snapshot: readBoolean(raw, 'snapshot'),
        content,
      }
    }
  }

  const statusDetail = buildTaskDetail(raw)
  switch (params.type) {
    case 'thread_status':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'thread_status',
        title: 'Codex thread status',
        detail: buildProviderStateDetail(raw),
        raw: raw ?? undefined,
      })
    case 'session_state':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'session_state',
        title: 'Claude session status',
        detail: buildProviderStateDetail(raw),
        raw: raw ?? undefined,
      })
    case 'task_started':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'task_started',
        title: 'Task started',
        detail: statusDetail,
        raw: raw ?? undefined,
      })
    case 'task_progress':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'task_progress',
        title: 'Task progress',
        detail: statusDetail,
        raw: raw ?? undefined,
      })
    case 'task_notification':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'task_notification',
        title: 'Task notification',
        detail: statusDetail,
        raw: raw ?? undefined,
      })
    case 'turn_reasoning_updated':
      return createProjectConversationTaskStatusEntry({
        id: params.id,
        turnId: params.turnId,
        statusType: 'reasoning_updated',
        title: 'Reasoning update',
        detail: buildReasoningDetail(raw),
        raw: raw ?? undefined,
      })
    default:
      return null
  }
}

function buildProviderStateDetail(raw: Record<string, unknown> | null) {
  const status = readString(raw, 'status')
  const detail = readString(raw, 'detail')
  const flags = readStringList(raw, 'active_flags')

  const parts = [status, detail, flags.length > 0 ? flags.join(', ') : undefined].filter(Boolean)
  return parts.length > 0 ? parts.join(' · ') : undefined
}

export function createProjectConversationTurnDoneEntry(params: {
  id: string
  turnId?: string
  costUSD?: number
}) {
  const detail =
    typeof params.costUSD === 'number' ? `Cost: $${params.costUSD.toFixed(2)}` : undefined

  return createProjectConversationTaskStatusEntry({
    id: params.id,
    turnId: params.turnId,
    statusType: 'turn_done',
    title: 'Turn completed',
    detail,
  })
}

export function createProjectConversationErrorEntry(params: {
  id: string
  turnId?: string
  message: string
}) {
  return createProjectConversationTaskStatusEntry({
    id: params.id,
    turnId: params.turnId,
    statusType: 'error',
    title: 'Turn failed',
    detail: params.message.trim() || undefined,
  })
}

function normalizeDiffPayload(
  payload: Record<string, unknown> | ChatDiffPayload,
  entryId: string,
): ChatDiffPayload {
  const object = asRecord(payload) ?? {}

  return {
    type: 'diff',
    entryId,
    file: readString(object, 'file') || '',
    hunks: readDiffHunks(object.hunks),
  }
}

function readDiffHunks(value: unknown): ChatDiffHunk[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => {
      const hunk = asRecord(item)
      if (!hunk) {
        return null
      }

      const lines = Array.isArray(hunk.lines)
        ? hunk.lines
            .map((line) => {
              const record = asRecord(line)
              const op = readString(record, 'op')
              const text = readString(record, 'text')
              if (!record || !op || text == null) {
                return null
              }
              if (op !== 'context' && op !== 'add' && op !== 'remove') {
                return null
              }

              return { op, text } as ChatDiffHunk['lines'][number]
            })
            .filter((line): line is ChatDiffHunk['lines'][number] => line != null)
        : []

      return {
        oldStart: readNumber(hunk, 'oldStart', 'old_start') ?? 0,
        oldLines: readNumber(hunk, 'oldLines', 'old_lines') ?? 0,
        newStart: readNumber(hunk, 'newStart', 'new_start') ?? 0,
        newLines: readNumber(hunk, 'newLines', 'new_lines') ?? 0,
        lines,
      } satisfies ChatDiffHunk
    })
    .filter((hunk): hunk is ChatDiffHunk => hunk != null)
}

function buildTaskDetail(raw: Record<string, unknown> | null) {
  return (
    readString(raw, 'message') ||
    readString(raw, 'text') ||
    describeStream(raw) ||
    describeStatus(raw)
  )
}

function describeStream(raw: Record<string, unknown> | null) {
  const stream = readString(raw, 'stream')
  const phase = readString(raw, 'phase')
  if (!stream && !phase) {
    return undefined
  }
  return [stream, phase].filter(Boolean).join(' / ')
}

function describeStatus(raw: Record<string, unknown> | null) {
  const status = readString(raw, 'status')
  return status ? `Status: ${status}` : undefined
}

function buildReasoningDetail(raw: Record<string, unknown> | null) {
  const delta = readString(raw, 'delta')
  if (delta) {
    return delta
  }

  const kind = readString(raw, 'kind')
  if (!kind) {
    return undefined
  }
  return `Kind: ${kind.replace(/_/g, ' ')}`
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null
  }
  return value as Record<string, unknown>
}

function readString(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key]
  return typeof value === 'string' ? value : undefined
}

function readBoolean(record: Record<string, unknown> | null, key: string) {
  return record?.[key] === true
}

function readNumber(record: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
  }
  return undefined
}

function readStringList(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
}
