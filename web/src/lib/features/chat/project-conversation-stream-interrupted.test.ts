import { describe, expect, it, vi } from 'vitest'
import { handleProjectConversationStreamEvent } from './project-conversation-stream'

function createHandlers() {
  return {
    appendAssistantChunk: vi.fn(),
    finalizeAssistantEntry: vi.fn(),
    appendActionProposal: vi.fn(),
    appendDiff: vi.fn(),
    appendToolCall: vi.fn(),
    appendCommandOutput: vi.fn(),
    appendTaskStatus: vi.fn(),
    confirmActionResult: vi.fn(),
    appendInterrupt: vi.fn(),
    resolveInterrupt: vi.fn(),
    setConversationId: vi.fn(),
    setPending: vi.fn(),
    setPhase: vi.fn(),
    onError: vi.fn(),
  }
}

describe('handleProjectConversationStreamEvent interrupted terminal states', () => {
  it('treats user-stopped interrupted events as terminal without duplicating a status entry', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'interrupted',
        payload: {
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          message: 'Turn stopped by user.',
          reason: 'stopped_by_user',
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledOnce()
    expect(handlers.appendTaskStatus).not.toHaveBeenCalled()
    expect(handlers.setPending).toHaveBeenCalledWith(false)
  })

  it('records non-user interrupted events as terminal interrupted status entries', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'interrupted',
        payload: {
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          message: 'approval required',
          reason: 'waiting_on_approval',
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledOnce()
    expect(handlers.appendTaskStatus).toHaveBeenCalledWith({
      statusType: 'interrupted',
      title: 'Turn interrupted',
      detail: 'approval required',
      raw: {
        conversation_id: 'conversation-1',
        turn_id: 'turn-1',
        reason: 'waiting_on_approval',
      },
    })
    expect(handlers.setPending).toHaveBeenCalledWith(false)
  })
})
