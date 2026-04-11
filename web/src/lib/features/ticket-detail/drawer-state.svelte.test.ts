import { describe, expect, it, vi } from 'vitest'

import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import {
  buildLiveContext,
  buildReferenceData,
  createDeferred,
  createRunDeps,
} from './drawer-state.test-fixtures'
import { createTicketDrawerState } from './drawer-state.svelte'

describe('createTicketDrawerState', () => {
  it('refreshes only the live ticket timeline snapshot', async () => {
    const initialReferenceData = buildReferenceData()
    const initialContext = buildLiveContext()
    const refreshedContext = buildLiveContext({
      ticket: {
        ...initialContext.ticket,
        title: 'Align Ticket Detail SSE refresh wiring',
      },
      timeline: [
        ...initialContext.timeline,
        {
          id: 'comment:comment-1',
          ticketId: 'ticket-1',
          kind: 'comment',
          commentId: 'comment-1',
          actor: { name: 'reviewer', type: 'user' },
          bodyMarkdown: 'Looks good.',
          createdAt: '2026-03-29T10:05:00Z',
          updatedAt: '2026-03-29T10:05:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          editCount: 0,
          revisionCount: 1,
          lastEditedBy: undefined,
        },
      ],
      hooks: [
        {
          id: 'hook-1',
          hookName: 'ticket.timeline.refresh',
          status: 'pass',
          timestamp: '2026-03-29T10:05:00Z',
        },
      ],
    })

    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValueOnce(initialContext)
      .mockResolvedValueOnce(refreshedContext)
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(initialReferenceData)

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    await state.refreshTimeline('project-1', 'ticket-1')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
    expect(state.ticket?.title).toBe('Align Ticket Detail SSE refresh wiring')
    expect(state.timeline).toEqual(refreshedContext.timeline)
    expect(state.hooks).toEqual(refreshedContext.hooks)
    expect(state.statuses).toEqual(initialReferenceData.statuses)
    expect(state.dependencyCandidates).toEqual(initialReferenceData.dependencyCandidatesByTicketId)
    expect(state.repoOptions).toEqual(initialReferenceData.repoOptions)
  })

  it('queues one follow-up refresh when another event arrives mid-refresh', async () => {
    const initialReferenceData = buildReferenceData()
    const initialContext = buildLiveContext()
    const interimContext = buildLiveContext({
      timeline: [
        ...initialContext.timeline,
        {
          id: 'activity:event-1',
          ticketId: 'ticket-1',
          kind: 'activity',
          actor: { name: 'dispatcher', type: 'system' },
          eventType: 'agent_started',
          title: 'agent_started',
          bodyText: 'Agent started work.',
          createdAt: '2026-03-29T10:06:00Z',
          updatedAt: '2026-03-29T10:06:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          metadata: {},
        },
      ],
    })
    const finalContext = buildLiveContext({
      timeline: [
        ...interimContext.timeline,
        {
          id: 'comment:comment-2',
          ticketId: 'ticket-1',
          kind: 'comment',
          commentId: 'comment-2',
          actor: { name: 'reviewer', type: 'user' },
          bodyMarkdown: 'History count updated.',
          createdAt: '2026-03-29T10:07:00Z',
          updatedAt: '2026-03-29T10:07:00Z',
          editedAt: undefined,
          isCollapsible: true,
          isDeleted: false,
          editCount: 0,
          revisionCount: 1,
          lastEditedBy: undefined,
        },
      ],
    })
    const deferredRefresh = createDeferred<TicketDetailLiveContext>()

    const fetchLiveContext = vi
      .fn<
        (
          projectId: string,
          ticketId: string,
          refs: TicketDetailProjectReferenceData,
        ) => Promise<TicketDetailLiveContext>
      >()
      .mockResolvedValueOnce(initialContext)
      .mockReturnValueOnce(deferredRefresh.promise)
      .mockResolvedValueOnce(finalContext)
    const fetchReferenceData = vi
      .fn<(projectId: string) => Promise<TicketDetailProjectReferenceData>>()
      .mockResolvedValue(initialReferenceData)

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })
    await state.load('project-1', 'ticket-1')

    const firstRefresh = state.refreshTimeline('project-1', 'ticket-1')
    const secondRefresh = state.refreshTimeline('project-1', 'ticket-1')

    await Promise.resolve()

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)

    deferredRefresh.resolve(interimContext)
    await Promise.all([firstRefresh, secondRefresh])

    expect(fetchLiveContext).toHaveBeenCalledTimes(3)
    expect(state.timeline).toEqual(finalContext.timeline)
  })

  it('reuses cached project references when opening another ticket in the same project', async () => {
    const referenceData = buildReferenceData({
      dependencyCandidatesByTicketId: [
        { id: 'ticket-1', identifier: 'ASE-336', title: 'Align Ticket Detail refresh wiring' },
        { id: 'ticket-2', identifier: 'ASE-337', title: 'Follow-up' },
        { id: 'ticket-3', identifier: 'ASE-338', title: 'Another ticket' },
      ],
    })
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
      .mockResolvedValueOnce(buildLiveContext())
      .mockResolvedValueOnce(
        buildLiveContext({
          ticket: {
            ...buildLiveContext().ticket,
            id: 'ticket-2',
            identifier: 'ASE-337',
            title: 'Follow-up',
          },
        }),
      )

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    await state.load('project-1', 'ticket-2')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
    expect(state.ticket?.id).toBe('ticket-2')
    expect(state.dependencyCandidates).toEqual([
      { id: 'ticket-1', identifier: 'ASE-336', title: 'Align Ticket Detail refresh wiring' },
      { id: 'ticket-3', identifier: 'ASE-338', title: 'Another ticket' },
    ])
  })

  it('keeps cached project references across drawer reset for the same project', async () => {
    const referenceData = buildReferenceData()
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
      .mockResolvedValue(buildLiveContext())

    const state = createTicketDrawerState({
      fetchLiveContext,
      fetchReferenceData,
      ...createRunDeps(),
    })

    await state.load('project-1', 'ticket-1')
    state.reset()
    await state.load('project-1', 'ticket-1')

    expect(fetchReferenceData).toHaveBeenCalledTimes(1)
    expect(fetchLiveContext).toHaveBeenCalledTimes(2)
  })

  it('uses separate transcript and transcript-entry cursors when loading earlier run history', async () => {
    const fetchRun = vi
      .fn()
      .mockResolvedValueOnce({
        run: {
          id: 'run-1',
          ticket_id: 'ticket-1',
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
          status: 'executing',
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
              cursor: ' 2026-04-03T12:00:11Z|step|0|11111111-1111-4111-8111-111111111111 ',
              step_entry: {
                id: '11111111-1111-4111-8111-111111111111',
                agent_run_id: 'run-1',
                step_status: 'running_command',
                summary: 'Running tests.',
                source_trace_event_id: null,
                created_at: '2026-04-03T12:00:11Z',
              },
            },
            {
              kind: 'trace',
              cursor: '2026-04-03T12:00:12Z|trace|2|22222222-2222-4222-8222-222222222222',
              trace_entry: {
                id: '22222222-2222-4222-8222-222222222222',
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
              summary: 'Running tests.',
              body_text: 'Running tests.',
              command: null,
              tool_name: null,
              metadata: {},
              created_at: '2026-04-03T12:00:11Z',
            },
          ],
          has_older: true,
          hidden_older_count: 4,
          has_newer: false,
          hidden_newer_count: 0,
          oldest_cursor: '2026-04-03T12:00:11Z|33333333-3333-4333-8333-333333333333',
          newest_cursor: '2026-04-03T12:00:11Z|33333333-3333-4333-8333-333333333333',
        },
        activities: [],
        trace_entries: [],
        step_entries: [],
      })
      .mockResolvedValueOnce({
        run: {
          id: 'run-1',
          ticket_id: 'ticket-1',
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
          status: 'executing',
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
          items: [],
          has_older: false,
          hidden_older_count: 0,
          has_newer: true,
          hidden_newer_count: 2,
        },
        trace_entries: [],
        step_entries: [],
      })
    const fetchRunTranscriptEntries = vi
      .fn()
      .mockResolvedValueOnce({
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
              summary: 'Running tests.',
              body_text: 'Running tests.',
              command: null,
              tool_name: null,
              metadata: {},
              created_at: '2026-04-03T12:00:11Z',
            },
          ],
          has_older: true,
          hidden_older_count: 4,
          has_newer: false,
          hidden_newer_count: 0,
          oldest_cursor: '2026-04-03T12:00:11Z|33333333-3333-4333-8333-333333333333',
          newest_cursor: '2026-04-03T12:00:11Z|33333333-3333-4333-8333-333333333333',
        },
      })
      .mockResolvedValueOnce({
        transcript_entries_page: {
          entries: [],
          has_older: false,
          hidden_older_count: 0,
          has_newer: true,
          hidden_newer_count: 2,
          oldest_cursor: '2026-04-03T12:00:08Z|55555555-5555-4555-8555-555555555555',
          newest_cursor: '2026-04-03T12:00:08Z|55555555-5555-4555-8555-555555555555',
        },
      })

    const state = createTicketDrawerState({
      fetchLiveContext: vi.fn().mockResolvedValue(buildLiveContext()),
      fetchReferenceData: vi
        .fn<() => Promise<TicketDetailProjectReferenceData>>()
        .mockResolvedValue(buildReferenceData()),
      ...createRunDeps(),
      fetchRuns: vi.fn().mockResolvedValue({
        runs: [
          {
            id: 'run-1',
            ticket_id: 'ticket-1',
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
            status: 'executing',
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
        ],
      }),
      fetchRun,
      fetchRunTranscriptEntries,
    })

    await state.load('project-1', 'ticket-1')
    await state.ensureRunsLoaded('project-1', 'ticket-1')
    await state.loadOlderRunTranscript('project-1', 'ticket-1', 'run-1')

    expect(fetchRun).toHaveBeenNthCalledWith(1, 'project-1', 'ticket-1', 'run-1')
    expect(fetchRun).toHaveBeenNthCalledWith(2, 'project-1', 'ticket-1', 'run-1', {
      before: '2026-04-03T12:00:11Z|step|0|11111111-1111-4111-8111-111111111111',
    })
    expect(fetchRunTranscriptEntries).toHaveBeenNthCalledWith(1, 'project-1', 'ticket-1', 'run-1')
    expect(fetchRunTranscriptEntries).toHaveBeenNthCalledWith(2, 'project-1', 'ticket-1', 'run-1', {
      before: '2026-04-03T12:00:11Z|33333333-3333-4333-8333-333333333333',
    })
  })
})
