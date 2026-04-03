import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  buildRunSummaryPrompt,
  defaultRunSummarySectionKeys,
} from '$lib/features/settings/run-summary-prompt-template'
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

  it('saves the generated agent run summary prompt with other project settings', async () => {
    appStore.currentProject = currentProject()
    const expectedPrompt = buildRunSummaryPrompt(
      [...defaultRunSummarySectionKeys, 'files_touched'],
      'Focus on verification gaps.',
    )
    updateProject.mockResolvedValue({
      project: {
        ...currentProject(),
        name: 'OpenASE Runtime',
        agent_run_summary_prompt: expectedPrompt,
      },
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    await fireEvent.input(getByLabelText('Project name'), {
      target: { value: 'OpenASE Runtime' },
    })
    await fireEvent.click(getByRole('checkbox', { name: 'Customize run summary prompt' }))
    await fireEvent.click(getByRole('checkbox', { name: /^Files Touched/ }))
    await fireEvent.input(getByLabelText('Additional instructions'), {
      target: { value: 'Focus on verification gaps.' },
    })
    await fireEvent.click(getByRole('button', { name: 'Apply selected sections' }))
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE Runtime',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: expectedPrompt,
      }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Project settings saved.')
  })

  it('sends blank summary prompts as default-fallback semantics when customization is disabled', async () => {
    appStore.currentProject = currentProject()
    updateProject.mockResolvedValue({ project: currentProject() })

    const { getByRole } = render(GeneralSettings)
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

  it('saves manually edited custom prompts verbatim', async () => {
    appStore.currentProject = currentProject({
      agent_run_summary_prompt: 'Existing custom summary prompt',
    })
    updateProject.mockResolvedValue({
      project: currentProject({
        agent_run_summary_prompt: 'Edited custom summary prompt',
      }),
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    await fireEvent.input(getByLabelText('Final run summary prompt'), {
      target: { value: 'Edited custom summary prompt' },
    })
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: 'Edited custom summary prompt',
      }),
    )
  })
})

function currentProject(overrides: Partial<ReturnType<typeof currentProjectBase>> = {}) {
  return {
    ...currentProjectBase(),
    ...overrides,
  }
}

function currentProjectBase() {
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
  }
}
