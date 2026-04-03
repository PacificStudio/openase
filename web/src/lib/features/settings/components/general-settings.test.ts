import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import GeneralSettings from './general-settings.svelte'

const { updateProject } = vi.hoisted(() => ({
  updateProject: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  archiveProject: vi.fn(),
  updateProject,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('General settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('displays the effective built-in prompt and preserves default-fallback semantics when unchanged', async () => {
    const effectivePrompt = 'Built-in run summary prompt currently in effect.'
    appStore.currentProject = currentProject({
      effective_agent_run_summary_prompt: effectivePrompt,
      agent_run_summary_prompt_source: 'builtin',
    })
    updateProject.mockResolvedValue({
      project: currentProject({
        effective_agent_run_summary_prompt: effectivePrompt,
        agent_run_summary_prompt_source: 'builtin',
      }),
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    expect((getByLabelText('Run summary prompt') as HTMLTextAreaElement).value).toBe(
      effectivePrompt,
    )
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: '',
      }),
    )
  })

  it('turns the built-in effective prompt into a project override once edited', async () => {
    const effectivePrompt = 'Built-in run summary prompt currently in effect.'
    const expectedPrompt =
      `${effectivePrompt}\n\n## Files Touched\n` +
      'Summarize the most important files or directories that were changed.'
    appStore.currentProject = currentProject({
      effective_agent_run_summary_prompt: effectivePrompt,
      agent_run_summary_prompt_source: 'builtin',
    })
    updateProject.mockResolvedValue({
      project: currentProject({
        agent_run_summary_prompt: expectedPrompt,
        effective_agent_run_summary_prompt: expectedPrompt,
        agent_run_summary_prompt_source: 'project_override',
      }),
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    expect((getByLabelText('Run summary prompt') as HTMLTextAreaElement).value).toBe(
      effectivePrompt,
    )
    await fireEvent.click(getByRole('button', { name: 'Files Touched' }))
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: expectedPrompt,
      }),
    )
  })

  it('saves blank summary prompts to reset an existing project override', async () => {
    const builtinPrompt = 'Built-in run summary prompt currently in effect.'
    appStore.currentProject = currentProject({
      agent_run_summary_prompt: 'Existing custom summary prompt',
      effective_agent_run_summary_prompt: 'Existing custom summary prompt',
      agent_run_summary_prompt_source: 'project_override',
    })
    updateProject.mockResolvedValue({
      project: currentProject({
        agent_run_summary_prompt: '',
        effective_agent_run_summary_prompt: builtinPrompt,
        agent_run_summary_prompt_source: 'builtin',
      }),
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    expect((getByLabelText('Run summary prompt') as HTMLTextAreaElement).value).toBe(
      'Existing custom summary prompt',
    )
    await fireEvent.input(getByLabelText('Run summary prompt'), {
      target: { value: '' },
    })
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: '',
      }),
    )
  })
})

function currentProject(overrides: Partial<Project> = {}): Project {
  return {
    ...currentProjectBase(),
    ...overrides,
  }
}

function currentProjectBase(): Project {
  return {
    id: 'project-1',
    organization_id: 'org-1',
    name: 'OpenASE',
    slug: 'openase',
    description: '',
    status: 'active',
    default_agent_provider_id: null,
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
    agent_run_summary_prompt: '',
    effective_agent_run_summary_prompt: 'Built-in run summary prompt currently in effect.',
    agent_run_summary_prompt_source: 'builtin' as const,
  }
}
