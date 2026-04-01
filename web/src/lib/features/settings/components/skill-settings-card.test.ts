import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Skill } from '$lib/api/contracts'
import SkillSettingsCard from './skill-settings-card.svelte'

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

  it('shows the current published version on the card', () => {
    const { getByText } = render(SkillSettingsCard, {
      props: {
        skill,
        onSelect: vi.fn(),
      },
    })

    expect(getByText('v2')).toBeTruthy()
  })

  it('shows bound workflow names', () => {
    const { getByText } = render(SkillSettingsCard, {
      props: {
        skill,
        onSelect: vi.fn(),
      },
    })

    expect(getByText('Coding Workflow')).toBeTruthy()
  })

  it('calls onSelect when clicked', async () => {
    const onSelect = vi.fn()
    const { getByText } = render(SkillSettingsCard, {
      props: {
        skill,
        onSelect,
      },
    })

    await getByText('commit').click()
    expect(onSelect).toHaveBeenCalledWith(skill)
  })
})
