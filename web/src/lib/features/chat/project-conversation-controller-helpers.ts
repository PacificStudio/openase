import type {
  ProjectConversationStreamEvent,
  ProjectConversationWorkspaceDiff,
} from '$lib/api/chat'
import {
  appendProjectConversationAssistantChunk,
  appendProjectConversationTranscriptEntry,
  appendProjectConversationTextEntry,
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
  projectId: string
  conversationId: string
  resolveState?: () => ProjectConversationControllerState | null
  onStateChanged?: () => void
  onError?: (message: string) => void
  onEvent?: (event: ProjectConversationStreamEvent) => void
  onReconnect?: (streamId: number) => void
  onRetrying?: (streamId: number) => void
  onClosed?: (streamId: number) => void
}) {
  const initialState = params.resolveState?.() ?? params.state
  const currentStreamId = initialState.streamId + 1
  let streamFailed = false,
    receivedNonSessionEvent = false
  const started = startProjectConversationStream({
    projectId: params.projectId,
    conversationId: params.conversationId,
    abortController: initialState.abortController,
    onEvent: (event) => {
      const liveState = params.resolveState?.() ?? params.state
      if (currentStreamId !== liveState.streamId) {
        return
      }
      if (event.kind !== 'session') {
        receivedNonSessionEvent = true
      }
      params.onEvent?.(event)
    },
    onReconnect: () => {
      const liveState = params.resolveState?.() ?? params.state
      if (currentStreamId !== liveState.streamId) return
      params.onReconnect?.(currentStreamId)
    },
    onRetrying: () => {
      const liveState = params.resolveState?.() ?? params.state
      if (currentStreamId !== liveState.streamId) return
      params.onRetrying?.(currentStreamId)
    },
    onError: (message) => {
      const liveState = params.resolveState?.() ?? params.state
      if (currentStreamId !== liveState.streamId) {
        return
      }
      streamFailed = true
      finalizeProjectConversationEntry(liveState)
      if (liveState.phase !== 'awaiting_interrupt') {
        liveState.phase = 'idle'
      }
      params.onStateChanged?.()
      params.onError?.(message)
    },
  })

  const connectedState = params.resolveState?.() ?? params.state
  connectedState.streamId = currentStreamId
  connectedState.abortController = started.controller
  void started.stream.finally(() => {
    const liveState = params.resolveState?.() ?? params.state
    const isCurrentStream =
      currentStreamId === liveState.streamId && liveState.abortController === started.controller
    if (isCurrentStream) {
      liveState.abortController = null
    }
    if (
      isCurrentStream &&
      !started.controller.signal.aborted &&
      !streamFailed &&
      receivedNonSessionEvent
    ) {
      params.onStateChanged?.()
      params.onClosed?.(currentStreamId)
    }
  })
  return started.connected
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
        command: payload.command,
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
        raw: payload.raw,
      })
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
