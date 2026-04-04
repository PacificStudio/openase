import type { ProjectConversationStreamEvent } from '$lib/api/chat'
import {
  createProjectConversationDiffEntriesFromUnifiedDiff,
  isDiffPayload,
  isTextPayload,
  mapProjectConversationTaskEntry,
} from './project-conversation-transcript-state'

type ProjectConversationStreamHandlers = {
  appendAssistantChunk: (content: string) => void
  finalizeAssistantEntry: () => void
  appendDiff: (entryId: string | undefined, payload: Record<string, unknown>) => void
  appendToolCall: (payload: { tool: string; arguments: unknown }) => void
  appendCommandOutput: (payload: {
    stream: string
    command?: string
    phase?: string
    snapshot: boolean
    content: string
  }) => void
  appendTaskStatus: (payload: {
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
  }) => void
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
    if (isDiffPayload(payload)) {
      handlers.appendDiff(payload.entryId, payload)
      return
    }
    if (!('raw' in payload)) {
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
        command: taskEntry.command,
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
        raw: taskEntry.raw,
      })
      return
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

  if (event.kind === 'diff_updated') {
    handlers.finalizeAssistantEntry()
    const diffEntries = createProjectConversationDiffEntriesFromUnifiedDiff({
      idBase: event.payload.entryId ?? '',
      diff: event.payload.diff,
    })
    for (const diffEntry of diffEntries) {
      handlers.appendDiff(diffEntry.id || undefined, diffEntry.diff)
    }
    return
  }

  if (event.kind === 'reasoning_updated') {
    handlers.finalizeAssistantEntry()
    handlers.appendTaskStatus({
      statusType: 'reasoning_updated',
      title: 'Reasoning update',
      detail: event.payload.delta || `Kind: ${event.payload.kind.replace(/_/g, ' ')}`,
      raw: {
        thread_id: event.payload.threadId,
        turn_id: event.payload.turnId,
        item_id: event.payload.itemId,
        kind: event.payload.kind,
        delta: event.payload.delta,
        summary_index: event.payload.summaryIndex,
        content_index: event.payload.contentIndex,
        entry_id: event.payload.entryId,
      },
    })
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
