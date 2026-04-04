import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail } from '../types'
import TicketSummary from './ticket-summary.svelte'

const ticket: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-2',
  title: 'Show ticket token usage',
  description: 'Expose input and output tokens in the detail sidebar.',
  status: { id: 'todo', name: 'Todo', color: '#94a3b8' },
  priority: 'high',
  type: 'feature',
  archived: false,
  repoScopes: [],
  attemptCount: 3,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 644414,
  costTokensOutput: 18598,
  costAmount: 0,
  budgetUsd: 10,
  dependencies: [],
  externalLinks: [],
  children: [],
  createdBy: 'user:tester',
  createdAt: '2026-04-01T11:30:00Z',
  updatedAt: '2026-04-01T11:30:00Z',
}

describe('TicketSummary', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('shows input and output token counters in the detail metadata', () => {
    const { getByText } = render(TicketSummary, {
      props: {
        ticket,
        availableTickets: [],
      },
    })

    expect(getByText('Input Tokens')).toBeTruthy()
    expect(getByText('644,414')).toBeTruthy()
    expect(getByText('Output Tokens')).toBeTruthy()
    expect(getByText('18,598')).toBeTruthy()
  })
})
