import { describe, expect, it } from 'vitest'

import {
  createProjectDraft,
  parseProjectDraft,
  projectStatusI18nKey,
  projectStatusOptionsForCreate,
} from './model'

describe('catalog creation model', () => {
  it('defaults new projects to the canonical Planned status', () => {
    expect(createProjectDraft().status).toBe('Planned')
    expect(createProjectDraft().maxConcurrentAgents).toBe('')
  })

  it('maps each canonical project status to its catalog dialog i18n key', () => {
    expect(projectStatusI18nKey('Backlog')).toBe('catalog.project.dialog.status.backlog')
    expect(projectStatusI18nKey('Planned')).toBe('catalog.project.dialog.status.planned')
    expect(projectStatusI18nKey('In Progress')).toBe(
      'catalog.project.dialog.status.inProgress',
    )
    expect(projectStatusI18nKey('Completed')).toBe('catalog.project.dialog.status.completed')
    expect(projectStatusI18nKey('Canceled')).toBe('catalog.project.dialog.status.canceled')
    expect(projectStatusI18nKey('Archived')).toBe('catalog.project.dialog.status.archived')
  })

  it('limits create-flow status options to non-terminal statuses', () => {
    expect(projectStatusOptionsForCreate).toEqual(['Backlog', 'Planned', 'In Progress'])
    expect(projectStatusOptionsForCreate).not.toContain('Completed')
    expect(projectStatusOptionsForCreate).not.toContain('Canceled')
    expect(projectStatusOptionsForCreate).not.toContain('Archived')
  })

  it('still accepts terminal statuses when parsing a project draft', () => {
    for (const status of ['Completed', 'Canceled', 'Archived'] as const) {
      const parsed = parseProjectDraft({
        ...createProjectDraft(),
        name: 'OpenASE',
        slug: 'openase',
        status,
      })

      expect(parsed).toEqual({
        ok: true,
        value: {
          name: 'OpenASE',
          slug: 'openase',
          description: '',
          status,
          max_concurrent_agents: undefined,
          default_agent_provider_id: undefined,
        },
      })
    }
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
