import { describe, expect, it } from 'vitest'

import type { Project } from '$lib/api/contracts'
import { buildProjectSummary, shouldShowProjectOnboarding } from './model'

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'TodoApp',
  slug: 'todoapp',
  description: 'Shared ticket workflow and automation dashboard.',
  status: 'Planned',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 5,
}

describe('dashboard model', () => {
  it('preserves raw project status and description in the project summary', () => {
    const summary = buildProjectSummary(
      projectFixture,
      { runningAgents: 2, activeTickets: 3 },
      '2026-03-27T10:00:00Z',
    )

    expect(summary).toEqual({
      id: 'project-1',
      name: 'TodoApp',
      description: 'Shared ticket workflow and automation dashboard.',
      status: 'Planned',
      activeAgents: 2,
      activeTickets: 3,
      lastActivity: '2026-03-27T10:00:00Z',
    })
  })

  it('keeps missing activity as null instead of fabricating a timestamp', () => {
    const summary = buildProjectSummary(
      projectFixture,
      { runningAgents: 0, activeTickets: 0 },
      null,
    )

    expect(summary.lastActivity).toBeNull()
  })

  it('shows onboarding for a newly created empty project even if activity exists elsewhere', () => {
    expect(
      shouldShowProjectOnboarding({
        dismissed: false,
        loading: false,
        stats: { runningAgents: 0, totalTickets: 0 },
        projectId: 'project-1',
        orgId: 'org-1',
      }),
    ).toBe(true)
  })

  it('hides onboarding once any tickets have ever been created', () => {
    expect(
      shouldShowProjectOnboarding({
        dismissed: false,
        loading: false,
        stats: { runningAgents: 0, totalTickets: 1 },
        projectId: 'project-1',
        orgId: 'org-1',
      }),
    ).toBe(false)
  })

  it('hides onboarding when all tickets are completed (activeTickets=0 but totalTickets>0)', () => {
    // Regression: completed tickets drop out of activeTickets but onboarding must stay hidden
    expect(
      shouldShowProjectOnboarding({
        dismissed: false,
        loading: false,
        stats: { runningAgents: 0, totalTickets: 3 },
        projectId: 'project-1',
        orgId: 'org-1',
      }),
    ).toBe(false)
  })

  it('hides onboarding when a running agent exists', () => {
    expect(
      shouldShowProjectOnboarding({
        dismissed: false,
        loading: false,
        stats: { runningAgents: 1, totalTickets: 0 },
        projectId: 'project-1',
        orgId: 'org-1',
      }),
    ).toBe(false)
  })
})
