import { describe, expect, it, vi } from 'vitest'

import { isAppContextFresh, mergeProjectIntoAppContext } from './project-shell-state'

describe('project shell state', () => {
  it('treats a recent successful workspace app-context fetch as fresh even when it returned zero organizations', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-03T12:00:00Z'))

    expect(isAppContextFresh('none::', 'none::', Date.now() - 5_000)).toBe(true)

    vi.useRealTimers()
  })

  it('treats app-context as stale after the freshness window expires', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-03T12:00:00Z'))

    expect(isAppContextFresh('none::', 'none::', Date.now() - 31_000)).toBe(false)
    expect(isAppContextFresh('none::', 'org:org-1:', Date.now() - 5_000)).toBe(false)

    vi.useRealTimers()
  })

  it('merges a refreshed project into both the project list and current selection', () => {
    const currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'Before',
      slug: 'before',
      description: 'Old description',
      status: 'Planned',
      default_agent_provider_id: '',
      max_concurrent_agents: 0,
      effective_agent_run_summary_prompt: '',
      agent_run_summary_prompt_source: 'builtin' as const,
      accessible_machine_ids: [],
    }

    expect(
      mergeProjectIntoAppContext(
        [currentProject, { ...currentProject, id: 'project-2', name: 'Other', slug: 'other' }],
        currentProject,
        { ...currentProject, name: 'After', description: 'New description' },
      ),
    ).toEqual({
      projects: [
        { ...currentProject, name: 'After', description: 'New description' },
        { ...currentProject, id: 'project-2', name: 'Other', slug: 'other' },
      ],
      currentProject: { ...currentProject, name: 'After', description: 'New description' },
    })
  })
})
