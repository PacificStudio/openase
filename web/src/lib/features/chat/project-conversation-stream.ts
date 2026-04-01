import type { ProjectConversationStreamEvent } from '$lib/api/chat'
import type { ChatActionExecutionResult } from './action-proposal-executor'
import {
  isActionProposalPayload,
  isDiffPayload,
  isTextPayload,
  mapProjectConversationTaskEntry,
} from './project-conversation-transcript-state'

type ProjectConversationStreamHandlers = {
  appendAssistantChunk: (content: string) => void
  finalizeAssistantEntry: () => void
  appendActionProposal: (entryId: string | undefined, payload: Record<string, unknown>) => void
  appendDiff: (entryId: string | undefined, payload: Record<string, unknown>) => void
  appendToolCall: (payload: { tool: string; arguments: unknown }) => void
  appendCommandOutput: (payload: {
    stream: string
    phase?: string
    snapshot: boolean
    content: string
  }) => void
  appendTaskStatus: (payload: {
    statusType: 'task_started' | 'task_progress' | 'task_notification' | 'turn_done' | 'error'
    title: string
    detail?: string
  }) => void
  confirmActionResult: (entryId: string, results: ChatActionExecutionResult[]) => void
  appendInterrupt: (payload: {
    interruptId: string
    provider: string
    kind: string
    payload: Record<string, unknown>
    options: { id: string; label: string }[]
  }) => void
  resolveInterrupt: (interruptId: string, decision?: string) => void
  setConversationId: (conversationId: string) => void
  setPending: (value: boolean) => void
  setPhase: (phase: 'awaiting_interrupt') => void
  onError: (message: string) => void
}

export function handleProjectConversationStreamEvent(
  event: ProjectConversationStreamEvent,
  handlers: ProjectConversationStreamHandlers,
) {
  if (event.kind === 'session') {
    handlers.setConversationId(event.payload.conversationId)
    return
  }

  if (event.kind === 'message') {
    const payload = event.payload
    if (isTextPayload(payload)) {
      handlers.appendAssistantChunk(payload.content)
      return
    }

    handlers.finalizeAssistantEntry()
    if (isActionProposalPayload(payload)) {
      handlers.appendActionProposal(payload.entryId, payload)
      return
    }

    if (isDiffPayload(payload)) {
      handlers.appendDiff(payload.entryId, payload)
      return
    }

    const taskEntry = mapProjectConversationTaskEntry({
      id: '',
      type: payload.type,
      raw: payload.raw,
    })
    if (taskEntry?.kind === 'tool_call') {
      handlers.appendToolCall({
        tool: taskEntry.tool,
        arguments: taskEntry.arguments,
      })
      return
    }

    if (taskEntry?.kind === 'command_output') {
      handlers.appendCommandOutput({
        stream: taskEntry.stream,
        phase: taskEntry.phase,
        snapshot: taskEntry.snapshot,
        content: taskEntry.content,
      })
      return
    }

    if (taskEntry?.kind === 'task_status') {
      handlers.appendTaskStatus({
        statusType: taskEntry.statusType,
        title: taskEntry.title,
        detail: taskEntry.detail,
      })
      return
    }

    if (payload.type === 'action_result') {
      const resultPayload = (
        payload.raw as { payload?: { entry_id?: string; results?: unknown[] } }
      )?.payload
      const resultEntryId = resultPayload?.entry_id
      const results = (resultPayload?.results ?? []) as ChatActionExecutionResult[]
      if (resultEntryId) {
        handlers.confirmActionResult(resultEntryId, results)
      }
    }
    return
  }

  if (event.kind === 'interrupt_requested') {
    handlers.finalizeAssistantEntry()
    handlers.setPhase('awaiting_interrupt')
    handlers.appendInterrupt(event.payload)
    return
  }

  if (event.kind === 'interrupt_resolved') {
    handlers.resolveInterrupt(event.payload.interruptId, event.payload.decision)
    handlers.setPending(true)
    return
  }

  if (event.kind === 'turn_done') {
    handlers.finalizeAssistantEntry()
    handlers.appendTaskStatus({
      statusType: 'turn_done',
      title: 'Turn completed',
      detail:
        typeof event.payload.costUSD === 'number'
          ? `Cost: $${event.payload.costUSD.toFixed(2)}`
          : undefined,
    })
    handlers.setPending(false)
    return
  }

  handlers.finalizeAssistantEntry()
  handlers.appendTaskStatus({
    statusType: 'error',
    title: 'Turn failed',
    detail: event.payload.message,
  })
  handlers.setPending(false)
  handlers.onError(event.payload.message)
}
