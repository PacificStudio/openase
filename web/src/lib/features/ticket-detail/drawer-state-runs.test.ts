import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import { buildLiveContext, buildReferenceData } from './drawer-state.test-fixtures'
import { createTicketDrawerState } from './drawer-state.svelte'

describe('createTicketDrawerState run loading', () => {
  it('lazy-loads ticket runs only when explicitly requested', async () => {
    const referenceData = buildReferenceData()
    const liveContext = buildLiveContext()
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(referenceData)
    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValue(liveContext)
    const fetchRuns = vi.fn().mockResolvedValue({
      runs: [
        {
          id: 'run-1',
          ticket_id: 'ticket-1',
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
          current_step_status: 'running_tests',
          current_step_summary: 'Running backend checks.',
          created_at: '2026-04-01T10:05:00Z',
          runtime_started_at: '2026-04-01T10:05:30Z',
          last_heartbeat_at: '2026-04-01T10:07:00Z',
          terminal_at: null,
          completed_at: null,
          last_error: null,
          completion_summary: null,
        },
      ],
    })
    const fetchRun = vi.fn().mockResolvedValue({
      run: {
        id: 'run-1',
        ticket_id: 'ticket-1',
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
        current_step_status: 'running_tests',
        current_step_summary: 'Running backend checks.',
        created_at: '2026-04-01T10:05:00Z',
        runtime_started_at: '2026-04-01T10:05:30Z',
        last_heartbeat_at: '2026-04-01T10:07:00Z',
        terminal_at: null,
        completed_at: null,
        last_error: null,
        completion_summary: null,
      },
      trace_entries: [],
      step_entries: [],
    })

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      fetchRuns,
      fetchRun,
    })

    await state.load('project-1', 'ticket-1')

    expect(fetchRuns).not.toHaveBeenCalled()
    expect(fetchRun).not.toHaveBeenCalled()
    expect(state.runsLoaded).toBe(false)

    await state.ensureRunsLoaded('project-1', 'ticket-1')

    expect(fetchRuns).toHaveBeenCalledTimes(1)
    expect(fetchRun).toHaveBeenCalledTimes(1)
    expect(state.runsLoaded).toBe(true)
    expect(state.runs[0]?.adapterType).toBe('codex-app-server')
    expect(state.runs[0]?.modelName).toBe('gpt-5.4')
    expect(state.runs[0]?.usage.prompt).toBe(920)

    await state.ensureRunsLoaded('project-1', 'ticket-1')
    expect(fetchRuns).toHaveBeenCalledTimes(1)
  })
})
