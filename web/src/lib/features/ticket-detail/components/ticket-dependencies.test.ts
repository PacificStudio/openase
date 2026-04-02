import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail, TicketReferenceOption } from '../types'
import TicketDependencies from './ticket-dependencies.svelte'

const ticket: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-1',
  title: 'Current ticket',
  description: '',
  status: {
    id: 'status-1',
    name: 'Todo',
    color: '#2563eb',
  },
  priority: 'medium',
  type: 'feature',
  repoScopes: [],
  attemptCount: 0,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
  costAmount: 0,
  budgetUsd: 0,
  dependencies: [
    {
      id: 'dep-1',
      targetId: 'ticket-2',
      identifier: 'ASE-2',
      title: 'Platform migration',
      relation: 'blocked_by',
      stage: 'started',
    },
    {
      id: 'dep-2',
      targetId: 'ticket-3',
      identifier: 'ASE-3',
      title: 'Frontend cleanup',
      relation: 'blocks',
      stage: 'unstarted',
    },
  ],
  externalLinks: [],
  children: [],
  createdBy: 'codex',
  createdAt: '2026-04-01T09:00:00Z',
  updatedAt: '2026-04-01T09:00:00Z',
}

const availableTickets: TicketReferenceOption[] = [
  { id: 'ticket-2', identifier: 'ASE-2', title: 'Platform migration' },
  { id: 'ticket-3', identifier: 'ASE-3', title: 'Frontend cleanup' },
]

describe('TicketDependencies', () => {
  afterEach(() => {
    cleanup()
  })

  it('renders blocked-by and blocking relationships as separate groups', () => {
    const onDeleteDependency = vi.fn()
    const { getAllByText, getByText, getByRole } = render(TicketDependencies, {
      props: {
        ticket,
        availableTickets,
        onDeleteDependency,
      },
    })

    expect(getAllByText('Blocked by').length).toBeGreaterThanOrEqual(2)
    expect(getByText('Blocking')).toBeTruthy()
    expect(getByText('Platform migration')).toBeTruthy()
    expect(getByText('Frontend cleanup')).toBeTruthy()
    expect(getByRole('button', { name: 'Remove ASE-2 relationship' })).toBeTruthy()
    expect(getByRole('button', { name: 'Remove ASE-3 relationship' })).toBeTruthy()
  })
})
