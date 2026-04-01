import { describe, expect, it } from 'vitest'
import { mapPersistedEntries } from './project-conversation-transcript-state'

describe('mapPersistedEntries', () => {
  it('restores system task entries into structured transcript cards', () => {
    const entries = mapPersistedEntries([
      {
        id: 'entry-started',
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        seq: 1,
        kind: 'system',
        payload: {
          type: 'task_started',
          raw: { status: 'running' },
        },
        createdAt: '2026-04-01T12:00:00Z',
      },
      {
        id: 'entry-tool',
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        seq: 2,
        kind: 'system',
        payload: {
          type: 'task_notification',
          raw: {
            tool: 'functions.exec_command',
            arguments: { cmd: 'git status' },
          },
        },
        createdAt: '2026-04-01T12:00:01Z',
      },
      {
        id: 'entry-command-1',
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        seq: 3,
        kind: 'system',
        payload: {
          type: 'task_progress',
          raw: {
            stream: 'command',
            phase: 'stdout',
            snapshot: false,
            text: 'first line\n',
          },
        },
        createdAt: '2026-04-01T12:00:02Z',
      },
      {
        id: 'entry-command-2',
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        seq: 4,
        kind: 'system',
        payload: {
          type: 'task_progress',
          raw: {
            stream: 'command',
            phase: 'stdout',
            snapshot: false,
            text: 'second line\n',
          },
        },
        createdAt: '2026-04-01T12:00:03Z',
      },
      {
        id: 'entry-diff',
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        seq: 5,
        kind: 'diff',
        payload: {
          type: 'diff',
          file: 'README.md',
          hunks: [
            {
              old_start: 1,
              old_lines: 1,
              new_start: 1,
              new_lines: 2,
              lines: [
                { op: 'context', text: 'old line' },
                { op: 'add', text: 'new line' },
              ],
            },
          ],
        },
        createdAt: '2026-04-01T12:00:04Z',
      },
    ])

    expect(entries).toMatchObject([
      {
        kind: 'task_status',
        title: 'Task started',
        detail: 'Status: running',
      },
      {
        kind: 'tool_call',
        tool: 'functions.exec_command',
        arguments: { cmd: 'git status' },
      },
      {
        kind: 'command_output',
        stream: 'command',
        phase: 'stdout',
        content: 'first line\nsecond line\n',
      },
      {
        kind: 'diff',
        diff: {
          file: 'README.md',
          hunks: [
            {
              oldStart: 1,
              oldLines: 1,
              newStart: 1,
              newLines: 2,
            },
          ],
        },
      },
    ])
  })
})
