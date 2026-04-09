import { afterAll, describe, expect, it, vi } from 'vitest'

import { buildAddDependencyMutation, buildAddExternalLinkMutation } from './mutation-builders'
import type { TicketDetail, TicketReferenceOption } from './types'

const now = vi.spyOn(Date, 'now').mockReturnValue(1234567890)

const currentTicket: TicketDetail = {
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
  archived: false,
  repoScopes: [],
  attemptCount: 0,
  consecutiveErrors: 0,
  retryPaused: false,
  costTokensInput: 0,
  costTokensOutput: 0,
  costTokensTotal: 0,
  costAmount: 0,
  budgetUsd: 0,
  dependencies: [],
  externalLinks: [],
  children: [],
  createdBy: 'codex',
  createdAt: '2026-04-01T09:00:00Z',
  updatedAt: '2026-04-01T09:00:00Z',
}

const availableTickets: TicketReferenceOption[] = [
  {
    id: 'ticket-2',
    identifier: 'ASE-2',
    title: 'Blocking ticket',
  },
]

describe('buildAddDependencyMutation', () => {
  it('accepts blocked_by and keeps the optimistic relationship direction', () => {
    const result = buildAddDependencyMutation(currentTicket, availableTickets, {
      targetTicketId: 'ticket-2',
      relation: 'blocked_by',
    })

    expect(result.ok).toBe(true)
    if (!result.ok) return

    expect(result.value.body).toEqual({
      target_ticket_id: 'ticket-2',
      type: 'blocked_by',
    })
    expect(result.value.optimisticUpdate(currentTicket).dependencies).toEqual([
      {
        id: 'pending-ticket-2',
        targetId: 'ticket-2',
        identifier: 'ASE-2',
        title: 'Blocking ticket',
        relation: 'blocked_by',
        stage: 'unstarted',
      },
    ])
  })
})

describe('buildAddExternalLinkMutation', () => {
  it('treats type as optional and does not synthesize a relation fallback', () => {
    const result = buildAddExternalLinkMutation({
      type: '   ',
      url: ' https://docs.example.com/spec ',
      externalId: ' SPEC-1 ',
      title: ' Product spec ',
      status: ' open ',
    })

    expect(result.ok).toBe(true)
    if (!result.ok) return

    expect(result.value.body).toEqual({
      url: 'https://docs.example.com/spec',
      external_id: 'SPEC-1',
      title: 'Product spec',
      status: 'open',
    })
    expect(result.value.optimisticUpdate(currentTicket).externalLinks).toEqual([
      {
        id: 'pending-1234567890',
        url: 'https://docs.example.com/spec',
        externalId: 'SPEC-1',
        title: 'Product spec',
        status: 'open',
      },
    ])
  })

  it('preserves a provided freeform type', () => {
    const result = buildAddExternalLinkMutation({
      type: 'review doc',
      url: 'https://docs.example.com/review',
      externalId: 'DOC-9',
      title: '',
      status: '',
    })

    expect(result.ok).toBe(true)
    if (!result.ok) return

    expect(result.value.body.type).toBe('review doc')
    expect(result.value.optimisticUpdate(currentTicket).externalLinks[0]?.type).toBe('review doc')
  })
})

afterAll(() => {
  now.mockRestore()
})
