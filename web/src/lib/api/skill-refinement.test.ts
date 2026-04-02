import { describe, expect, it } from 'vitest'

import { parseSkillRefinementStreamEvent } from './skill-refinement'

describe('parseSkillRefinementStreamEvent', () => {
  it('parses rich runtime events from the skill refinement SSE stream', () => {
    expect(
      parseSkillRefinementStreamEvent({
        event: 'message',
        data: JSON.stringify({
          type: 'task_progress',
          raw: { text: 'bash -n scripts/check.sh', stream: 'command' },
        }),
      }),
    ).toEqual({
      kind: 'message',
      payload: {
        type: 'task_progress',
        raw: { text: 'bash -n scripts/check.sh', stream: 'command' },
        content: undefined,
      },
    })

    expect(
      parseSkillRefinementStreamEvent({
        event: 'interrupt_requested',
        data: JSON.stringify({
          request_id: 'req-1',
          kind: 'command_execution',
          payload: { command: 'git status' },
          options: [{ id: 'approve_once', label: 'Approve once' }],
        }),
      }),
    ).toEqual({
      kind: 'interrupt_requested',
      payload: {
        requestId: 'req-1',
        kind: 'command_execution',
        payload: { command: 'git status' },
        options: [{ id: 'approve_once', label: 'Approve once' }],
      },
    })

    expect(
      parseSkillRefinementStreamEvent({
        event: 'thread_status',
        data: JSON.stringify({
          thread_id: 'thread-1',
          status: 'active',
          active_flags: ['running'],
          entry_id: 'entry-1',
        }),
      }),
    ).toEqual({
      kind: 'thread_status',
      payload: {
        threadId: 'thread-1',
        status: 'active',
        activeFlags: ['running'],
        entryId: 'entry-1',
      },
    })

    expect(
      parseSkillRefinementStreamEvent({
        event: 'session_state',
        data: JSON.stringify({
          status: 'requires_action',
          active_flags: ['waiting_for_input'],
          detail: 'Claude is waiting for approval.',
          entry_id: 'entry-session-1',
        }),
      }),
    ).toEqual({
      kind: 'session_state',
      payload: {
        status: 'requires_action',
        activeFlags: ['waiting_for_input'],
        detail: 'Claude is waiting for approval.',
        raw: undefined,
        entryId: 'entry-session-1',
      },
    })

    expect(
      parseSkillRefinementStreamEvent({
        event: 'session_anchor',
        data: JSON.stringify({
          ProviderThreadID: 'thread-1',
          LastTurnID: 'turn-1',
          ProviderAnchorID: 'thread-1',
          ProviderAnchorKind: 'thread',
          ProviderTurnSupported: true,
        }),
      }),
    ).toEqual({
      kind: 'session_anchor',
      payload: {
        providerThreadId: 'thread-1',
        providerTurnId: 'turn-1',
        providerAnchorId: 'thread-1',
        providerAnchorKind: 'thread',
        providerTurnSupported: true,
      },
    })
  })
})
