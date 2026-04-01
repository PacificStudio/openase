import type {
  ChatActionProposalPayload,
  ChatDiffHunk,
  ChatDiffPayload,
  ChatMessagePayload,
  ProjectConversationEntry,
} from '$lib/api/chat'
import type { ChatActionExecutionResult } from './action-proposal-executor'

type ProjectConversationRole = 'user' | 'assistant' | 'system'

type ProjectConversationTextEntry = {
  id: string
  kind: 'text'
  role: ProjectConversationRole
  turnId?: string
  content: string
  streaming: boolean
}

type ProjectConversationActionProposalEntry = {
  id: string
  kind: 'action_proposal'
  role: 'assistant'
  proposal: ChatActionProposalPayload
  status: 'pending' | 'executing' | 'confirmed'
  results: ChatActionExecutionResult[]
}

type ProjectConversationDiffEntry = {
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
  phase?: string
  snapshot: boolean
  content: string
}

export type ProjectConversationTaskStatusEntry = {
  id: string
  kind: 'task_status'
  role: 'system'
  turnId?: string
  statusType: 'task_started' | 'task_progress' | 'task_notification' | 'turn_done' | 'error'
  title: string
  detail?: string
}

type ProjectConversationInterruptEntry = {
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
  | ProjectConversationActionProposalEntry
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
        snapshot: entry.snapshot,
        content: entry.snapshot ? entry.content : `${last.content}${entry.content}`,
      } satisfies ProjectConversationCommandOutputEntry,
    ]
  }

  return [...entries, entry]
}

export function createProjectConversationActionProposalEntry(params: {
  id: string
  proposal: ChatActionProposalPayload
}) {
  return {
    id: params.id,
    kind: 'action_proposal',
    role: 'assistant',
    proposal: params.proposal,
    status: 'pending',
    results: [],
  } satisfies ProjectConversationActionProposalEntry
}

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
}) {
  return {
    id: params.id,
    kind: 'task_status',
    role: 'system',
    turnId: params.turnId,
    statusType: params.statusType,
    title: params.title,
    detail: params.detail,
  } satisfies ProjectConversationTaskStatusEntry
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
        phase: readString(raw, 'phase') || undefined,
        snapshot: readBoolean(raw, 'snapshot'),
        content,
      }
    }
  }

  if (params.type === 'task_started') {
    return createProjectConversationTaskStatusEntry({
      id: params.id,
      turnId: params.turnId,
      statusType: 'task_started',
      title: 'Task started',
      detail: buildTaskDetail(raw),
    })
  }

  if (params.type === 'task_progress') {
    return createProjectConversationTaskStatusEntry({
      id: params.id,
      turnId: params.turnId,
      statusType: 'task_progress',
      title: 'Task progress',
      detail: buildTaskDetail(raw),
    })
  }

  if (params.type === 'task_notification') {
    return createProjectConversationTaskStatusEntry({
      id: params.id,
      turnId: params.turnId,
      statusType: 'task_notification',
      title: 'Task notification',
      detail: buildTaskDetail(raw),
    })
  }

  return null
}

export function createProjectConversationTurnDoneEntry(params: {
  id: string
  turnId?: string
  costUSD?: number
}) {
  const costText =
    typeof params.costUSD === 'number' ? `Cost: $${params.costUSD.toFixed(2)}` : undefined

  return createProjectConversationTaskStatusEntry({
    id: params.id,
    turnId: params.turnId,
    statusType: 'turn_done',
    title: 'Turn completed',
    detail: costText,
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

export function appendProjectConversationTextEntry(
  entries: ProjectConversationTranscriptEntry[],
  role: ProjectConversationRole,
  content: string,
  options?: { turnId?: string; streaming?: boolean; entryId?: string },
) {
  return [
    ...entries,
    {
      id: options?.entryId ?? '',
      kind: 'text',
      role,
      turnId: options?.turnId,
      content,
      streaming: options?.streaming ?? false,
    } satisfies ProjectConversationTextEntry,
  ]
}

export function finalizeProjectConversationAssistantEntry(
  entries: ProjectConversationTranscriptEntry[],
  activeAssistantEntryId: string,
) {
  if (!activeAssistantEntryId) {
    return entries
  }

  return entries.map((entry) =>
    entry.kind === 'text' && entry.id === activeAssistantEntryId
      ? { ...entry, streaming: false }
      : entry,
  )
}

export function appendProjectConversationAssistantChunk(params: {
  entries: ProjectConversationTranscriptEntry[]
  activeAssistantEntryId: string
  nextEntryId: string
  content: string
  turnId?: string
}) {
  if (!params.activeAssistantEntryId) {
    return {
      entries: [
        ...params.entries,
        {
          id: params.nextEntryId,
          kind: 'text',
          role: 'assistant',
          turnId: params.turnId,
          content: params.content,
          streaming: true,
        } satisfies ProjectConversationTextEntry,
      ],
      activeAssistantEntryId: params.nextEntryId,
    }
  }

  return {
    entries: params.entries.map((entry) =>
      entry.kind === 'text' && entry.id === params.activeAssistantEntryId
        ? { ...entry, content: `${entry.content}${params.content}`, streaming: true }
        : entry,
    ),
    activeAssistantEntryId: params.activeAssistantEntryId,
  }
}

export function mapPersistedEntries(
  entries: ProjectConversationEntry[],
): ProjectConversationTranscriptEntry[] {
  const transcript: ProjectConversationTranscriptEntry[] = []

  for (const entry of entries) {
    if (entry.kind === 'user_message') {
      transcript.push({
        id: entry.id,
        kind: 'text',
        role: 'user',
        turnId: entry.turnId,
        content: String(entry.payload.content ?? ''),
        streaming: false,
      })
      continue
    }

    if (entry.kind === 'assistant_text_delta') {
      const last = transcript.at(-1)
      if (
        last &&
        last.kind === 'text' &&
        last.role === 'assistant' &&
        last.turnId === entry.turnId
      ) {
        last.content = `${last.content}${String(entry.payload.content ?? '')}`
      } else {
        transcript.push({
          id: entry.id,
          kind: 'text',
          role: 'assistant',
          turnId: entry.turnId,
          content: String(entry.payload.content ?? ''),
          streaming: false,
        })
      }
      continue
    }

    if (entry.kind === 'action_proposal') {
      transcript.push(
        createProjectConversationActionProposalEntry({
          id: entry.id,
          proposal: {
            ...(entry.payload as unknown as ChatActionProposalPayload),
            type: 'action_proposal',
            entryId: entry.id,
          },
        }),
      )
      continue
    }

    if (entry.kind === 'diff') {
      transcript.push(createProjectConversationDiffEntry({ id: entry.id, payload: entry.payload }))
      continue
    }

    if (entry.kind === 'system') {
      const derived = mapProjectConversationTaskEntry({
        id: entry.id,
        turnId: entry.turnId,
        type: String(entry.payload.type ?? ''),
        raw: entry.payload.raw,
      })
      if (derived) {
        transcript.splice(
          0,
          transcript.length,
          ...appendProjectConversationTranscriptEntry(transcript, derived),
        )
      }
      continue
    }

    if (entry.kind === 'interrupt') {
      transcript.push(
        createProjectConversationInterruptEntry({
          id: entry.id,
          interruptId: String(entry.payload.interrupt_id ?? ''),
          provider: String(entry.payload.provider ?? 'codex'),
          interruptKind: String(entry.payload.kind ?? ''),
          payload: (entry.payload.payload as Record<string, unknown>) ?? {},
          options: ((entry.payload.options as { id: string; label: string }[]) ?? []).map(
            (option) => ({
              id: option.id,
              label: option.label,
            }),
          ),
        }),
      )
      continue
    }

    if (entry.kind === 'interrupt_resolution') {
      const interruptId = String(entry.payload.interrupt_id ?? '')
      const decision = entry.payload.decision ? String(entry.payload.decision) : undefined
      for (const transcriptEntry of transcript) {
        if (transcriptEntry.kind === 'interrupt' && transcriptEntry.interruptId === interruptId) {
          transcriptEntry.status = 'resolved'
          transcriptEntry.decision = decision
        }
      }
      continue
    }

    if (entry.kind === 'action_result') {
      const payload = entry.payload as {
        entry_id?: string
        results?: ChatActionExecutionResult[]
      }
      if (payload.entry_id) {
        for (const transcriptEntry of transcript) {
          if (
            transcriptEntry.kind === 'action_proposal' &&
            transcriptEntry.id === payload.entry_id
          ) {
            transcriptEntry.status = 'confirmed'
            transcriptEntry.results = payload.results ?? []
          }
        }
      }
    }
  }

  return transcript
}

export function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

export function isActionProposalPayload(
  payload: ChatMessagePayload,
): payload is ChatActionProposalPayload {
  return payload.type === 'action_proposal'
}

export function isDiffPayload(payload: ChatMessagePayload): payload is ChatDiffPayload {
  return payload.type === 'diff'
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
              if (!record) {
                return null
              }

              const op = readString(record, 'op')
              const text = readString(record, 'text')
              if (!op || text == null) {
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
