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
  archived: false,
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
  pickupDiagnosis: {
    state: 'running',
    primaryReasonCode: 'running_current_run',
    primaryReasonMessage: 'Ticket already has an active run.',
    nextActionHint: 'Wait for the current run to finish or inspect the active runtime.',
    reasons: [
      {
        code: 'running_current_run',
        message: 'Current run is still attached to the ticket.',
        severity: 'info',
      },
    ],
    workflow: {
      id: 'workflow-1',
      name: 'Coding Workflow',
      isActive: true,
      pickupStatusMatch: true,
    },
    agent: {
      id: 'agent-1',
      name: 'workflow-seed',
      runtimeControlState: 'active',
    },
    provider: {
      id: 'provider-1',
      name: 'OpenAI Codex',
      machineId: 'machine-1',
      machineName: 'builder-01',
      machineStatus: 'online',
      availabilityState: 'available',
    },
    retry: {
      attemptCount: 3,
      retryPaused: false,
    },
    capacity: {
      workflow: { limited: true, activeRuns: 1, capacity: 2 },
      project: { limited: true, activeRuns: 1, capacity: 5 },
      provider: { limited: true, activeRuns: 1, capacity: 3 },
      status: { limited: false, activeRuns: 1 },
    },
    blockedBy: [],
  },
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

  it('renders diagnosis-driven content instead of the legacy fallback summary', () => {
    const { getByText, queryByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'unavailable',
            primaryReasonCode: 'provider_unavailable',
            primaryReasonMessage: 'Provider is unavailable.',
            nextActionHint: 'Fix the provider health issue before expecting pickup.',
            provider: {
              ...baseTicket.pickupDiagnosis!.provider!,
              availabilityState: 'unavailable',
              availabilityReason: 'not_ready',
            },
          },
        },
      },
    })

    expect(getByText('Unavailable')).toBeTruthy()
    expect(getByText('Provider is unavailable.')).toBeTruthy()
    expect(getByText('Provider')).toBeTruthy()
    expect(getByText('OpenAI Codex · Unavailable (CLI not ready)')).toBeTruthy()
    expect(queryByText('No agent runtime attached yet.')).toBeNull()
  })

  it('shows a live retry countdown with deterministic UTC formatting', async () => {
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'waiting',
            primaryReasonCode: 'retry_backoff',
            primaryReasonMessage: 'Waiting for retry backoff to expire.',
            retry: {
              attemptCount: 3,
              retryPaused: false,
              nextRetryAt: '2026-04-01T12:03:12Z',
            },
          },
        },
      },
    })

    expect(getByText('Retrying in 3m 12s (at 2026-04-01 12:03:12 UTC)')).toBeTruthy()

    await vi.advanceTimersByTimeAsync(2000)

    expect(getByText('Retrying in 3m 10s (at 2026-04-01 12:03:12 UTC)')).toBeTruthy()
  })

  it('does not show a negative retry countdown after expiry', async () => {
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'waiting',
            primaryReasonCode: 'retry_backoff',
            primaryReasonMessage: 'Waiting for retry backoff to expire.',
            retry: {
              attemptCount: 3,
              retryPaused: false,
              nextRetryAt: '2026-04-01T12:00:02Z',
            },
          },
        },
      },
    })

    await vi.advanceTimersByTimeAsync(4000)

    expect(getByText('Retry window elapsed (at 2026-04-01 12:00:02 UTC)')).toBeTruthy()
  })

  it('renders blocked dependencies from the backend diagnosis', () => {
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'blocked',
            primaryReasonCode: 'blocked_dependency',
            primaryReasonMessage: 'Waiting for blocking tickets to finish.',
            blockedBy: [
              {
                id: 'ticket-2',
                identifier: 'ASE-77',
                title: 'Stabilize project conversation restore',
                statusId: 'review',
                statusName: 'In Review',
              },
            ],
          },
        },
      },
    })

    expect(getByText('Waiting for blocking tickets to finish.')).toBeTruthy()
    expect(getByText('Dependencies')).toBeTruthy()
    expect(getByText('ASE-77 Stabilize project conversation restore')).toBeTruthy()
  })

  it('keeps the repeated-stall continue retry action when diagnosis requires manual retry', async () => {
    const onResumeRetry = vi.fn()
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          assignedAgent: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'blocked',
            primaryReasonCode: 'retry_paused_repeated_stalls',
            primaryReasonMessage: 'Retries are paused after repeated stalls.',
            retry: {
              attemptCount: 3,
              retryPaused: true,
              pauseReason: 'repeated_stalls',
            },
          },
        },
        onResumeRetry,
      },
    })

    expect(getByText('Continue Retry')).toBeTruthy()

    await fireEvent.click(getByText('Continue Retry'))

    expect(onResumeRetry).toHaveBeenCalledTimes(1)
  })

  it('shows a reset workspace action when no run is attached', async () => {
    const onResetWorkspace = vi.fn()
    const { getByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          currentRunId: undefined,
          pickupDiagnosis: {
            ...baseTicket.pickupDiagnosis!,
            state: 'runnable',
            primaryReasonCode: 'ready_for_pickup',
            primaryReasonMessage: 'Ready for rerun.',
          },
        },
        onResetWorkspace,
      },
    })

    await fireEvent.click(getByText('Reset Workspace'))

    expect(onResetWorkspace).toHaveBeenCalledTimes(1)
  })

  it('hides the reset workspace action while a run is still attached', () => {
    const { queryByText } = render(TicketRuntimeStateCard, {
      props: {
        ticket: {
          ...baseTicket,
          currentRunId: 'run-1',
        },
        onResetWorkspace: vi.fn(),
      },
    })

    expect(queryByText('Reset Workspace')).toBeNull()
  })
})
