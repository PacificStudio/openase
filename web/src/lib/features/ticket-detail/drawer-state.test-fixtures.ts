import { vi } from 'vitest'
import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'

export function buildLiveContext(
  overrides: Partial<TicketDetailLiveContext> = {},
): TicketDetailLiveContext {
  return {
    ticket: {
      id: 'ticket-1',
      identifier: 'ASE-336',
      title: 'Align Ticket Detail refresh wiring',
      description: 'Initial description',
      archived: false,
      status: { id: 'status-1', name: 'Todo', color: '#2563eb' },
      priority: 'high',
      type: 'feature',
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
      createdAt: '2026-03-29T10:00:00Z',
      updatedAt: '2026-03-29T10:00:00Z',
    },
    timeline: [
      {
        id: 'description:ticket-1',
        ticketId: 'ticket-1',
        kind: 'description',
        actor: { name: 'codex', type: 'user' },
        title: 'Align Ticket Detail refresh wiring',
        bodyMarkdown: 'Initial description',
        createdAt: '2026-03-29T10:00:00Z',
        updatedAt: '2026-03-29T10:00:00Z',
        editedAt: undefined,
        isCollapsible: false,
        isDeleted: false,
        identifier: 'ASE-336',
      },
    ],
    hooks: [],
    ...overrides,
  }
}

export function buildReferenceData(
  overrides: Partial<TicketDetailProjectReferenceData> = {},
): TicketDetailProjectReferenceData {
  return {
    statusLookup: [{ id: 'status-1', stage: 'unstarted', color: '#2563eb' }],
    statuses: [{ id: 'status-1', name: 'Todo', color: '#2563eb' }],
    dependencyCandidatesByTicketId: [{ id: 'ticket-2', identifier: 'ASE-337', title: 'Follow-up' }],
    repoOptions: [{ id: 'repo-1', name: 'openase', defaultBranch: 'main' }],
    ...overrides,
  }
}

export function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

export function createRunDeps() {
  return {
    fetchRuns: vi.fn().mockResolvedValue({ runs: [] }),
    fetchRun: vi.fn().mockResolvedValue({ run: null, trace_entries: [], step_entries: [] }),
  }
}
