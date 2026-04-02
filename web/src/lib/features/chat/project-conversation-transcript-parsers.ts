import type { ChatDiffPayload } from '$lib/api/chat'
import {
  createProjectConversationTaskStatusEntry,
  type ProjectConversationDiffEntry,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'
import {
  asRecord,
  buildProviderStateDetail,
  buildReasoningDetail,
  buildTaskDetail,
  normalizeDiffPayload,
  parseUnifiedDiffPayloads,
  readBoolean,
  readString,
} from './project-conversation-transcript-parser-helpers'

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

export function createProjectConversationDiffEntriesFromUnifiedDiff(params: {
  idBase: string
  diff: string
}) {
  return parseUnifiedDiffPayloads(params.diff).map((payload, index) =>
    createProjectConversationDiffEntry({
      id: index === 0 ? params.idBase : `${params.idBase}-${index + 1}`,
      payload,
    }),
  )
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
