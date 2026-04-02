import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail } from '../types'
import TicketRuntimeStateCard from './ticket-runtime-state-card.svelte'

const baseTicket: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-3',
  title: 'Support active/completed filters',
  description: 'Track workflow runtime state clearly.',
  status: { id: 'todo', name: 'Todo', color: '#94a3b8' },
  priority: 'high',
  type: 'feature',
  assignedAgent: {
    id: 'agent-1',
    name: 'workflow-seed',
    provider: 'OpenAI Codex',
    runtimeControlState: 'active',
    runtimePhase: 'ready',
  },
  repoScopes: [],
  attemptCount: 3,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
  costAmount: 0,
  budgetUsd: 10,
  dependencies: [],
  externalLinks: [],
  children: [],
  createdBy: 'user:tester',
  createdAt: '2026-04-01T11:30:00Z',
  updatedAt: '2026-04-01T11:30:00Z',
  startedAt: '2026-04-01T11:48:00Z',
}

describe('TicketRuntimeStateCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('shows a running summary with the current phase and runtime details', () => {
    const { getByText, queryByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: baseTicket,
      },
    })

    expect(getByText('Current State')).toBeTruthy()
    expect(getByText('State Running')).toBeTruthy()
    expect(getByText('Phase Ready')).toBeTruthy()
    expect(getByText('workflow-seed')).toBeTruthy()
    expect(getByText('OpenAI Codex')).toBeTruthy()
    expect(getByText('12m ago')).toBeTruthy()
    expect(queryByText('Control Active')).toBeNull()
  })

  it('surfaces paused control state when the runtime is paused', () => {
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: {
            ...baseTicket.assignedAgent!,
            runtimeControlState: 'paused',
            runtimePhase: 'executing',
          },
        },
      },
    })

    expect(getByText('State Paused')).toBeTruthy()
    expect(getByText('Control Paused')).toBeTruthy()
  })

  it('surfaces stalled state and continue retry action for repeated stalls', async () => {
    const onResumeRetry = vi.fn()
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          retryPaused: true,
          pauseReason: 'repeated_stalls',
        },
        onResumeRetry,
      },
    })

    expect(getByText('State Stalled')).toBeTruthy()
    await fireEvent.click(getByText('Continue Retry'))
    expect(onResumeRetry).toHaveBeenCalledTimes(1)
  })
})
