import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  setTicketRunList,
} from './run-transcript'
import type { TicketRun, TicketRunDetail } from './types'

const liveRun: TicketRun = {
  id: 'run-2',
  attemptNumber: 2,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'Codex',
  adapterType: 'codex-app-server',
  modelName: 'gpt-5.4',
  usage: {
    total: 1540,
    input: 1200,
    output: 340,
    cachedInput: 120,
    cacheCreation: 45,
    reasoning: 80,
    prompt: 920,
    candidate: 260,
    tool: 30,
  },
  status: 'executing',
  currentStepStatus: 'running_tests',
  currentStepSummary: 'Running backend checks.',
  createdAt: '2026-04-01T10:05:00Z',
  runtimeStartedAt: '2026-04-01T10:05:30Z',
  lastHeartbeatAt: '2026-04-01T10:07:00Z',
}

describe('ticket run interrupt blocks', () => {
  it('builds first-class interrupt blocks from persisted trace events', () => {
    const detail: TicketRunDetail = {
      run: liveRun,
      stepEntries: [],
      traceEntries: [
        {
          id: 'trace-interrupt',
          agentRunId: liveRun.id,
          sequence: 1,
          provider: 'codex',
          kind: 'approval_requested',
          stream: 'interrupt',
          output: 'Waiting for command approval to run "make check"',
          payload: {
            request_id: 'approval-1',
            kind: 'command_execution',
            command: 'make check',
            options: [{ id: 'approve_once', label: 'Approve once' }],
          },
          createdAt: '2026-04-01T10:05:36Z',
        },
      ],
    }

    const hydrated = hydrateTicketRunDetail(
      setTicketRunList(createEmptyTicketRunTranscriptState(), [liveRun]),
      detail,
    )

    expect(hydrated.blocks).toContainEqual({
      kind: 'interrupt',
      id: 'interrupt:approval-1',
      interruptKind: 'command_execution_approval',
      title: 'Command approval required',
      summary: 'Waiting for command approval to run "make check"',
      at: '2026-04-01T10:05:36Z',
      payload: detail.traceEntries[0]!.payload,
      options: [{ id: 'approve_once', label: 'Approve once', rawDecision: undefined }],
    })
  })

  it('preserves same-status progress entries when the summary changes', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [liveRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-1',
          agent_run_id: liveRun.id,
          ticket_id: 'ticket-1',
          step_status: 'running_command',
          summary: 'Running backend checks.',
          created_at: '2026-04-01T10:06:02Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-2',
          agent_run_id: liveRun.id,
          ticket_id: 'ticket-1',
          step_status: 'running_command',
          summary: 'Running frontend checks.',
          created_at: '2026-04-01T10:06:05Z',
        },
      }),
    })

    expect(state.blocks.filter((block) => block.kind === 'step')).toHaveLength(2)
    expect(state.currentRun?.currentStepSummary).toBe('Running frontend checks.')
  })
})
