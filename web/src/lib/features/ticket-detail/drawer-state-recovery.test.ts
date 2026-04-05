import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import { createTicketDrawerState } from './drawer-state.svelte'

function buildContext(overrides: Partial<TicketDetailLiveContext> = {}): TicketDetailLiveContext {
  return {
    ticket: {
      id: 'ticket-1',
      identifier: 'ASE-336',
      title: 'Align Ticket Detail refresh wiring',
      description: 'Initial description',
      status: { id: 'status-1', name: 'Todo', color: '#2563eb' },
      priority: 'high',
      type: 'feature',
      archived: false,
      repoScopes: [],
      attemptCount: 0,
      consecutiveErrors: 0,
      retryPaused: false,
      costTokensInput: 0,
      costTokensOutput: 0,
      costTokensTotal: 0,
      costAmount: 0,
      budgetUsd: 0,
      dependencies: [],
      externalLinks: [],
      children: [],
      createdBy: 'codex',
      createdAt: '2026-03-29T10:00:00Z',
      updatedAt: '2026-03-29T10:00:00Z',
    },
    timeline: [],
    hooks: [],
    ...overrides,
  }
}

describe('createTicketDrawerState recovery', () => {
  it('backfills the selected run transcript after a stream reconnect without duplicating blocks', async () => {
    const run = {
      id: 'run-1',
      attempt_number: 1,
      agent_id: 'agent-1',
      agent_name: 'Ticket Runner',
      provider: 'Codex',
      adapter_type: 'codex-app-server',
      model_name: 'gpt-5.4',
      usage: {
        total: 1540,
        input: 1200,
        output: 340,
        cached_input: 120,
        cache_creation: 45,
        reasoning: 80,
        prompt: 920,
        candidate: 260,
        tool: 30,
      },
      status: 'executing',
      current_step_status: 'running_command',
      current_step_summary: 'Running checks.',
      created_at: '2026-04-01T10:00:00Z',
      runtime_started_at: '2026-04-01T10:00:05Z',
      last_heartbeat_at: '2026-04-01T10:00:10Z',
    }
    const runDeps = {
      fetchRuns: vi.fn().mockResolvedValue({ runs: [run] }),
      fetchRun: vi
        .fn()
        .mockResolvedValueOnce({
          run,
          trace_entries: [
            {
              id: 'trace-1',
              agent_run_id: 'run-1',
              sequence: 1,
              provider: 'codex',
              kind: 'assistant_delta',
              stream: 'assistant',
              output: 'First chunk.',
              payload: { item_id: 'assistant-1' },
              created_at: '2026-04-01T10:00:06Z',
            },
          ],
          step_entries: [],
        })
        .mockResolvedValueOnce({
          run: {
            ...run,
            current_step_summary: 'Recovered checks.',
          },
          trace_entries: [
            {
              id: 'trace-1',
              agent_run_id: 'run-1',
              sequence: 1,
              provider: 'codex',
              kind: 'assistant_delta',
              stream: 'assistant',
              output: 'First chunk.',
              payload: { item_id: 'assistant-1' },
              created_at: '2026-04-01T10:00:06Z',
            },
            {
              id: 'trace-2',
              agent_run_id: 'run-1',
              sequence: 2,
              provider: 'codex',
              kind: 'assistant_delta',
              stream: 'assistant',
              output: ' Missing chunk.',
              payload: { item_id: 'assistant-1' },
              created_at: '2026-04-01T10:00:07Z',
            },
          ],
          step_entries: [
            {
              id: 'step-1',
              agent_run_id: 'run-1',
              step_status: 'running_command',
              summary: 'Recovered checks.',
              source_trace_event_id: null,
              created_at: '2026-04-01T10:00:08Z',
            },
          ],
        }),
    }

    const state = createTicketDrawerState({
      fetchLiveContext: vi.fn().mockResolvedValue(buildContext()),
      fetchReferenceData: vi
        .fn<() => Promise<TicketDetailProjectReferenceData>>()
        .mockResolvedValue({
          statusLookup: [{ id: 'status-1', stage: 'unstarted', color: '#2563eb' }],
          statuses: [{ id: 'status-1', name: 'Todo', color: '#2563eb' }],
          dependencyCandidatesByTicketId: [],
          repoOptions: [],
        }),
      ...runDeps,
    })

    await state.load('project-1', 'ticket-1')
    state.applyRunStreamFrame({
      event: 'ticket.run.trace',
      data: JSON.stringify({
        entry: {
          id: 'trace-live',
          agent_run_id: 'run-1',
          ticket_id: 'ticket-1',
          sequence: 2,
          provider: 'codex',
          kind: 'assistant_delta',
          stream: 'assistant',
          output: ' Missing chunk.',
          payload: { item_id: 'assistant-1' },
          created_at: '2026-04-01T10:00:07Z',
        },
      }),
    })

    await state.recoverRunTranscript('project-1', 'ticket-1')

    expect(runDeps.fetchRuns).toHaveBeenCalledTimes(2)
    expect(runDeps.fetchRun).toHaveBeenCalledTimes(2)
    expect(state.currentRun?.currentStepSummary).toBe('Recovered checks.')
    expect(state.runBlocks.filter((block) => block.kind === 'assistant_message')).toHaveLength(1)
    expect(
      state.runBlocks.find((block) => block.kind === 'assistant_message' && 'text' in block)?.text,
    ).toBe('First chunk. Missing chunk.')
  })
})
