import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import StatusSettings from './status-settings.svelte'

const { createStatus, deleteStatus, listStatuses, resetStatuses, updateStatus } = vi.hoisted(
  () => ({
    createStatus: vi.fn(),
    deleteStatus: vi.fn(),
    listStatuses: vi.fn(),
    resetStatuses: vi.fn(),
    updateStatus: vi.fn(),
  }),
)

vi.mock('$lib/api/openase', () => ({
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
}))

describe('Status settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the status editor without embedding stage runtime cards', async () => {
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'OpenASE',
      slug: 'openase',
      description: '',
      status: 'active',
      default_workflow_id: null,
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }

    listStatuses.mockResolvedValue({
      stages: [
        {
          id: 'stage-1',
          project_id: 'project-1',
          key: 'backlog',
          name: 'Backlog',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
          description: '',
        },
        {
          id: 'stage-2',
          project_id: 'project-1',
          key: 'in_progress',
          name: 'In Progress',
          position: 2,
          active_runs: 1,
          max_active_runs: 1,
          description: 'Shared coding capacity',
        },
      ],
      stage_groups: [],
      statuses: [],
    })

    const { findByText, queryByText } = render(StatusSettings)

    expect(await findByText('Statuses')).toBeTruthy()
    expect(
      await findByText(
        'No statuses yet. Add one above or use reset to seed the default workflow template.',
      ),
    ).toBeTruthy()
    expect(queryByText('Stage Concurrency')).toBeNull()
    expect(queryByText('In Progress')).toBeNull()
  })

  it('loads statuses once when the project settings page mounts', async () => {
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'OpenASE',
      slug: 'openase',
      description: '',
      status: 'active',
      default_workflow_id: null,
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }

    listStatuses.mockResolvedValue({
      stages: [],
      stage_groups: [],
      statuses: [],
    })

    const { findByText } = render(StatusSettings)

    expect(await findByText('Statuses')).toBeTruthy()
    expect(listStatuses).toHaveBeenCalledTimes(1)
  })
})
