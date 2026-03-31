import type {
  ChatActionProposalPayload,
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
  | ProjectConversationInterruptEntry

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
      transcript.push({
        id: entry.id,
        kind: 'action_proposal',
        role: 'assistant',
        proposal: {
          ...(entry.payload as unknown as ChatActionProposalPayload),
          type: 'action_proposal',
          entryId: entry.id,
        },
        status: 'pending',
        results: [],
      })
      continue
    }

    if (entry.kind === 'diff') {
      transcript.push({
        id: entry.id,
        kind: 'diff',
        role: 'assistant',
        diff: {
          ...(entry.payload as unknown as ChatDiffPayload),
          type: 'diff',
          entryId: entry.id,
        },
      })
      continue
    }

    if (entry.kind === 'interrupt') {
      transcript.push({
        id: entry.id,
        kind: 'interrupt',
        role: 'system',
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
        status: 'pending',
      })
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
