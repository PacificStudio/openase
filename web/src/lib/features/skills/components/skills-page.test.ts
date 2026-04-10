import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Skill } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import SkillsPage from './skills-page.svelte'

const { createSkill, listSkills, goto } = vi.hoisted(() => ({
  createSkill: vi.fn(),
  listSkills: vi.fn(),
  goto: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createSkill,
  listSkills,
}))

vi.mock('$app/navigation', () => ({
  goto,
}))

function buildSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'skill-1',
    name: 'deploy-openase',
    description: 'Build and redeploy OpenASE locally.',
    path: '.openase/skills/deploy-openase/SKILL.md',
    current_version: 1,
    is_builtin: false,
    is_enabled: true,
    created_by: 'user:manual',
    created_at: '2026-04-01T12:00:00Z',
    bound_workflows: [],
    ...overrides,
  }
}

describe('SkillsPage', () => {
  beforeEach(() => {
    appStore.currentOrg = {
      id: 'org-1',
      name: 'OpenAI',
      slug: 'openai',
      status: 'active',
      default_agent_provider_id: null,
    }
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'OpenASE',
      slug: 'openase',
      description: '',
      status: 'active',
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }
  })

  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('wraps the skills list in a route scroll container so long lists stay reachable', async () => {
    listSkills.mockResolvedValue({
      skills: [
        buildSkill({ id: 'skill-1', name: 'deploy-openase', bound_workflows: [] }),
        buildSkill({
          id: 'skill-2',
          name: 'commit',
          is_builtin: true,
          bound_workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
        }),
      ],
    })

    const view = render(SkillsPage)
    const scrollContainer = view.getByTestId('route-scroll-container')

    expect(scrollContainer.className).toContain('min-h-0')
    expect(scrollContainer.className).toContain('flex-1')
    expect(scrollContainer.className).toContain('overflow-y-auto')

    expect(await view.findByText('deploy-openase')).toBeTruthy()
    expect(view.getByText('2 skills')).toBeTruthy()
    expect(view.getByText('2 enabled')).toBeTruthy()
    expect(view.getByText('1 bound')).toBeTruthy()
    expect(listSkills).toHaveBeenCalledWith('project-1')
  })

  it('still renders the empty state inside the scroll container', async () => {
    listSkills.mockResolvedValue({ skills: [] })

    const view = render(SkillsPage)

    expect(view.getByTestId('route-scroll-container').className).toContain('overflow-y-auto')
    expect(await view.findByText('No skills yet')).toBeTruthy()
    expect(view.getByText('0 skills')).toBeTruthy()
  })
})
