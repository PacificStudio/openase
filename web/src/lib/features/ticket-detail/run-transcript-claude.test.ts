import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  setTicketRunList,
} from './run-transcript'
import { latestRun } from './run-transcript.test-fixtures'
import type { TicketRunTranscriptBlock } from './types'

describe('ticket run transcript reducer Claude traces', () => {
  it('renders Claude task traces as command output, tool calls, session status, and errors', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-task-progress',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 1,
          provider: 'claude',
          kind: 'task_progress',
          stream: 'task',
          output: '',
          payload: {
            stream: 'command',
            command: 'pwd',
            text: '/repo\n',
            snapshot: true,
            item_id: 'tool-use-1',
          },
          created_at: '2026-04-01T10:06:10Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-task-notice',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 2,
          provider: 'claude',
          kind: 'task_notification',
          stream: 'task',
          output: '',
          payload: {
            tool: 'functions.exec_command',
            arguments: { cmd: 'pwd' },
          },
          created_at: '2026-04-01T10:06:11Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-session-state',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 3,
          provider: 'claude',
          kind: 'session_state',
          stream: 'task',
          output: '',
          payload: {
            status: 'active',
            detail: 'Running',
            active_flags: ['running'],
          },
          created_at: '2026-04-01T10:06:12Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-error',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 4,
          provider: 'claude',
          kind: 'error',
          stream: 'task',
          output: 'Claude Code reported an empty error result.',
          payload: {
            type: 'result',
            subtype: 'error',
          },
          created_at: '2026-04-01T10:06:13Z',
        },
      }),
    })

    expect(
      state.blocks.find(
        (block): block is Extract<TicketRunTranscriptBlock, { kind: 'terminal_output' }> =>
          block.kind === 'terminal_output',
      ),
    ).toMatchObject({
      id: 'terminal_output:tool-use-1',
      command: 'pwd',
      stream: 'command',
      text: '/repo\n',
    })
    expect(state.blocks).toContainEqual({
      kind: 'tool_call',
      id: 'tool:trace-task-notice',
      toolName: 'functions.exec_command',
      arguments: { cmd: 'pwd' },
      summary: undefined,
      at: '2026-04-01T10:06:11Z',
    })
    expect(state.blocks).toContainEqual({
      kind: 'task_status',
      id: 'status:trace-session-state',
      statusType: 'session_state',
      title: 'Claude session status',
      detail: 'active · Running · running',
      raw: { status: 'active', detail: 'Running', active_flags: ['running'] },
      at: '2026-04-01T10:06:12Z',
    })
    expect(state.blocks).toContainEqual({
      kind: 'task_status',
      id: 'status:trace-error',
      statusType: 'error',
      title: 'Turn failed',
      detail: 'Claude Code reported an empty error result.',
      raw: { type: 'result', subtype: 'error' },
      at: '2026-04-01T10:06:13Z',
    })
  })
})
