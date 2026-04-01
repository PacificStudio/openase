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
          harnessDraft: '---\nworkflow:\n  name: Draft\n---\n',
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
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/chat',
      expect.objectContaining({
        body: JSON.stringify({
          message: 'Help me tighten this harness.',
          source: 'harness_editor',
          provider_id: undefined,
          session_id: undefined,
          context: {
            project_id: 'project-1',
            workflow_id: 'workflow-1',
            ticket_id: undefined,
            harness_draft: '---\nworkflow:\n  name: Draft\n---\n',
          },
        }),
      }),
    )
  })

  it('serializes skill editor chat context', async () => {
    consumeEventStream.mockImplementation(async () => {})

    await streamChatTurn(
      {
        message: 'Tighten this deploy script.',
        source: 'skill_editor',
        providerId: 'provider-1',
        sessionId: 'session-skill-1',
        context: {
          projectId: 'project-1',
          skillId: 'skill-1',
          skillFilePath: 'scripts/redeploy.sh',
          skillFileDraft: '#!/usr/bin/env bash\necho updated\n',
        },
      },
      {
        onEvent: () => {},
      },
    )

    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/chat',
      expect.objectContaining({
        body: JSON.stringify({
          message: 'Tighten this deploy script.',
          source: 'skill_editor',
          provider_id: 'provider-1',
          session_id: 'session-skill-1',
          context: {
            project_id: 'project-1',
            workflow_id: undefined,
            ticket_id: undefined,
            harness_draft: undefined,
            skill_id: 'skill-1',
            skill_file_path: 'scripts/redeploy.sh',
            skill_file_draft: '#!/usr/bin/env bash\necho updated\n',
          },
        }),
      }),
    )
  })

  it('parses structured bundle diff payloads distinctly from text', async () => {
    consumeEventStream.mockImplementation(async (_body, onFrame) => {
      onFrame({
        event: 'message',
        data: JSON.stringify({
          type: 'bundle_diff',
          files: [
            {
              file: 'SKILL.md',
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
            },
            {
              file: 'scripts/redeploy.sh',
              hunks: [
                {
                  old_start: 1,
                  old_lines: 0,
                  new_start: 1,
                  new_lines: 1,
                  lines: [{ op: 'add', text: '#!/usr/bin/env bash' }],
                },
              ],
            },
          ],
        }),
      })
    })

    const events: unknown[] = []
    await streamChatTurn(
      {
        message: 'Refactor this skill bundle.',
        source: 'skill_editor',
        context: {
          projectId: 'project-1',
          skillId: 'skill-1',
          skillFilePath: 'SKILL.md',
          skillFileDraft: '---\nname: "deploy"\n---\n',
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
          type: 'bundle_diff',
          files: [
            {
              file: 'SKILL.md',
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
            {
              file: 'scripts/redeploy.sh',
              hunks: [
                {
                  oldStart: 1,
                  oldLines: 0,
                  newStart: 1,
                  newLines: 1,
                  lines: [{ op: 'add', text: '#!/usr/bin/env bash' }],
                },
              ],
            },
          ],
        },
      },
    ])
  })
})
