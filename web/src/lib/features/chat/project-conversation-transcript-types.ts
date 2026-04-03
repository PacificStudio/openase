import type { ChatDiffPayload } from '$lib/api/chat'

export type ProjectConversationRole = 'user' | 'assistant' | 'system'

export type ProjectConversationTextEntry = {
  id: string
  kind: 'text'
  role: ProjectConversationRole
  turnId?: string
  content: string
  streaming: boolean
}

export type ProjectConversationDiffEntry = {
  id: string
  kind: 'diff'
  role: 'assistant'
  diff: ChatDiffPayload
}

export type ProjectConversationToolCallEntry = {
  id: string
  kind: 'tool_call'
  role: 'system'
  turnId?: string
  tool: string
  arguments: unknown
}

export type ProjectConversationCommandOutputEntry = {
  id: string
  kind: 'command_output'
  role: 'system'
  turnId?: string
  stream: string
  command?: string
  phase?: string
  snapshot: boolean
  content: string
}

export type ProjectConversationTaskStatusEntry = {
  id: string
  kind: 'task_status'
  role: 'system'
  turnId?: string
  statusType:
    | 'task_started'
    | 'task_progress'
    | 'task_notification'
    | 'reasoning_updated'
    | 'turn_done'
    | 'error'
    | 'thread_status'
    | 'session_state'
  title: string
  detail?: string
  raw?: Record<string, unknown>
}

export type ProjectConversationInterruptEntry = {
  id: string
  kind: 'interrupt'
  role: 'system'
  interruptId: string
  provider: string
  interruptKind: string
  payload: Record<string, unknown>
  options: { id: string; label: string }[]
  status: 'pending' | 'resolved'
  decision?: string
}

export type ProjectConversationTranscriptEntry =
  | ProjectConversationTextEntry
  | ProjectConversationDiffEntry
  | ProjectConversationToolCallEntry
  | ProjectConversationCommandOutputEntry
  | ProjectConversationTaskStatusEntry
  | ProjectConversationInterruptEntry

export function appendProjectConversationTranscriptEntry(
  entries: ProjectConversationTranscriptEntry[],
  entry: ProjectConversationTranscriptEntry,
) {
  const last = entries.at(-1)
  if (
    last &&
    last.kind === 'command_output' &&
    entry.kind === 'command_output' &&
    last.turnId === entry.turnId &&
    last.stream === entry.stream &&
    last.phase === entry.phase
  ) {
    return [
      ...entries.slice(0, -1),
      {
        ...last,
        command: entry.command || last.command,
        snapshot: entry.snapshot,
        content: entry.snapshot ? entry.content : `${last.content}${entry.content}`,
      } satisfies ProjectConversationCommandOutputEntry,
    ]
  }

  return [...entries, entry]
}

export function createProjectConversationInterruptEntry(params: {
  id: string
  interruptId: string
  provider: string
  interruptKind: string
  payload: Record<string, unknown>
  options: { id: string; label: string }[]
}) {
  return {
    id: params.id,
    kind: 'interrupt',
    role: 'system',
    interruptId: params.interruptId,
    provider: params.provider,
    interruptKind: params.interruptKind,
    payload: params.payload,
    options: params.options,
    status: 'pending',
  } satisfies ProjectConversationInterruptEntry
}

export function createProjectConversationTaskStatusEntry(params: {
  id: string
  turnId?: string
  statusType: ProjectConversationTaskStatusEntry['statusType']
  title: string
  detail?: string
  raw?: Record<string, unknown>
}) {
  return {
    id: params.id,
    kind: 'task_status',
    role: 'system',
    turnId: params.turnId,
    statusType: params.statusType,
    title: params.title,
    detail: params.detail,
    raw: params.raw,
  } satisfies ProjectConversationTaskStatusEntry
}
