import type { ChatDiffPayload, ChatMessagePayload, ProjectConversationEntry } from '$lib/api/chat'
import {
  createProjectConversationDiffEntriesFromUnifiedDiff,
  createProjectConversationDiffEntry,
  mapProjectConversationTaskEntry,
} from './project-conversation-transcript-parsers'
import {
  appendProjectConversationTranscriptEntry,
  createProjectConversationInterruptEntry,
  type ProjectConversationRole,
  type ProjectConversationTextEntry,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'

export {
  appendProjectConversationTranscriptEntry,
  createProjectConversationInterruptEntry,
} from './project-conversation-transcript-types'
export {
  createProjectConversationDiffEntriesFromUnifiedDiff,
  createProjectConversationDiffEntry,
  mapProjectConversationTaskEntry,
} from './project-conversation-transcript-parsers'
export type {
  ProjectConversationCommandOutputEntry,
  ProjectConversationDiffEntry,
  ProjectConversationInterruptEntry,
  ProjectConversationTaskStatusEntry,
  ProjectConversationTextEntry,
  ProjectConversationToolCallEntry,
  ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'

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

    if (entry.kind === 'diff') {
      transcript.push(createProjectConversationDiffEntry({ id: entry.id, payload: entry.payload }))
      continue
    }

    if (entry.kind === 'system') {
      if (String(entry.payload.type ?? '') === 'turn_diff_updated') {
        const diffEntries = createProjectConversationDiffEntriesFromUnifiedDiff({
          idBase: entry.id,
          diff: String(entry.payload.diff ?? ''),
        })
        if (diffEntries.length > 0) {
          let nextTranscript = transcript
          for (const diffEntry of diffEntries) {
            nextTranscript = appendProjectConversationTranscriptEntry(nextTranscript, diffEntry)
          }
          transcript.splice(0, transcript.length, ...nextTranscript)
          continue
        }
      }

      const derived = mapProjectConversationTaskEntry({
        id: entry.id,
        turnId: entry.turnId,
        type: String(entry.payload.type ?? ''),
        raw:
          entry.payload.raw && typeof entry.payload.raw === 'object'
            ? entry.payload.raw
            : entry.payload,
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
  }

  return transcript
}

export function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

export function isDiffPayload(payload: ChatMessagePayload): payload is ChatDiffPayload {
  return payload.type === 'diff'
}
