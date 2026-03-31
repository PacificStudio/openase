import { beforeEach, describe, expect, it, vi } from 'vitest'

const { consumeEventStream } = vi.hoisted(() => ({
  consumeEventStream: vi.fn(),
}))

vi.mock('./sse', () => ({
  consumeEventStream,
}))

import { streamChatTurn } from './chat'

describe('streamChatTurn', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        body: {} as ReadableStream<Uint8Array>,
      }),
    )
  })

  it('parses structured diff message payloads distinctly from text', async () => {
    consumeEventStream.mockImplementation(async (_body, onFrame) => {
      onFrame({
        event: 'message',
        data: JSON.stringify({
          type: 'diff',
          file: 'harness content',
          hunks: [
            {
              old_start: 1,
              old_lines: 1,
              new_start: 1,
              new_lines: 2,
              lines: [
                { op: 'context', text: '---' },
                { op: 'add', text: 'new line' },
              ],
            },
          ],
        }),
      })
    })

    const events: unknown[] = []
    await streamChatTurn(
      {
        message: 'Help me tighten this harness.',
        source: 'harness_editor',
        context: {
          projectId: 'project-1',
          workflowId: 'workflow-1',
        },
      },
      {
        onEvent: (event) => {
          events.push(event)
        },
      },
    )

    expect(events).toEqual([
      {
        kind: 'message',
        payload: {
          type: 'diff',
          file: 'harness content',
          hunks: [
            {
              oldStart: 1,
              oldLines: 1,
              newStart: 1,
              newLines: 2,
              lines: [
                { op: 'context', text: '---' },
                { op: 'add', text: 'new line' },
              ],
            },
          ],
        },
      },
    ])
  })
})
