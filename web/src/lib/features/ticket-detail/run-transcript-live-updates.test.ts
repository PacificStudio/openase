import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  selectTicketRun,
  setTicketRunList,
} from './run-transcript'
import { buildRunTimeline } from './run-transcript-blocks'
import { buildNewerRun, latestRun, olderRun } from './run-transcript.test-fixtures'
import { buildHydratedRunDetail } from './run-transcript.test-fixtures'
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

describe('ticket run transcript live updates', () => {
  it('reaches the same transcript state when replaying streamed step/trace events as when hydrating the same run detail', () => {
    const detail = buildHydratedRunDetail()
    const initial = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    const hydrated = hydrateTicketRunDetail(initial, detail)

    let replayed = initial
    for (const item of buildRunTimeline(detail.stepEntries, detail.traceEntries)) {
      replayed = applyTicketRunStreamFrame(replayed, {
        event: item.kind === 'step' ? 'ticket.run.step' : 'ticket.run.trace',
        data: JSON.stringify({
          entry:
            item.kind === 'step'
              ? {
                  id: item.entry.id,
                  agent_run_id: item.entry.agentRunId,
                  ticket_id: 'ticket-1',
                  step_status: item.entry.stepStatus,
                  summary: item.entry.summary,
                  source_trace_event_id: item.entry.sourceTraceEventId ?? null,
                  created_at: item.entry.createdAt,
                }
              : {
                  id: item.entry.id,
                  agent_run_id: item.entry.agentRunId,
                  ticket_id: 'ticket-1',
                  sequence: item.entry.sequence,
                  provider: item.entry.provider,
                  kind: item.entry.kind,
                  stream: item.entry.stream,
                  output: item.entry.output,
                  payload: item.entry.payload,
                  created_at: item.entry.createdAt,
                },
        }),
      })
    }

    expect(replayed).toEqual(hydrated)
  })

  it('keeps a manually selected historical run focused while newer lifecycle events arrive', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = selectTicketRun(state, olderRun.id)

    const newerRun = buildNewerRun({
      status: 'launching',
      runtimeStartedAt: undefined,
    })

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.lifecycle',
      data: JSON.stringify({
        run: toRunRecord(newerRun),
        lifecycle: {
          event_type: 'agent.launching',
          message: 'Launching retry runtime',
          created_at: '2026-04-01T11:00:00Z',
        },
      }),
    })

    expect(state.selectedRunId).toBe(olderRun.id)
    expect(state.followLatest).toBe(false)
    expect(state.currentRun?.id).toBe(olderRun.id)
    expect(state.runs.map((run) => run.id)).toEqual(['run-3', latestRun.id, olderRun.id])
  })

  it('updates the current run completion summary from live summary events', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.summary',
      data: JSON.stringify({
        project_id: 'project-1',
        ticket_id: 'ticket-1',
        run_id: latestRun.id,
        run: toRunRecord({
          ...latestRun,
          status: 'ended',
          terminalAt: '2026-04-01T10:09:59Z',
        }),
        completion_summary: {
          status: 'completed',
          markdown: '## Overview\n\nSummary ready.',
          generated_at: '2026-04-01T10:10:00Z',
          json: { provider: 'Codex' },
        },
      }),
    })

    expect(state.currentRun?.completionSummary).toEqual({
      status: 'completed',
      markdown: '## Overview\n\nSummary ready.',
      generatedAt: '2026-04-01T10:10:00Z',
      json: { provider: 'Codex' },
      error: undefined,
    })
    expect(state.currentRun?.status).toBe('ended')
    expect(state.currentRun?.terminalAt).toBe('2026-04-01T10:09:59Z')
    expect(state.runs[0]?.completionSummary?.status).toBe('completed')
  })

  it('preserves stream-supplemented transcript blocks when hydrating the same run detail again', () => {
    const detail = buildHydratedRunDetail()
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])
    state = hydrateTicketRunDetail(state, detail)

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-stream-tail',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 99,
          provider: 'codex',
          kind: 'assistant_snapshot',
          stream: 'assistant',
          output: 'Stream-only tail message.',
          payload: { item_id: 'assistant-tail' },
          created_at: '2026-04-01T10:06:55Z',
        },
      }),
    })

    const beforeRefreshBlockIDs = state.blocks.map((block) => block.id)
    state = hydrateTicketRunDetail(state, detail)

    expect(state.blocks.map((block) => block.id)).toEqual(beforeRefreshBlockIDs)
    expect(
      state.blocks.some(
        (block) =>
          block.kind === 'assistant_message' && block.id === 'assistant_message:assistant-tail',
      ),
    ).toBe(true)
  })

  it('does not let an older non-terminal lifecycle snapshot regress a terminal run after detail hydration', () => {
    const terminalRun: TicketRun = {
      ...latestRun,
      status: 'ended',
      terminalAt: '2026-04-01T10:09:59Z',
    }

    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [terminalRun, olderRun])
    state = hydrateTicketRunDetail(state, {
      run: terminalRun,
      stepEntries: [],
      traceEntries: [],
    })

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.lifecycle',
      data: JSON.stringify({
        run: toRunRecord({
          ...terminalRun,
          status: 'ready',
          terminalAt: undefined,
          currentStepStatus: undefined,
          currentStepSummary: undefined,
        }),
        lifecycle: {
          event_type: 'agent.ready',
          message: 'Runtime ready',
          created_at: '2026-04-01T10:05:30Z',
        },
      }),
    })

    expect(state.currentRun?.status).toBe('ended')
    expect(state.currentRun?.terminalAt).toBe('2026-04-01T10:09:59Z')
  })

  it('keeps trace-backed step entries out of transcript blocks while updating current step state', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun, olderRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.step',
      data: JSON.stringify({
        entry: {
          id: 'step-trace-backed',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          step_status: 'running_command',
          summary: 'pnpm vitest run',
          source_trace_event_id: 'trace-command-1',
          created_at: '2026-04-01T10:06:02Z',
        },
      }),
    })

    expect(state.currentRun?.currentStepStatus).toBe('running_command')
    expect(state.currentRun?.currentStepSummary).toBe('pnpm vitest run')
    expect(state.blocks.some((block) => block.kind === 'step')).toBe(false)
  })

  it('keeps assistant snapshots without item ids as separate transcript blocks', () => {
    let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [latestRun])

    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-no-item-1',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 1,
          provider: 'claude',
          kind: 'assistant_snapshot',
          stream: 'assistant',
          output: 'First Claude snapshot.',
          payload: {},
          created_at: '2026-04-01T10:06:20Z',
        },
      }),
    })
    state = applyTicketRunStreamFrame(state, {
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-no-item-2',
          agent_run_id: latestRun.id,
          ticket_id: 'ticket-1',
          sequence: 2,
          provider: 'claude',
          kind: 'assistant_snapshot',
          stream: 'assistant',
          output: 'Second Claude snapshot.',
          payload: {},
          created_at: '2026-04-01T10:06:21Z',
        },
      }),
    })

    const assistantBlocks = state.blocks.filter(
      (block): block is Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' }> =>
        block.kind === 'assistant_message',
    )
    expect(assistantBlocks).toHaveLength(2)
    expect(assistantBlocks.map((block) => block.id)).toEqual([
      'assistant_message:trace-no-item-1',
      'assistant_message:trace-no-item-2',
    ])
    expect(assistantBlocks.map((block) => block.text)).toEqual([
      'First Claude snapshot.',
      'Second Claude snapshot.',
    ])
  })
})
