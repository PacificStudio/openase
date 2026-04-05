import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail, TicketRun, TicketRunTranscriptBlock } from '../types'
import TicketRunHistoryPanel from './ticket-run-history-panel.svelte'

const ticket: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-434',
  title: 'Run history',
  description: 'Track each ticket run separately.',
  status: { id: 'todo', name: 'Todo', color: '#94a3b8' },
  priority: 'high',
  type: 'feature',
  archived: false,
  repoScopes: [],
  attemptCount: 2,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
  costTokensTotal: 0,
  costAmount: 0,
  budgetUsd: 50,
  dependencies: [],
  externalLinks: [],
  children: [],
  createdBy: 'user:tester',
  createdAt: '2026-04-01T09:00:00Z',
  updatedAt: '2026-04-01T09:00:00Z',
}

const latestRun: TicketRun = {
  id: 'run-2',
  attemptNumber: 2,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'OpenAI Codex',
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
  completionSummary: {
    status: 'pending',
  },
}

const failedRun: TicketRun = {
  id: 'run-1',
  attemptNumber: 1,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'OpenAI Codex',
  adapterType: 'codex-app-server',
  modelName: 'gpt-5.4',
  usage: {
    total: 880,
    input: 650,
    output: 230,
    cachedInput: 90,
    cacheCreation: 20,
    reasoning: 30,
    prompt: 520,
    candidate: 180,
    tool: 15,
  },
  status: 'failed',
  createdAt: '2026-04-01T09:00:00Z',
  completedAt: '2026-04-01T09:10:00Z',
  lastError: 'Unit tests failed.',
  completionSummary: {
    status: 'failed',
    error: 'provider unavailable after run completion',
  },
}

const latestBlocks: TicketRunTranscriptBlock[] = [
  {
    kind: 'phase',
    id: 'phase:launching:1',
    phase: 'launching',
    at: '2026-04-01T10:05:00Z',
    summary: 'Run created.',
  },
  {
    kind: 'tool_call',
    id: 'tool:1',
    toolName: 'functions.exec_command',
    arguments: { cmd: 'pnpm vitest run', workdir: '/repo' },
    at: '2026-04-01T10:05:31Z',
  },
  {
    kind: 'terminal_output',
    id: 'terminal_output:1',
    stream: 'command',
    command: 'pnpm vitest run',
    text: 'PASS src/app.test.ts',
    streaming: true,
  },
]

const failedBlocks: TicketRunTranscriptBlock[] = [
  {
    kind: 'phase',
    id: 'phase:launching:2',
    phase: 'launching',
    at: '2026-04-01T09:00:00Z',
    summary: 'Run created.',
  },
]

describe('TicketRunHistoryPanel', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('lists attempts in order and marks the latest run as live', () => {
    const { getAllByRole, getAllByText: getAllStatuses } = render(TicketRunHistoryPanel, {
      props: {
        ticket,
        runs: [latestRun, failedRun],
        currentRun: latestRun,
        blocks: latestBlocks,
        runStreamState: 'live',
      },
    })

    // Chronological order: oldest first
    const buttons = getAllByRole('button', { name: /View Attempt / })
    expect(buttons[0]?.textContent).toContain('#1')
    expect(buttons[1]?.textContent).toContain('#2')
    // Status badge on attempt row only (no separate header badge)
    expect(getAllStatuses('Running')).toHaveLength(1)
    expect(getAllStatuses('Summary pending')).toHaveLength(1)
  })

  it('requests a run switch and renders historical terminal summaries', async () => {
    const onSelectRun = vi.fn()
    const { getByRole, getAllByText, getByText, rerender } = render(TicketRunHistoryPanel, {
      props: {
        ticket,
        runs: [latestRun, failedRun],
        currentRun: latestRun,
        blocks: latestBlocks,
        onSelectRun,
      },
    })

    await fireEvent.click(getByRole('button', { name: 'View Attempt 1' }))
    expect(onSelectRun).toHaveBeenCalledWith(failedRun.id)

    await rerender({
      ticket,
      runs: [latestRun, failedRun],
      currentRun: failedRun,
      blocks: failedBlocks,
      onSelectRun,
    })

    expect(getAllByText('Failed').length).toBeGreaterThanOrEqual(1)
    expect(getByText('Summary failed')).toBeTruthy()
    expect(getByText('provider unavailable after run completion')).toBeTruthy()
  })

  it('renders command/tool metadata instead of collapsing them into opaque noise', async () => {
    const { getByText, queryByText } = render(TicketRunHistoryPanel, {
      props: {
        ticket,
        runs: [latestRun, failedRun],
        currentRun: latestRun,
        blocks: latestBlocks,
        runStreamState: 'live',
      },
    })

    // Tool call shows summarized command; terminal output shows raw command
    expect(getByText(/Ran .pnpm vitest run/)).toBeTruthy()
    expect(getByText('pnpm vitest run')).toBeTruthy()
    expect(queryByText(/operations/)).toBeNull()

    // Expanding tool call reveals target workdir
    await fireEvent.click(getByText(/Ran .pnpm vitest run/))
    expect(getByText('/repo')).toBeTruthy()
  })

  it('renders completed summaries without hiding the raw transcript', () => {
    const completedRun: TicketRun = {
      ...failedRun,
      id: 'run-3',
      attemptNumber: 3,
      status: 'completed',
      completionSummary: {
        status: 'completed',
        markdown: '## Overview\n\nImplemented the ticket flow.\n\n## Outcome\n\nDone.',
        generatedAt: '2026-04-01T11:30:00Z',
      },
    }

    const { getByText } = render(TicketRunHistoryPanel, {
      props: {
        ticket,
        runs: [completedRun],
        currentRun: completedRun,
        blocks: failedBlocks,
      },
    })

    expect(getByText('Summary')).toBeTruthy()
    expect(getByText('Implemented the ticket flow.')).toBeTruthy()
    expect(getByText('Done.')).toBeTruthy()
    expect(getByText('Run created.')).toBeTruthy()
  })
})
