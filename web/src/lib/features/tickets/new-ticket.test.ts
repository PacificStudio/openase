import { describe, expect, it } from 'vitest'

import { createNewTicketDraft, parseNewTicketDraft, type TicketRepoOption } from './new-ticket'

const statusOptions = [{ id: 'todo', label: 'Todo', color: '#94a3b8', stage: 'unstarted' as const }]

describe('new ticket repo scope parsing', () => {
  it('auto-selects the only repository in single-repo projects', () => {
    const repoOptions: TicketRepoOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
    ]

    const draft = createNewTicketDraft(statusOptions, repoOptions)
    const parsed = parseNewTicketDraft(
      {
        ...draft,
        title: 'Ship login validation',
      },
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      payload: {
        title: 'Ship login validation',
        priority: null,
        status_id: 'todo',
        repo_scopes: [{ repo_id: 'repo-1' }],
      },
    })
  })

  it('requires explicit repo scopes for multi-repo projects', () => {
    const repoOptions: TicketRepoOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
    ]

    const parsed = parseNewTicketDraft(
      {
        ...createNewTicketDraft(statusOptions, repoOptions),
        title: 'Ship login validation',
      },
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: false,
      error: 'Select at least one repository scope for this ticket.',
    })
  })

  it('returns every explicitly selected repo scope for multi-repo projects', () => {
    const repoOptions: TicketRepoOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
      { id: 'repo-3', label: 'docs', defaultBranch: 'main' },
    ]

    const parsed = parseNewTicketDraft(
      {
        ...createNewTicketDraft(statusOptions, repoOptions),
        title: 'Ship login validation',
        repoIds: ['repo-2', 'repo-1'],
      },
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      payload: {
        title: 'Ship login validation',
        priority: null,
        status_id: 'todo',
        repo_scopes: [{ repo_id: 'repo-1' }, { repo_id: 'repo-2' }],
      },
    })
  })

  it('includes explicit repo branch overrides only when the user provides them', () => {
    const repoOptions: TicketRepoOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
    ]

    const parsed = parseNewTicketDraft(
      {
        ...createNewTicketDraft(statusOptions, repoOptions),
        title: 'Ship login validation',
        repoIds: ['repo-2', 'repo-1'],
        repoBranchOverrides: {
          'repo-2': 'release/candidate',
        },
      },
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      payload: {
        title: 'Ship login validation',
        priority: null,
        status_id: 'todo',
        repo_scopes: [
          { repo_id: 'repo-1' },
          { repo_id: 'repo-2', branch_name: 'release/candidate' },
        ],
      },
    })
  })

  it('preserves an explicit priority selection when provided', () => {
    const repoOptions: TicketRepoOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
    ]

    const parsed = parseNewTicketDraft(
      {
        ...createNewTicketDraft(statusOptions, repoOptions),
        title: 'Ship login validation',
        priority: 'high',
      },
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      payload: {
        title: 'Ship login validation',
        priority: 'high',
        status_id: 'todo',
        repo_scopes: [{ repo_id: 'repo-1' }],
      },
    })
  })
})
