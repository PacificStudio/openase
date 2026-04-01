import type { ProjectConversationStreamEvent } from '$lib/api/chat'
import {
  appendProjectConversationAssistantChunk,
  appendProjectConversationTextEntry,
  finalizeProjectConversationAssistantEntry,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import { handleProjectConversationStreamEvent } from './project-conversation-stream'
import { storeProjectConversationId } from './project-conversation-storage'
import { startProjectConversationStream } from './project-conversation-runtime'

export type ProjectConversationPhase =
  | 'idle'
  | 'restoring'
  | 'creating_conversation'
  | 'connecting_stream'
  | 'submitting_turn'
  | 'awaiting_reply'
  | 'awaiting_interrupt'
  | 'resetting'

export type ProjectConversationControllerState = {
  phase: ProjectConversationPhase
  conversationId: string
  entries: ProjectConversationTranscriptEntry[]
  activeAssistantEntryId: string
  abortController: AbortController | null
  entryCounter: number
  operationId: number
  streamId: number
}

export function beginProjectConversationOperation(
  state: ProjectConversationControllerState,
  nextPhase: ProjectConversationPhase,
) {
  state.operationId += 1
  state.phase = nextPhase
  return state.operationId
}

export function isCurrentProjectConversationOperation(
  state: ProjectConversationControllerState,
  operationId: number,
) {
  return operationId === state.operationId
}

export function setProjectConversationIdleIfCurrent(
  state: ProjectConversationControllerState,
  operationId: number,
) {
  if (isCurrentProjectConversationOperation(state, operationId)) {
    state.phase = 'idle'
  }
}

export function invalidateProjectConversationStream(state: ProjectConversationControllerState) {
  state.streamId += 1
  state.abortController?.abort()
  state.abortController = null
}

export function projectConversationHasPendingInterrupt(
  entries: ProjectConversationTranscriptEntry[],
) {
  return entries.some((entry) => entry.kind === 'interrupt' && entry.status === 'pending')
}

export function appendProjectConversationText(
  state: ProjectConversationControllerState,
  role: 'user' | 'assistant' | 'system',
  content: string,
) {
  state.entryCounter += 1
  state.entries = appendProjectConversationTextEntry(state.entries, role, content, {
    entryId: `entry-${state.entryCounter}`,
  })
}

export function finalizeProjectConversationEntry(state: ProjectConversationControllerState) {
  state.entries = finalizeProjectConversationAssistantEntry(
    state.entries,
    state.activeAssistantEntryId,
  )
  state.activeAssistantEntryId = ''
}

export function appendProjectConversationChunk(
  state: ProjectConversationControllerState,
  content: string,
  turnId?: string,
) {
  if (!state.activeAssistantEntryId) {
    state.entryCounter += 1
  }
  const next = appendProjectConversationAssistantChunk({
    entries: state.entries,
    activeAssistantEntryId: state.activeAssistantEntryId,
    nextEntryId: state.activeAssistantEntryId || `entry-${state.entryCounter}`,
    content,
    turnId,
  })
  state.entries = next.entries
  state.activeAssistantEntryId = next.activeAssistantEntryId
}

export function connectProjectConversationStream(params: {
  state: ProjectConversationControllerState
  conversationId: string
  getProjectId: () => string
  getProviderId: () => string
  onError?: (message: string) => void
}) {
  const currentStreamId = params.state.streamId + 1
  const started = startProjectConversationStream({
    conversationId: params.conversationId,
    abortController: params.state.abortController,
    onEvent: (event) => {
      if (currentStreamId !== params.state.streamId) {
        return
      }
      applyProjectConversationStreamEvent({
        state: params.state,
        event,
        getProjectId: params.getProjectId,
        getProviderId: params.getProviderId,
        onError: params.onError,
      })
    },
    onError: (message) => {
      if (currentStreamId !== params.state.streamId) {
        return
      }
      finalizeProjectConversationEntry(params.state)
      if (params.state.phase !== 'awaiting_interrupt') {
        params.state.phase = 'idle'
      }
      params.onError?.(message)
    },
  })

  params.state.streamId = currentStreamId
  params.state.abortController = started.controller
  void started.stream.finally(() => {
    if (
      currentStreamId === params.state.streamId &&
      params.state.abortController === started.controller
    ) {
      params.state.abortController = null
    }
  })
}

export function applyProjectConversationStreamEvent(params: {
  state: ProjectConversationControllerState
  event: ProjectConversationStreamEvent
  getProjectId: () => string
  getProviderId: () => string
  onError?: (message: string) => void
}) {
  handleProjectConversationStreamEvent(params.event, {
    appendAssistantChunk: (content: string) =>
      appendProjectConversationChunk(params.state, content),
    finalizeAssistantEntry: () => finalizeProjectConversationEntry(params.state),
    appendActionProposal: (entryId, payload) => {
      params.state.entries = [
        ...params.state.entries,
        {
          id: entryId ?? `entry-${++params.state.entryCounter}`,
          kind: 'action_proposal',
          role: 'assistant',
          proposal: payload as never,
          status: 'pending',
          results: [],
        },
      ]
    },
    appendDiff: (entryId, payload) => {
      params.state.entries = [
        ...params.state.entries,
        {
          id: entryId ?? `entry-${++params.state.entryCounter}`,
          kind: 'diff',
          role: 'assistant',
          diff: payload as never,
        },
      ]
    },
    confirmActionResult: (entryId, results) => {
      params.state.entries = params.state.entries.map((entry) =>
        entry.kind === 'action_proposal' && entry.id === entryId
          ? { ...entry, status: 'confirmed', results }
          : entry,
      )
    },
    appendInterrupt: (payload) => {
      params.state.entryCounter += 1
      params.state.entries = [
        ...params.state.entries,
        {
          id: `entry-${params.state.entryCounter}`,
          kind: 'interrupt',
          role: 'system',
          interruptId: payload.interruptId,
          provider: payload.provider,
          interruptKind: payload.kind,
          payload: payload.payload,
          options: payload.options,
          status: 'pending',
        },
      ]
    },
    resolveInterrupt: (interruptId, decision) => {
      params.state.entries = params.state.entries.map((entry) =>
        entry.kind === 'interrupt' && entry.interruptId === interruptId
          ? { ...entry, status: 'resolved', decision }
          : entry,
      )
    },
    setConversationId: (conversationId) => {
      params.state.conversationId = conversationId
      storeProjectConversationId(
        params.getProjectId(),
        params.getProviderId(),
        params.state.conversationId,
      )
    },
    setPending: (value) => {
      params.state.phase = value ? 'awaiting_reply' : 'idle'
    },
    setPhase: (phase) => {
      params.state.phase = phase
    },
    onError: (message) => params.onError?.(message),
  })
}
