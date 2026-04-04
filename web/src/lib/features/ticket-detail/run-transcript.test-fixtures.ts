import type { TicketRun, TicketRunDetail } from './types'

export const latestRun: TicketRun = {
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

export const olderRun: TicketRun = {
  id: 'run-1',
  attemptNumber: 1,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'Codex',
  status: 'completed',
  createdAt: '2026-04-01T09:00:00Z',
}

export function buildHydratedRunDetail(): TicketRunDetail {
  return {
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
        output: 'functions.exec_command',
        payload: { tool: 'functions.exec_command', arguments: { cmd: 'pnpm vitest run' } },
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
        payload: { item_id: 'command-1', command: 'pnpm vitest run' },
        createdAt: '2026-04-01T10:05:38Z',
      },
      {
        id: 'trace-4',
        agentRunId: latestRun.id,
        sequence: 4,
        provider: 'codex',
        kind: 'thread_status',
        stream: 'system',
        output: 'active · waitingOnUserInput',
        payload: { status: 'active', active_flags: ['waitingOnUserInput'] },
        createdAt: '2026-04-01T10:05:39Z',
      },
      {
        id: 'trace-5',
        agentRunId: latestRun.id,
        sequence: 5,
        provider: 'codex',
        kind: 'reasoning_updated',
        stream: 'reasoning',
        output: 'Inspecting the reducer.',
        payload: { kind: 'text_delta', content_index: 0 },
        createdAt: '2026-04-01T10:05:40Z',
      },
      {
        id: 'trace-6',
        agentRunId: latestRun.id,
        sequence: 6,
        provider: 'codex',
        kind: 'turn_diff_updated',
        stream: 'diff',
        output: ['diff --git a/app.ts b/app.ts', '@@ -1 +1 @@', '-old', '+new'].join('\n'),
        payload: {},
        createdAt: '2026-04-01T10:05:41Z',
      },
    ],
  }
}

export function buildNewerRun(overrides: Partial<TicketRun> = {}): TicketRun {
  return {
    ...latestRun,
    id: 'run-3',
    attemptNumber: 3,
    status: 'ready',
    createdAt: '2026-04-01T11:00:00Z',
    runtimeStartedAt: '2026-04-01T11:00:10Z',
    currentStepStatus: undefined,
    currentStepSummary: undefined,
    ...overrides,
  }
}
