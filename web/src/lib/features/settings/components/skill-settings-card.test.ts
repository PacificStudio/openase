import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Skill, Workflow } from '$lib/api/contracts'
import SkillSettingsCard from './skill-settings-card.svelte'

const {
  bindSkill,
  deleteSkill,
  disableSkill,
  enableSkill,
  getSkill,
  listSkillHistory,
  unbindSkill,
  updateSkill,
} = vi.hoisted(() => ({
  bindSkill: vi.fn(),
  deleteSkill: vi.fn(),
  disableSkill: vi.fn(),
  enableSkill: vi.fn(),
  getSkill: vi.fn(),
  listSkillHistory: vi.fn(),
  unbindSkill: vi.fn(),
  updateSkill: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  bindSkill,
  deleteSkill,
  disableSkill,
  enableSkill,
  getSkill,
  listSkillHistory,
  unbindSkill,
  updateSkill,
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

const workflows: Workflow[] = [
  {
    id: 'wf-1',
    project_id: 'project-1',
    agent_id: null,
    name: 'Coding Workflow',
    type: 'coding',
    harness_path: '.openase/harnesses/coding.md',
    hooks: {},
    max_concurrent: 1,
    max_retry_attempts: 1,
    timeout_minutes: 30,
    stall_timeout_minutes: 5,
    version: 3,
    is_active: true,
    pickup_status_ids: ['todo'],
    finish_status_ids: ['done'],
  } as Workflow,
]

const skill: Skill = {
  id: 'skill-1',
  name: 'commit',
  description: 'Create a well-formed git commit.',
  path: '/skills/commit',
  current_version: 2,
  is_builtin: true,
  is_enabled: true,
  created_by: 'system:init',
  created_at: '2026-03-28T12:00:00Z',
  bound_workflows: [
    { id: 'wf-1', name: 'Coding Workflow', harness_path: '.openase/harnesses/coding.md' },
  ],
} as Skill

describe('SkillSettingsCard', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('shows the current published version on the collapsed card', () => {
    const { getByText } = render(SkillSettingsCard, {
      props: {
        skill,
        workflows,
        onChanged: vi.fn(),
      },
    })

    expect(getByText('Published v2')).toBeTruthy()
  })

  it('loads and renders skill version history when editing', async () => {
    getSkill.mockResolvedValue({
      skill,
      content: '# Commit\n\nWrite a conventional commit message.',
    })
    listSkillHistory.mockResolvedValue({
      history: [
        { id: 'v2', version: 2, created_by: 'user:gary', created_at: '2026-03-28T12:00:00Z' },
        { id: 'v1', version: 1, created_by: 'system:init', created_at: '2026-03-27T12:00:00Z' },
      ],
    })

    const { getByText, getByRole, findByText } = render(SkillSettingsCard, {
      props: {
        skill,
        workflows,
        onChanged: vi.fn(),
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Actions for commit' }))
    await fireEvent.click(getByText('Edit'))

    expect(await findByText('Published versions')).toBeTruthy()
    expect(await findByText('v2')).toBeTruthy()
    expect(await findByText('user:gary')).toBeTruthy()

    await waitFor(() => {
      expect(getSkill).toHaveBeenCalledWith('skill-1')
      expect(listSkillHistory).toHaveBeenCalledWith('skill-1')
    })
  })
})
