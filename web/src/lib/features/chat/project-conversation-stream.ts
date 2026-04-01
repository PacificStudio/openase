import type { ProjectConversationStreamEvent } from '$lib/api/chat'
import type { ChatActionExecutionResult } from './action-proposal-executor'
import {
  isActionProposalPayload,
  isDiffPayload,
  isTextPayload,
} from './project-conversation-transcript-state'

type ProjectConversationStreamHandlers = {
  appendAssistantChunk: (content: string) => void
  finalizeAssistantEntry: () => void
  appendActionProposal: (entryId: string | undefined, payload: Record<string, unknown>) => void
  appendDiff: (entryId: string | undefined, payload: Record<string, unknown>) => void
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
    handlers.setPending(false)
    return
  }

  handlers.finalizeAssistantEntry()
  handlers.setPending(false)
  handlers.onError(event.payload.message)
}
