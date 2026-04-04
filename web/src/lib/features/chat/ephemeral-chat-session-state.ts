import type { ChatSource, ChatStreamEvent } from '$lib/api/chat'
import { formatEphemeralChatUsageSummary } from './session-policy'
import {
  appendAssistantTextChunk,
  createTextTranscriptEntry,
  finalizeAssistantTextChunk,
  mapChatPayloadToTranscriptEntry,
  type EphemeralChatRole,
  type EphemeralChatTranscriptEntry,
} from './transcript'

export type EphemeralChatSessionState = {
  pending: boolean
  sessionId: string
  entries: EphemeralChatTranscriptEntry[]
  entryCounter: number
  activeAssistantEntryId: string
}

export function appendEphemeralChatEntry(
  state: EphemeralChatSessionState,
  role: EphemeralChatRole,
  content: string,
  options?: { streaming?: boolean },
) {
  state.entryCounter += 1
  state.entries = [
    ...state.entries,
    createTextTranscriptEntry(`entry-${state.entryCounter}`, role, content, options),
  ]
}

export function finalizeEphemeralAssistantText(state: EphemeralChatSessionState) {
  const update = finalizeAssistantTextChunk({
    entries: state.entries,
    activeAssistantEntryId: state.activeAssistantEntryId,
    entryCounter: state.entryCounter,
  })
  state.entries = update.entries
  state.activeAssistantEntryId = update.activeAssistantEntryId
  state.entryCounter = update.entryCounter
}

export function handleEphemeralChatStreamEvent(
  state: EphemeralChatSessionState,
  event: ChatStreamEvent,
  input: {
    source: ChatSource
    onError?: (message: string) => void
  },
) {
  if (event.kind === 'session') {
    state.sessionId = event.payload.sessionId
    return
  }
  if (event.kind === 'done') {
    state.sessionId = event.payload.sessionId
    finalizeEphemeralAssistantText(state)
    appendEphemeralChatEntry(
      state,
      'system',
      formatEphemeralChatUsageSummary(input.source, event.payload),
    )
    state.pending = false
    return
  }
  if (event.kind === 'error') {
    finalizeEphemeralAssistantText(state)
    input.onError?.(event.payload.message)
    state.pending = false
    return
  }

  const payload = event.payload
  if (payload.type === 'text') {
    const textPayload = payload as Extract<typeof payload, { type: 'text' }>
    const update = appendAssistantTextChunk({
      entries: state.entries,
      activeAssistantEntryId: state.activeAssistantEntryId,
      entryCounter: state.entryCounter,
      content: textPayload.content,
    })
    state.entries = update.entries
    state.activeAssistantEntryId = update.activeAssistantEntryId
    state.entryCounter = update.entryCounter
    return
  }

  finalizeEphemeralAssistantText(state)
  state.entryCounter += 1
  state.entries = [
    ...state.entries,
    mapChatPayloadToTranscriptEntry(`entry-${state.entryCounter}`, payload),
  ]
}

export function clearEphemeralChatSessionState(
  state: EphemeralChatSessionState,
  options: { clearEntries: boolean },
) {
  state.activeAssistantEntryId = ''
  state.pending = false
  state.sessionId = ''
  if (options.clearEntries) {
    state.entries = []
  }
}
