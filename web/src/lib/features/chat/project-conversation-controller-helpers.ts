import type {
  ProjectConversationStreamEvent,
  ProjectConversationWorkspaceDiff,
} from '$lib/api/chat'
import {
  appendProjectConversationAssistantChunk,
  appendProjectConversationTranscriptEntry,
  appendProjectConversationTextEntry,
  createProjectConversationActionProposalEntry,
  createProjectConversationDiffEntry,
  createProjectConversationInterruptEntry,
  finalizeProjectConversationAssistantEntry,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import { handleProjectConversationStreamEvent } from './project-conversation-stream'
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
  workspaceDiff: ProjectConversationWorkspaceDiff | null
  workspaceDiffLoading: boolean
  workspaceDiffError: string
  workspaceDiffRequestId: number
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

export function clearProjectConversationSelection(state: ProjectConversationControllerState) {
  state.conversationId = ''
  state.entries = []
  state.workspaceDiff = null
  state.workspaceDiffLoading = false
  state.workspaceDiffError = ''
  state.activeAssistantEntryId = ''
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

export function appendProjectConversationStructuredEntry(
  state: ProjectConversationControllerState,
  entry: ProjectConversationTranscriptEntry,
) {
  state.entries = appendProjectConversationTranscriptEntry(state.entries, entry)
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
  onError?: (message: string) => void
  onEvent?: (event: ProjectConversationStreamEvent) => void
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
        onError: params.onError,
      })
      params.onEvent?.(event)
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
  onError?: (message: string) => void
}) {
  handleProjectConversationStreamEvent(params.event, {
    appendAssistantChunk: (content: string) =>
      appendProjectConversationChunk(params.state, content),
    finalizeAssistantEntry: () => finalizeProjectConversationEntry(params.state),
    appendActionProposal: (entryId, payload) => {
      appendProjectConversationStructuredEntry(
        params.state,
        createProjectConversationActionProposalEntry({
          id: entryId ?? `entry-${++params.state.entryCounter}`,
          proposal: payload as never,
        }),
      )
    },
    appendDiff: (entryId, payload) => {
      appendProjectConversationStructuredEntry(
        params.state,
        createProjectConversationDiffEntry({
          id: entryId ?? `entry-${++params.state.entryCounter}`,
          payload,
        }),
      )
    },
    appendToolCall: (payload) => {
      params.state.entryCounter += 1
      appendProjectConversationStructuredEntry(params.state, {
        id: `entry-${params.state.entryCounter}`,
        kind: 'tool_call',
        role: 'system',
        tool: payload.tool,
        arguments: payload.arguments,
      })
    },
    appendCommandOutput: (payload) => {
      params.state.entryCounter += 1
      appendProjectConversationStructuredEntry(params.state, {
        id: `entry-${params.state.entryCounter}`,
        kind: 'command_output',
        role: 'system',
        stream: payload.stream,
        phase: payload.phase,
        snapshot: payload.snapshot,
        content: payload.content,
      })
    },
    appendTaskStatus: (payload) => {
      params.state.entryCounter += 1
      appendProjectConversationStructuredEntry(params.state, {
        id: `entry-${params.state.entryCounter}`,
        kind: 'task_status',
        role: 'system',
        statusType: payload.statusType,
        title: payload.title,
        detail: payload.detail,
      })
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
      appendProjectConversationStructuredEntry(
        params.state,
        createProjectConversationInterruptEntry({
          id: `entry-${params.state.entryCounter}`,
          interruptId: payload.interruptId,
          provider: payload.provider,
          interruptKind: payload.kind,
          payload: payload.payload,
          options: payload.options,
        }),
      )
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
