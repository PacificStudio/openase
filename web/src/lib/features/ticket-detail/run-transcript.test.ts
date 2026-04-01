import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  selectTicketRun,
  setTicketRunList,
} from './run-transcript'
import type { TicketRun, TicketRunDetail, TicketRunTranscriptBlock } from './types'

const latestRun: TicketRun = {
  id: 'run-2',
  attemptNumber: 2,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'Codex',
  status: 'executing',
  currentStepStatus: 'running_tests',
  currentStepSummary: 'Running backend checks.',
  createdAt: '2026-04-01T10:05:00Z',
  runtimeStartedAt: '2026-04-01T10:05:30Z',
  lastHeartbeatAt: '2026-04-01T10:07:00Z',
}

const olderRun: TicketRun = {
  id: 'run-1',
  attemptNumber: 1,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'Codex',
  status: 'completed',
  createdAt: '2026-04-01T09:00:00Z',
}

describe('ticket run transcript reducer', () => {
  it('auto-selects the latest run from true attempt ordering', () => {
    const state = setTicketRunList(createEmptyTicketRunTranscriptState(), [olderRun, latestRun])

    expect(state.selectedRunId).toBe(latestRun.id)
    expect(state.followLatest).toBe(true)
    expect(state.currentRun?.id).toBe(latestRun.id)
  })

  it('hydrates semantic blocks from step and trace entries', () => {
    const detail: TicketRunDetail = {
      run: latestRun,
      stepEntries: [
        {
          id: 'step-1',
          agentRunId: latestRun.id,
          stepStatus: 'planning',
          summary: 'Inspecting ticket detail wiring.',
          createdAt: '2026-04-01T10:05:35Z',
        },
      ],
      traceEntries: [
        {
          id: 'trace-1',
          agentRunId: latestRun.id,
          sequence: 1,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: 'Inspecting the ticket detail panel.',
          payload: { item_id: 'assistant-1' },
          createdAt: '2026-04-01T10:05:36Z',
        },
        {
          id: 'trace-2',
          agentRunId: latestRun.id,
          sequence: 2,
          provider: 'codex',
          kind: 'tool_call_started',
          stream: 'tool',
          output: 'exec_command',
          payload: { tool: 'exec_command' },
          createdAt: '2026-04-01T10:05:37Z',
        },
        {
          id: 'trace-3',
          agentRunId: latestRun.id,
          sequence: 3,
          provider: 'codex',
          kind: 'command_output_delta',
          stream: 'command',
          output: 'ok   ./internal/httpapi\n',
          payload: { item_id: 'command-1' },
          createdAt: '2026-04-01T10:05:38Z',
        },
      ],
    }

    const initial = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])
    const hydrated = hydrateTicketRunDetail(initial, detail)

    expect(hydrated.currentRun?.id).toBe(latestRun.id)
    expect(hydrated.blocks.map((block) => block.kind)).toContain('phase')
    expect(hydrated.blocks).toContainEqual({
      kind: 'step',
      id: 'step:step-1',
      stepStatus: 'planning',
      summary: 'Inspecting ticket detail wiring.',
      at: '2026-04-01T10:05:35Z',
    })
    expect(
      hydrated.blocks.find(
        (block): block is Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' }> =>
          block.kind === 'assistant_message',
      ),
    ).toMatchObject({
      id: 'assistant_message:assistant-1',
      text: 'Inspecting the ticket detail panel.',
      streaming: true,
    })
    expect(hydrated.blocks).toContainEqual({
      kind: 'tool_call',
      id: 'tool:trace-2',
      toolName: 'exec_command',
      summary: undefined,
      at: '2026-04-01T10:05:37Z',
    })
    expect(
      hydrated.blocks.find(
        (block): block is Extract<TicketRunTranscriptBlock, { kind: 'terminal_output' }> =>
          block.kind === 'terminal_output',
      ),
    ).toMatchObject({
      id: 'terminal_output:command-1',
      text: 'ok   ./internal/httpapi\n',
      streaming: true,
    })
  })

  it('builds first-class interrupt blocks from persisted trace events', () => {
    const detail: TicketRunDetail = {
      run: latestRun,
      stepEntries: [],
      traceEntries: [
        {
          id: 'trace-interrupt',
          agentRunId: latestRun.id,
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
      setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun]),
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

  it('merges live output incrementally and switches to a newer run on lifecycle events', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-a',
          agentRunId: latestRun.id,
          sequence: 1,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: 'First chunk. ',
          payload: { item_id: 'assistant-1' },
          createdAt: '2026-04-01T10:06:00Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-b',
          agentRunId: latestRun.id,
          sequence: 2,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: 'Second chunk.',
          payload: { item_id: 'assistant-1' },
          createdAt: '2026-04-01T10:06:01Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-2',
          agentRunId: latestRun.id,
          stepStatus: 'running_tests',
          summary: 'Running backend checks.',
          createdAt: '2026-04-01T10:06:02Z',
        },
      }),
    })

    expect(
      state.blocks.find(
        (block): block is Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' }> =>
          block.kind === 'assistant_message',
      )?.text,
    ).toBe('First chunk. Second chunk.')
    expect(state.currentRun?.currentStepStatus).toBe('running_tests')

    const newerRun: TicketRun = {
      ...latestRun,
      id: 'run-3',
      attemptNumber: 3,
      status: 'ready',
      createdAt: '2026-04-01T11:00:00Z',
      runtimeStartedAt: '2026-04-01T11:00:10Z',
      currentStepStatus: undefined,
      currentStepSummary: undefined,
    }

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.lifecycle',
      data: JSON.stringify({
        run: newerRun,
        lifecycle: {
          eventType: 'agent.ready',
          message: 'Runtime ready',
          createdAt: '2026-04-01T11:00:10Z',
        },
      }),
    })

    expect(state.currentRun?.id).toBe('run-3')
    expect(state.blocks.some((block) => block.kind === 'assistant_message')).toBe(false)
    expect(state.blocks).toContainEqual({
      kind: 'phase',
      id: 'phase:agent.ready:2026-04-01T11:00:10Z',
      phase: 'ready',
      at: '2026-04-01T11:00:10Z',
      summary: 'Runtime ready',
    })
  })

  it('keeps a manually selected historical run focused while newer lifecycle events arrive', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = selectTicketRun(state, olderRun.id)

    const newerRun: TicketRun = {
      ...latestRun,
      id: 'run-3',
      attemptNumber: 3,
      status: 'launching',
      createdAt: '2026-04-01T11:00:00Z',
      runtimeStartedAt: undefined,
    }

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.lifecycle',
      data: JSON.stringify({
        run: newerRun,
        lifecycle: {
          eventType: 'agent.launching',
          message: 'Launching retry runtime',
          createdAt: '2026-04-01T11:00:00Z',
        },
      }),
    })

    expect(state.selectedRunId).toBe(olderRun.id)
    expect(state.followLatest).toBe(false)
    expect(state.currentRun?.id).toBe(olderRun.id)
    expect(state.runs.map((run) => run.id)).toEqual(['run-3', latestRun.id, olderRun.id])
  })

  it('preserves same-status progress entries when the summary changes', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-1',
          agentRunId: latestRun.id,
          stepStatus: 'running_command',
          summary: 'Running backend checks.',
          createdAt: '2026-04-01T10:06:02Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-2',
          agentRunId: latestRun.id,
          stepStatus: 'running_command',
          summary: 'Running frontend checks.',
          createdAt: '2026-04-01T10:06:05Z',
        },
      }),
    })

    expect(state.blocks.filter((block) => block.kind === 'step')).toHaveLength(2)
    expect(state.currentRun?.currentStepSummary).toBe('Running frontend checks.')
  })
})
