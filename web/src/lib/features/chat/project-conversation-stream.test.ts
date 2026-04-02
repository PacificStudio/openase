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

describe('handleProjectConversationStreamEvent', () => {
  it('maps runtime task messages into structured Project AI entries', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'message',
        payload: {
          type: 'task_notification',
          raw: {
            tool: 'functions.exec_command',
            arguments: { cmd: 'git status' },
          },
        },
      },
      handlers,
    )

    handleProjectConversationStreamEvent(
      {
        kind: 'message',
        payload: {
          type: 'task_progress',
          raw: {
            stream: 'command',
            phase: 'stdout',
            snapshot: false,
            text: 'M web/src/app.ts\n',
          },
        },
      },
      handlers,
    )

    handleProjectConversationStreamEvent(
      {
        kind: 'turn_done',
        payload: {
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          costUSD: 0.03,
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledTimes(3)
    expect(handlers.appendToolCall).toHaveBeenCalledWith({
      tool: 'functions.exec_command',
      arguments: { cmd: 'git status' },
    })
    expect(handlers.appendCommandOutput).toHaveBeenCalledWith({
      stream: 'command',
      phase: 'stdout',
      snapshot: false,
      content: 'M web/src/app.ts\n',
    })
    expect(handlers.appendTaskStatus).toHaveBeenLastCalledWith({
      statusType: 'turn_done',
      title: 'Turn completed',
      detail: 'Cost: $0.03',
      raw: undefined,
    })
    expect(handlers.setPending).toHaveBeenLastCalledWith(false)
  })

  it('records a failure status before surfacing the stream error', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'error',
        payload: {
          message: 'codex chat turn failed',
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledOnce()
    expect(handlers.appendTaskStatus).toHaveBeenCalledWith({
      statusType: 'error',
      title: 'Turn failed',
      detail: 'codex chat turn failed',
      raw: undefined,
    })
    expect(handlers.setPending).toHaveBeenCalledWith(false)
    expect(handlers.onError).toHaveBeenCalledWith('codex chat turn failed')
  })

  it('preserves structured task status payload details for rendering', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'message',
        payload: {
          type: 'task_progress',
          raw: {
            status: 'running',
            command: 'pnpm test',
            file: 'README.md',
            patch: '@@ -1 +1 @@\n-old\n+new',
          },
        },
      },
      handlers,
    )

    expect(handlers.appendTaskStatus).toHaveBeenCalledWith({
      statusType: 'task_progress',
      title: 'Task progress',
      detail: 'Status: running',
      raw: {
        status: 'running',
        command: 'pnpm test',
        file: 'README.md',
        patch: '@@ -1 +1 @@\n-old\n+new',
      },
    })
  })
})
