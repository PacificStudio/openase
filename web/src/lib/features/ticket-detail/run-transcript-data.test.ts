import { describe, expect, it } from 'vitest'

import type { TicketRunDetailPayload } from '$lib/api/contracts'
import { mapTicketRun, mapTicketRunDetail } from './run-transcript-data'

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

describe('mapTicketRunDetail', () => {
  it('falls back to transcript item cursors when page cursors are malformed', () => {
    const payload: TicketRunDetailPayload = {
      run: {
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
        status: 'ready',
        current_step_status: null,
        current_step_summary: null,
        created_at: '2026-04-03T12:00:00Z',
        runtime_started_at: '2026-04-03T12:00:10Z',
        last_heartbeat_at: '2026-04-03T12:01:00Z',
        terminal_at: null,
        completed_at: null,
        last_error: null,
        completion_summary: undefined,
      },
      transcript_page: {
        items: [
          {
            kind: 'step',
            cursor: ' 2026-04-03T12:00:11Z|step|0|step-1 ',
            trace_entry: {
              id: 'trace-unused',
              agent_run_id: 'run-1',
              sequence: 0,
              provider: 'codex',
              kind: 'assistant_delta',
              stream: 'assistant',
              output: '',
              payload: {},
              created_at: '2026-04-03T12:00:11Z',
            },
            step_entry: {
              id: 'step-1',
              agent_run_id: 'run-1',
              step_status: 'running_command',
              summary: 'Running tests.',
              source_trace_event_id: null,
              created_at: '2026-04-03T12:00:11Z',
            },
          },
          {
            kind: 'trace',
            cursor: '2026-04-03T12:00:12Z|trace|2|trace-2',
            step_entry: {
              id: 'step-unused',
              agent_run_id: 'run-1',
              step_status: 'completed',
              summary: 'Unused step payload.',
              source_trace_event_id: null,
              created_at: '2026-04-03T12:00:12Z',
            },
            trace_entry: {
              id: 'trace-2',
              agent_run_id: 'run-1',
              sequence: 2,
              provider: 'codex',
              kind: 'assistant_delta',
              stream: 'assistant',
              output: 'Tests passed.',
              payload: {},
              created_at: '2026-04-03T12:00:12Z',
            },
          },
        ],
        has_older: true,
        hidden_older_count: 4,
        has_newer: false,
        hidden_newer_count: 0,
        oldest_cursor: 'broken-cursor',
        newest_cursor: 'still-broken',
      },
      trace_entries: [],
      step_entries: [],
    }

    const detail = mapTicketRunDetail(payload)

    expect(detail.transcriptPage.oldestCursor).toBe('2026-04-03T12:00:11Z|step|0|step-1')
    expect(detail.transcriptPage.newestCursor).toBe('2026-04-03T12:00:12Z|trace|2|trace-2')
  })

  it('keeps transcript-page and transcript-entry cursors separate for projected transcript pages', () => {
    const detail = mapTicketRunDetail({
      run: {
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
        status: 'ready',
        current_step_status: null,
        current_step_summary: null,
        created_at: '2026-04-03T12:00:00Z',
        runtime_started_at: '2026-04-03T12:00:10Z',
        last_heartbeat_at: '2026-04-03T12:01:00Z',
        terminal_at: null,
        completed_at: null,
        last_error: null,
        completion_summary: undefined,
      },
      transcript_page: {
        items: [
          {
            kind: 'step',
            cursor: '2026-04-03T11:59:58Z|step|0|11111111-1111-4111-8111-111111111111',
            step_entry: {
              id: '11111111-1111-4111-8111-111111111111',
              agent_run_id: 'run-1',
              step_status: 'running_command',
              summary: 'Running tests.',
              source_trace_event_id: null,
              created_at: '2026-04-03T11:59:58Z',
            },
          },
        ],
        has_older: true,
        hidden_older_count: 4,
        has_newer: false,
        hidden_newer_count: 0,
        oldest_cursor: '2026-04-03T11:59:58Z|step|0|11111111-1111-4111-8111-111111111111',
        newest_cursor: '2026-04-03T11:59:59Z|trace|3|22222222-2222-4222-8222-222222222222',
      },
      transcript_entries_page: {
        entries: [
          {
            id: '33333333-3333-4333-8333-333333333333',
            provider: 'codex',
            entry_key: 'assistant:1',
            entry_kind: 'assistant_message',
            activity_kind: 'assistant_message',
            activity_id: '44444444-4444-4444-8444-444444444444',
            title: 'Assistant',
            summary: 'Summarized output',
            body_text: 'Summarized output',
            command: null,
            tool_name: null,
            metadata: {},
            created_at: '2026-04-03T11:59:58Z',
          },
        ],
        has_older: true,
        hidden_older_count: 4,
        has_newer: false,
        hidden_newer_count: 0,
        oldest_cursor: '2026-04-03T11:59:58Z|33333333-3333-4333-8333-333333333333',
        newest_cursor: '2026-04-03T11:59:58Z|33333333-3333-4333-8333-333333333333',
      },
      activities: [],
      trace_entries: [],
      step_entries: [],
    })

    expect(detail.transcriptPage.oldestCursor).toBe(
      '2026-04-03T11:59:58Z|step|0|11111111-1111-4111-8111-111111111111',
    )
    expect(detail.transcriptPage.newestCursor).toBe(
      '2026-04-03T11:59:59Z|trace|3|22222222-2222-4222-8222-222222222222',
    )
    expect(detail.transcriptPage.oldestEventCursor).toBe(
      '2026-04-03T11:59:58Z|33333333-3333-4333-8333-333333333333',
    )
    expect(detail.transcriptPage.newestEventCursor).toBe(
      '2026-04-03T11:59:58Z|33333333-3333-4333-8333-333333333333',
    )
  })
})
