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
  repoScopes: [],
  attemptCount: 2,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
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
  status: 'executing',
  currentStepStatus: 'running_tests',
  currentStepSummary: 'Running backend checks.',
  createdAt: '2026-04-01T10:05:00Z',
  runtimeStartedAt: '2026-04-01T10:05:30Z',
  lastHeartbeatAt: '2026-04-01T10:07:00Z',
}

const failedRun: TicketRun = {
  id: 'run-1',
  attemptNumber: 1,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'OpenAI Codex',
  status: 'failed',
  createdAt: '2026-04-01T09:00:00Z',
  completedAt: '2026-04-01T09:10:00Z',
  lastError: 'Unit tests failed.',
}

const latestBlocks: TicketRunTranscriptBlock[] = [
  {
    kind: 'phase',
    id: 'phase:launching:1',
    phase: 'launching',
    at: '2026-04-01T10:05:00Z',
    summary: 'Run created.',
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
  {
    kind: 'result',
    id: 'result:failed:run-1',
    outcome: 'failed',
    summary: 'Unit tests failed.',
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
    const { getAllByRole, getAllByText } = render(TicketRunHistoryPanel, {
      props: {
        ticket,
        runs: [latestRun, failedRun],
        currentRun: latestRun,
        blocks: latestBlocks,
        runStreamState: 'live',
      },
    })

    const buttons = getAllByRole('button', { name: /View Attempt / })
    expect(buttons[0]?.textContent).toContain('Attempt 2')
    expect(buttons[1]?.textContent).toContain('Attempt 1')
    expect(getAllByText('Live')).toHaveLength(2)
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

    expect(getByText('Attempt 1 transcript')).toBeTruthy()
    expect(getAllByText('Failed').length).toBeGreaterThanOrEqual(2)
    expect(getByText('Unit tests failed.')).toBeTruthy()
  })
})
