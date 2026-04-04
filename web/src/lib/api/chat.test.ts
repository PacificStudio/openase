import { beforeEach, describe, expect, it, vi } from 'vitest'

const { consumeEventStream } = vi.hoisted(() => ({
  consumeEventStream: vi.fn(),
}))

vi.mock('./sse', () => ({
  consumeEventStream,
}))

import {
  createProjectConversation,
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  parseRawProjectConversationMuxFrame,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  streamChatTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
} from './chat'

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

describe('startProjectConversationTurn', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ turn: { id: 'turn-1', turn_index: 1, status: 'started' } }),
      }),
    )
  })

  it('serializes per-turn workflow focus metadata', async () => {
    await startProjectConversationTurn('conversation-1', {
      message: '帮我看看这里要怎么改',
      focus: {
        kind: 'workflow',
        projectId: 'project-1',
        workflowId: 'workflow-1',
        workflowName: 'Backend Engineer',
        workflowType: 'coding',
        harnessPath: '.openase/harnesses/backend.md',
        isActive: true,
        selectedArea: 'harness',
        hasDirtyDraft: true,
      },
    })

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/chat/conversations/conversation-1/turns'),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          message: '帮我看看这里要怎么改',
          focus: {
            kind: 'workflow',
            workflow_id: 'workflow-1',
            workflow_name: 'Backend Engineer',
            workflow_type: 'coding',
            harness_path: '.openase/harnesses/backend.md',
            is_active: true,
            selected_area: 'harness',
            has_dirty_draft: true,
          },
        }),
      }),
    )
  })
})

describe('watchProjectConversation', () => {
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

  it('keeps task message raw payloads on the inner raw object for project conversation streams', async () => {
    consumeEventStream.mockImplementation(async (_body, onFrame) => {
      onFrame({
        event: 'message',
        data: JSON.stringify({
          type: 'task_notification',
          raw: {
            tool: 'functions.exec_command',
            arguments: { cmd: 'git status' },
          },
        }),
      })
    })

    const events: unknown[] = []
    await watchProjectConversation('conversation-1', {
      onEvent: (event) => {
        events.push(event)
      },
    })

    expect(events).toEqual([
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
    ])
  })

  it('downgrades platform command proposals to plain assistant text in project conversation streams', async () => {
    consumeEventStream.mockImplementation(async (_body, onFrame) => {
      onFrame({
        event: 'message',
        data: JSON.stringify({
          type: 'platform_command_proposal',
          entry_id: 'entry-1',
          summary: 'Update ASE-1',
          commands: [
            {
              command: 'ticket.update',
              args: {
                ticket: 'ASE-1',
                status: 'Todo',
              },
            },
          ],
        }),
      })
    })

    const events: unknown[] = []
    await watchProjectConversation('conversation-1', {
      onEvent: (event) => {
        events.push(event)
      },
    })

    expect(events).toEqual([
      {
        kind: 'message',
        payload: {
          type: 'text',
          content: 'Update ASE-1',
        },
      },
    ])
  })

  it('parses provider anchor metadata from session events', async () => {
    consumeEventStream.mockImplementation(async (_body, onFrame) => {
      onFrame({
        event: 'session',
        data: JSON.stringify({
          conversation_id: 'conversation-1',
          runtime_state: 'ready',
          provider_anchor_kind: 'session',
          provider_anchor_id: 'claude-session-1',
          provider_turn_supported: false,
          provider_status: 'requires_action',
          provider_active_flags: ['requires_action'],
        }),
      })
    })

    const events: unknown[] = []
    await watchProjectConversation('conversation-1', {
      onEvent: (event) => {
        events.push(event)
      },
    })

    expect(events).toEqual([
      {
        kind: 'session',
        payload: {
          conversationId: 'conversation-1',
          runtimeState: 'ready',
          providerAnchorKind: 'session',
          providerAnchorId: 'claude-session-1',
          providerTurnId: undefined,
          providerTurnSupported: false,
          providerStatus: 'requires_action',
          providerActiveFlags: ['requires_action'],
        },
      },
    ])
  })
})

describe('project conversation REST mapping', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps snake_case project conversation payloads to camelCase fields', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            conversations: [
              {
                id: 'conversation-1',
                project_id: 'project-1',
                user_id: 'user-1',
                source: 'project_sidebar',
                provider_id: 'provider-1',
                status: 'active',
                rolling_summary: 'Latest thread',
                last_activity_at: '2026-04-02T05:00:00Z',
                created_at: '2026-04-02T04:00:00Z',
                updated_at: '2026-04-02T05:00:00Z',
              },
            ],
          }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 201,
          json: async () => ({
            conversation: {
              id: 'conversation-2',
              project_id: 'project-1',
              user_id: 'user-1',
              source: 'project_sidebar',
              provider_id: 'provider-2',
              status: 'active',
              rolling_summary: '',
              last_activity_at: '2026-04-02T06:00:00Z',
              created_at: '2026-04-02T06:00:00Z',
              updated_at: '2026-04-02T06:00:00Z',
            },
          }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({
            conversation: {
              id: 'conversation-3',
              project_id: 'project-1',
              user_id: 'user-1',
              source: 'project_sidebar',
              provider_id: 'provider-3',
              status: 'active',
              rolling_summary: 'Recovered thread',
              created_at: '2026-04-02T07:00:00Z',
              updated_at: '2026-04-02T07:05:00Z',
            },
          }),
        }),
    )

    await expect(
      listProjectConversations({ projectId: 'project-1', providerId: 'provider-1' }),
    ).resolves.toEqual({
      conversations: [
        {
          id: 'conversation-1',
          projectId: 'project-1',
          userId: 'user-1',
          source: 'project_sidebar',
          providerId: 'provider-1',
          providerActiveFlags: [],
          status: 'active',
          rollingSummary: 'Latest thread',
          lastActivityAt: '2026-04-02T05:00:00Z',
          createdAt: '2026-04-02T04:00:00Z',
          updatedAt: '2026-04-02T05:00:00Z',
        },
      ],
    })

    await expect(
      createProjectConversation({ projectId: 'project-1', providerId: 'provider-2' }),
    ).resolves.toEqual({
      conversation: {
        id: 'conversation-2',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-2',
        providerActiveFlags: [],
        status: 'active',
        rollingSummary: '',
        lastActivityAt: '2026-04-02T06:00:00Z',
        createdAt: '2026-04-02T06:00:00Z',
        updatedAt: '2026-04-02T06:00:00Z',
      },
    })

    await expect(getProjectConversation('conversation-3')).resolves.toEqual({
      conversation: {
        id: 'conversation-3',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-3',
        providerActiveFlags: [],
        status: 'active',
        rollingSummary: 'Recovered thread',
        lastActivityAt: '2026-04-02T07:05:00Z',
        createdAt: '2026-04-02T07:00:00Z',
        updatedAt: '2026-04-02T07:05:00Z',
      },
    })
  })

  it('maps snake_case project conversation entries to camelCase fields', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          entries: [
            {
              id: 'entry-1',
              conversation_id: 'conversation-1',
              turn_id: 'turn-1',
              seq: 1,
              kind: 'user_message',
              payload: { content: 'hello' },
              created_at: '2026-04-02T08:00:00Z',
            },
          ],
        }),
      }),
    )

    await expect(listProjectConversationEntries('conversation-1')).resolves.toEqual({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'hello' },
          createdAt: '2026-04-02T08:00:00Z',
        },
      ],
    })
  })

  it('sends chat user headers when responding to project conversation interrupts', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          interrupt: {
            id: 'interrupt-1',
          },
        }),
      }),
    )

    await respondProjectConversationInterrupt('conversation-1', 'interrupt-1', {
      decision: 'approve_once',
    })

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining(
        '/api/v1/chat/conversations/conversation-1/interrupts/interrupt-1/respond',
      ),
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'X-OpenASE-Chat-User': expect.any(String),
        }),
        body: JSON.stringify({
          decision: 'approve_once',
          answer: undefined,
        }),
      }),
    )
  })

  it('sends chat user headers when opening the project conversation mux stream', async () => {
    consumeEventStream.mockImplementation(async () => {})
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        body: {} as ReadableStream<Uint8Array>,
      }),
    )

    await watchProjectConversationMuxStream('project-1', {
      onFrame: vi.fn(),
    })

    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/chat/projects/project-1/conversations/stream',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          accept: 'text/event-stream',
          'X-OpenASE-Chat-User': expect.any(String),
        }),
      }),
    )
  })
})

describe('parseRawProjectConversationMuxFrame', () => {
  it('parses multiplexed project conversation frames into typed events', () => {
    expect(
      parseRawProjectConversationMuxFrame({
        event: 'turn_done',
        data: JSON.stringify({
          conversation_id: 'conversation-1',
          sent_at: '2026-04-04T12:34:56Z',
          payload: {
            conversation_id: 'conversation-1',
            turn_id: 'turn-1',
            cost_usd: 1.25,
          },
        }),
      }),
    ).toEqual({
      ok: true,
      value: {
        conversationId: 'conversation-1',
        sentAt: '2026-04-04T12:34:56Z',
        event: {
          kind: 'turn_done',
          payload: {
            conversationId: 'conversation-1',
            turnId: 'turn-1',
            costUSD: 1.25,
          },
        },
      },
    })
  })

  it('rejects malformed multiplexed frames before they reach controller code', () => {
    const result = parseRawProjectConversationMuxFrame({
      event: 'message',
      data: JSON.stringify({
        sent_at: '2026-04-04T12:34:56Z',
        payload: {
          type: 'text',
          content: 'hello',
        },
      }),
    })

    expect(result.ok).toBe(false)
    expect(result.ok ? '' : result.error.message).toContain('conversation_id')
  })
})
