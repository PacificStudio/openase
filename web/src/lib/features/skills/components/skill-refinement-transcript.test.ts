import { describe, expect, it } from 'vitest'

import {
  appendSkillRefinementTranscriptEvent,
  createSkillRefinementTranscriptState,
  updateSkillRefinementAnchorState,
} from './skill-refinement-transcript'

describe('skill refinement transcript reducer', () => {
  it('maps diff_updated events into standalone diff transcript entries', () => {
    const nextState = appendSkillRefinementTranscriptEvent(createSkillRefinementTranscriptState(), {
      kind: 'diff_updated',
      payload: {
        threadId: 'thread-1',
        turnId: 'turn-1',
        entryId: 'entry-diff-1',
        diff: [
          'diff --git a/SKILL.md b/SKILL.md',
          '--- a/SKILL.md',
          '+++ b/SKILL.md',
          '@@ -1,1 +1,2 @@',
          ' Use safe steps.',
          '+Verify rollback steps before production deploys.',
        ].join('\n'),
      },
    })

    expect(nextState.entries).toMatchObject([
      {
        kind: 'diff',
        role: 'assistant',
        diff: {
          file: 'SKILL.md',
          hunks: [
            {
              oldStart: 1,
              oldLines: 1,
              newStart: 1,
              newLines: 2,
              lines: [
                { op: 'context', text: 'Use safe steps.' },
                { op: 'add', text: 'Verify rollback steps before production deploys.' },
              ],
            },
          ],
        },
      },
    ])
  })

  it('maps thread_compacted and session_anchor events into task status entries and anchor state', () => {
    let state = createSkillRefinementTranscriptState()

    state = appendSkillRefinementTranscriptEvent(state, {
      kind: 'thread_compacted',
      payload: {
        threadId: 'thread-1',
        turnId: 'turn-1',
        entryId: 'entry-compact-1',
      },
    })
    state = appendSkillRefinementTranscriptEvent(state, {
      kind: 'session_anchor',
      payload: {
        providerThreadId: 'thread-1',
        providerTurnId: 'turn-1',
        providerAnchorId: 'thread-1',
        providerAnchorKind: 'thread',
        providerTurnSupported: true,
      },
    })

    expect(state.entries).toMatchObject([
      {
        kind: 'task_status',
        statusType: 'task_notification',
        title: 'Thread compacted',
        detail: 'Thread thread-1 compacted',
        raw: {
          thread_id: 'thread-1',
          turn_id: 'turn-1',
          entry_id: 'entry-compact-1',
        },
      },
      {
        kind: 'task_status',
        statusType: 'task_notification',
        title: 'Provider thread anchored',
        detail: 'anchor: thread-1\nturn: turn-1\nturn support: yes',
        raw: {
          provider_anchor_id: 'thread-1',
          provider_anchor_kind: 'thread',
          provider_thread_id: 'thread-1',
          provider_turn_id: 'turn-1',
          provider_turn_supported: true,
        },
      },
    ])

    const anchorState = updateSkillRefinementAnchorState(
      {},
      {
        kind: 'session_anchor',
        payload: {
          providerThreadId: 'thread-1',
          providerTurnId: 'turn-1',
          providerAnchorId: 'thread-1',
          providerAnchorKind: 'thread',
          providerTurnSupported: true,
        },
      },
    )

    expect(anchorState).toEqual({
      anchorKind: 'thread',
      anchorId: 'thread-1',
      turnId: 'turn-1',
    })
  })
})
