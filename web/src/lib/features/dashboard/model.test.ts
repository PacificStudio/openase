import { describe, expect, it } from 'vitest'

import type { Project } from '$lib/api/contracts'
import { buildProjectSummary } from './model'

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'TodoApp',
  slug: 'todoapp',
  description: 'Shared ticket workflow and automation dashboard.',
  status: 'Planned',
  default_workflow_id: null,
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
})
