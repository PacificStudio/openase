import { describe, expect, it } from 'vitest'

import { mapTicketRun } from './run-transcript-data'

describe('mapTicketRun', () => {
  it('maps legacy stalled status to ended and preserves terminal timestamps', () => {
    const run = mapTicketRun({
      id: 'run-1',
      attempt_number: 1,
      agent_id: 'agent-1',
      agent_name: 'Runner',
      provider: 'Codex',
      adapter_type: 'codex-app-server',
      model_name: 'gpt-5.4',
      usage: {
        total: 25,
        input: 20,
        output: 5,
        cached_input: 3,
        cache_creation: 2,
        reasoning: 1,
        prompt: 18,
        candidate: 4,
        tool: 2,
      },
      status: 'stalled',
      current_step_status: null,
      current_step_summary: null,
      created_at: '2026-04-03T12:00:00Z',
      runtime_started_at: '2026-04-03T12:00:10Z',
      last_heartbeat_at: '2026-04-03T12:01:00Z',
      terminal_at: '2026-04-03T12:02:00Z',
      completed_at: null,
      last_error: null,
      completion_summary: undefined,
    })

    expect(run.status).toBe('ended')
    expect(run.adapterType).toBe('codex-app-server')
    expect(run.modelName).toBe('gpt-5.4')
    expect(run.usage.cacheCreation).toBe(2)
    expect(run.terminalAt).toBe('2026-04-03T12:02:00Z')
    expect(run.completedAt).toBeUndefined()
  })
})
