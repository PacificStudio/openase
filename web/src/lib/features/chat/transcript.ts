import type { ChatBundleDiffPayload, ChatDiffPayload, ChatMessagePayload } from '$lib/api/chat'

export type EphemeralChatRole = 'user' | 'assistant' | 'system'

export type EphemeralChatTextEntry = {
  id: string
  role: EphemeralChatRole
  kind: 'text'
  content: string
  streaming: boolean
}

export type EphemeralChatDiffEntry = {
  id: string
  role: 'assistant'
  kind: 'diff'
  diff: ChatDiffPayload
}

export type EphemeralChatBundleDiffEntry = {
  id: string
  role: 'assistant'
  kind: 'bundle_diff'
  bundleDiff: ChatBundleDiffPayload
}

export type EphemeralChatTranscriptEntry =
  | EphemeralChatTextEntry
  | EphemeralChatDiffEntry
  | EphemeralChatBundleDiffEntry

type AssistantTextUpdate = {
  entries: EphemeralChatTranscriptEntry[]
  activeAssistantEntryId: string
  entryCounter: number
}

export function mapChatPayloadToTranscriptEntry(
  id: string,
  payload: ChatMessagePayload,
): EphemeralChatTranscriptEntry {
  if (isTextPayload(payload)) {
    return {
      id,
      role: 'assistant',
      kind: 'text',
      content: payload.content,
      streaming: false,
    }
  }

  if (isDiffPayload(payload)) {
    return {
      id,
      role: 'assistant',
      kind: 'diff',
      diff: payload,
    }
  }

  if (isBundleDiffPayload(payload)) {
    return {
      id,
      role: 'assistant',
      kind: 'bundle_diff',
      bundleDiff: payload,
    }
  }

  return {
    id,
    role: 'system',
    kind: 'text',
    content: describeSystemMessage(payload.type),
    streaming: false,
  }
}

export function createTextTranscriptEntry(
  id: string,
  role: EphemeralChatRole,
  content: string,
  options?: {
    streaming?: boolean
  },
): EphemeralChatTextEntry {
  return {
    id,
    role,
    kind: 'text',
    content,
    streaming: options?.streaming ?? false,
  }
}

export function isDiffEntry(entry: EphemeralChatTranscriptEntry): entry is EphemeralChatDiffEntry {
  return entry.kind === 'diff'
}

export function isBundleDiffEntry(
  entry: EphemeralChatTranscriptEntry,
): entry is EphemeralChatBundleDiffEntry {
  return entry.kind === 'bundle_diff'
}

export function isTextTranscriptEntry(
  entry: EphemeralChatTranscriptEntry,
): entry is EphemeralChatTextEntry {
  return entry.kind === 'text'
}

export function appendAssistantTextChunk(input: {
  entries: EphemeralChatTranscriptEntry[]
  activeAssistantEntryId: string
  entryCounter: number
  content: string
}): AssistantTextUpdate {
  if (!input.activeAssistantEntryId) {
    const nextEntryCounter = input.entryCounter + 1
    const entryId = `entry-${nextEntryCounter}`
    return {
      entries: [
        ...input.entries,
        createTextTranscriptEntry(entryId, 'assistant', input.content, {
          streaming: true,
        }),
      ],
      activeAssistantEntryId: entryId,
      entryCounter: nextEntryCounter,
    }
  }

  return {
    entries: input.entries.map((entry) => {
      if (!isTextTranscriptEntry(entry) || entry.id !== input.activeAssistantEntryId) {
        return entry
      }

      return {
        ...entry,
        content: `${entry.content}${input.content}`,
        streaming: true,
      }
    }),
    activeAssistantEntryId: input.activeAssistantEntryId,
    entryCounter: input.entryCounter,
  }
}

export function finalizeAssistantTextChunk(input: {
  entries: EphemeralChatTranscriptEntry[]
  activeAssistantEntryId: string
  entryCounter: number
}): AssistantTextUpdate {
  if (!input.activeAssistantEntryId) {
    return input
  }

  return {
    entries: input.entries.map((entry) => {
      if (!isTextTranscriptEntry(entry) || entry.id !== input.activeAssistantEntryId) {
        return entry
      }

      return {
        ...entry,
        streaming: false,
      }
    }),
    activeAssistantEntryId: '',
    entryCounter: input.entryCounter,
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function describeSystemMessage(type: string) {
  switch (type) {
    case 'task_started':
      return 'Assistant started a background task.'
    case 'task_progress':
      return 'Assistant reported task progress.'
    case 'task_notification':
      return 'Assistant emitted a task notification.'
    default:
      return `Assistant emitted ${type}.`
  }
}

export function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

function isDiffPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'diff' }> {
  return payload.type === 'diff'
}

function isBundleDiffPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'bundle_diff' }> {
  return payload.type === 'bundle_diff'
}
