import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  setTicketRunList,
} from './run-transcript'
import {
  buildHydratedRunDetail,
  buildNewerRun,
  latestRun,
  olderRun,
} from './run-transcript.test-fixtures'
import type { TicketRun, TicketRunTranscriptBlock } from './types'

function toRunRecord(run: TicketRun) {
  return {
    id: run.id,
    ticket_id: 'ticket-1',
    attempt_number: run.attemptNumber,
    agent_id: run.agentId,
    agent_name: run.agentName,
    provider: run.provider,
    status: run.status,
    current_step_status: run.currentStepStatus ?? null,
    current_step_summary: run.currentStepSummary ?? null,
    created_at: run.createdAt,
    runtime_started_at: run.runtimeStartedAt ?? null,
    last_heartbeat_at: run.lastHeartbeatAt ?? null,
    completed_at: run.completedAt ?? null,
    terminal_at: run.terminalAt ?? run.completedAt ?? null,
    last_error: run.lastError ?? null,
    completion_summary: run.completionSummary
      ? {
          status: run.completionSummary.status,
          markdown: run.completionSummary.markdown ?? null,
          json: run.completionSummary.json ?? null,
          generated_at: run.completionSummary.generatedAt ?? null,
          error: run.completionSummary.error ?? null,
        }
      : null,
  }
}

describe('ticket run transcript reducer', () => {
  it('auto-selects the latest run from true attempt ordering', () => {
    const state = setTicketRunList(createEmptyTicketRunTranscriptState(), [olderRun, latestRun])

    expect(state.selectedRunId).toBe(latestRun.id)
    expect(state.followLatest).toBe(true)
    expect(state.currentRun?.id).toBe(latestRun.id)
  })

  it('hydrates semantic blocks from step and trace entries', () => {
    const detail = buildHydratedRunDetail()
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
      toolName: 'functions.exec_command',
      arguments: { cmd: 'pnpm vitest run' },
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
      command: 'pnpm vitest run',
      stream: 'command',
      text: 'ok   ./internal/httpapi\n',
      streaming: true,
    })
    expect(hydrated.blocks).toContainEqual({
      kind: 'task_status',
      id: 'status:trace-4',
      statusType: 'thread_status',
      title: 'Codex thread status',
      detail: 'active · waitingOnUserInput',
      raw: { status: 'active', active_flags: ['waitingOnUserInput'] },
      at: '2026-04-01T10:05:39Z',
    })
    expect(hydrated.blocks).toContainEqual({
      kind: 'task_status',
      id: 'reasoning:trace-5',
      statusType: 'reasoning_updated',
      title: 'Reasoning update',
      detail: 'Inspecting the reducer.',
      raw: { kind: 'text_delta', content_index: 0 },
      at: '2026-04-01T10:05:40Z',
    })
    expect(
      hydrated.blocks.find(
        (block): block is Extract<TicketRunTranscriptBlock, { kind: 'diff' }> =>
          block.kind === 'diff',
      ),
    ).toMatchObject({
      id: 'diff:trace-6',
      diff: {
        file: 'app.ts',
        hunks: [
          {
            oldStart: 1,
            oldLines: 1,
            newStart: 1,
            newLines: 1,
            lines: [
              { op: 'remove', text: 'old' },
              { op: 'add', text: 'new' },
            ],
          },
        ],
      },
    })
  })

  it('merges live output incrementally and switches to a newer run on lifecycle events', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-a',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 1,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: 'First chunk. ',
          payload: { item_id: 'assistant-1' },
          created_at: '2026-04-01T10:06:00Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-b',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 2,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: 'Second chunk.',
          payload: { item_id: 'assistant-1' },
          created_at: '2026-04-01T10:06:01Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-2',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          step_status: 'running_tests',
          summary: 'Running backend checks.',
          created_at: '2026-04-01T10:06:02Z',
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

    const newerRun = buildNewerRun()

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.lifecycle',
      data: JSON.stringify({
        run: toRunRecord(newerRun),
        lifecycle: {
          event_type: 'agent.ready',
          message: 'Runtime ready',
          created_at: '2026-04-01T11:00:10Z',
        },
      }),
    })

    expect(state.currentRun?.id).toBe('run-3')
    expect(state.currentRun?.attemptNumber).toBe(3)
    expect(state.currentRun?.createdAt).toBe('2026-04-01T11:00:00Z')
    expect(state.blocks.some((block) => block.kind === 'assistant_message')).toBe(false)
    expect(state.blocks).toContainEqual({
      kind: 'phase',
      id: 'phase:agent.ready:2026-04-01T11:00:10Z',
      phase: 'ready',
      at: '2026-04-01T11:00:10Z',
      summary: 'Runtime ready',
    })
  })
})
