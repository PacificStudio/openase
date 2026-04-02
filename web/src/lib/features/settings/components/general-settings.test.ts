import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

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

  it('saves the agent run summary prompt with other project settings', async () => {
    appStore.currentProject = currentProject()
    updateProject.mockResolvedValue({
      project: {
        ...currentProject(),
        name: 'OpenASE Runtime',
        agent_run_summary_prompt: 'Custom summary prompt',
      },
    })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    await fireEvent.input(getByLabelText('Project name'), {
      target: { value: 'OpenASE Runtime' },
    })
    await fireEvent.input(getByLabelText('Run summary prompt'), {
      target: { value: 'Custom summary prompt' },
    })
    await fireEvent.click(getByRole('button', { name: 'Save changes' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith(currentProject().id, {
        name: 'OpenASE Runtime',
        description: '',
        max_concurrent_agents: 4,
        agent_run_summary_prompt: 'Custom summary prompt',
      }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Project settings saved.')
  })

  it('sends blank summary prompts as default-fallback semantics', async () => {
    appStore.currentProject = currentProject()
    updateProject.mockResolvedValue({ project: currentProject() })

    const { getByLabelText, getByRole } = render(GeneralSettings)

    await fireEvent.input(getByLabelText('Run summary prompt'), {
      target: { value: '   ' },
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

function currentProject() {
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
