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
      command: undefined,
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

  it('maps codex thread and claude session status messages into structured status entries', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'message',
        payload: {
          type: 'thread_status',
          raw: {
            anchor_kind: 'thread',
            thread_id: 'thread-1',
            status: 'waitingOnApproval',
            active_flags: ['waitingOnApproval'],
          },
        },
      },
      handlers,
    )

    handleProjectConversationStreamEvent(
      {
        kind: 'message',
        payload: {
          type: 'session_state',
          raw: {
            anchor_kind: 'session',
            status: 'requires_action',
            detail: 'approval required',
            active_flags: ['requires_action'],
          },
        },
      },
      handlers,
    )

    expect(handlers.appendTaskStatus).toHaveBeenNthCalledWith(1, {
      statusType: 'thread_status',
      title: 'Codex thread status',
      detail: 'waitingOnApproval · waitingOnApproval',
      raw: {
        anchor_kind: 'thread',
        thread_id: 'thread-1',
        status: 'waitingOnApproval',
        active_flags: ['waitingOnApproval'],
      },
    })
    expect(handlers.appendTaskStatus).toHaveBeenNthCalledWith(2, {
      statusType: 'session_state',
      title: 'Claude session status',
      detail: 'requires_action · approval required · requires_action',
      raw: {
        anchor_kind: 'session',
        status: 'requires_action',
        detail: 'approval required',
        active_flags: ['requires_action'],
      },
    })
  })

  it('maps reasoning updates from the stream into task status entries', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'reasoning_updated',
        payload: {
          threadId: 'thread-1',
          turnId: 'turn-1',
          itemId: 'item-1',
          kind: 'summary_text_delta',
          delta: 'Check the failing tests first.',
          summaryIndex: 0,
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledOnce()
    expect(handlers.appendTaskStatus).toHaveBeenCalledWith({
      statusType: 'reasoning_updated',
      title: 'Reasoning update',
      detail: 'Check the failing tests first.',
      raw: {
        thread_id: 'thread-1',
        turn_id: 'turn-1',
        item_id: 'item-1',
        kind: 'summary_text_delta',
        delta: 'Check the failing tests first.',
        summary_index: 0,
        content_index: undefined,
        entry_id: undefined,
      },
    })
  })

  it('maps diff updates from the stream into structured diff entries', () => {
    const handlers = createHandlers()

    handleProjectConversationStreamEvent(
      {
        kind: 'diff_updated',
        payload: {
          threadId: 'thread-1',
          turnId: 'turn-1',
          diff: [
            'diff --git a/app.ts b/app.ts',
            '@@ -1,1 +1,2 @@',
            '-old line',
            '+new line',
            '+second line',
          ].join('\n'),
          entryId: 'entry-diff',
        },
      },
      handlers,
    )

    expect(handlers.finalizeAssistantEntry).toHaveBeenCalledOnce()
    expect(handlers.appendDiff).toHaveBeenCalledWith('entry-diff', {
      type: 'diff',
      entryId: 'entry-diff',
      file: 'app.ts',
      hunks: [
        {
          oldStart: 1,
          oldLines: 1,
          newStart: 1,
          newLines: 2,
          lines: [
            { op: 'remove', text: 'old line' },
            { op: 'add', text: 'new line' },
            { op: 'add', text: 'second line' },
          ],
        },
      ],
    })
  })
})
