import { describe, expect, it } from 'vitest'

import { createProjectDraft, parseProjectDraft } from './model'

describe('catalog creation model', () => {
  it('defaults new projects to the canonical Planned status', () => {
    expect(createProjectDraft().status).toBe('Planned')
    expect(createProjectDraft().maxConcurrentAgents).toBe('')
  })

  it('accepts canonical project statuses without rewriting them', () => {
    const parsed = parseProjectDraft({
      ...createProjectDraft(),
      name: 'OpenASE',
      slug: 'openase',
      status: 'In Progress',
    })

    expect(parsed).toEqual({
      ok: true,
      value: {
        name: 'OpenASE',
        slug: 'openase',
        description: '',
        status: 'In Progress',
        max_concurrent_agents: undefined,
        default_agent_provider_id: undefined,
      },
    })
  })

  it('rejects legacy, lowercase, and whitespace-padded project statuses', () => {
    for (const status of ['active', 'planned', ' In Progress ']) {
      const parsed = parseProjectDraft({
        ...createProjectDraft(),
        name: 'OpenASE',
        slug: 'openase',
        status,
      })

      expect(parsed.ok).toBe(false)
      expect(parsed).toEqual({
        ok: false,
        error:
          'Project status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived.',
      })
    }
  })

  it('treats a blank max concurrent input as unlimited and rejects non-positive integers', () => {
    const unlimited = parseProjectDraft({
      ...createProjectDraft(),
      name: 'OpenASE',
      slug: 'openase',
      maxConcurrentAgents: '',
    })
    expect(unlimited).toEqual({
      ok: true,
      value: {
        name: 'OpenASE',
        slug: 'openase',
        description: '',
        status: 'Planned',
        max_concurrent_agents: undefined,
        default_agent_provider_id: undefined,
      },
    })

    const invalid = parseProjectDraft({
      ...createProjectDraft(),
      name: 'OpenASE',
      slug: 'openase',
      maxConcurrentAgents: '0',
    })
    expect(invalid).toEqual({
      ok: false,
      error: 'Max concurrent agents must be a positive integer.',
    })
  })
})
