import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import StatusSettings from './status-settings.svelte'

const {
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
  connectEventStream,
} = vi.hoisted(() => ({
  createStatus: vi.fn(),
  deleteStatus: vi.fn(),
  listStatuses: vi.fn(),
  resetStatuses: vi.fn(),
  updateStatus: vi.fn(),
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

describe('Status settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders shared stage occupancy and capacity from the statuses payload', async () => {
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

    connectEventStream.mockReturnValue(() => {})
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

    const { findByText } = render(StatusSettings)

    expect(await findByText('Stage Concurrency')).toBeTruthy()
    expect(await findByText('In Progress')).toBeTruthy()
    expect(await findByText('1 active now, capacity 1')).toBeTruthy()
    expect(await findByText('0 active now, unlimited capacity')).toBeTruthy()
  })

  it('refreshes stage occupancy when the ticket stream emits runtime changes', async () => {
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

    let onEvent: (() => void) | undefined
    connectEventStream.mockImplementation((_url, options) => {
      onEvent = () => options.onEvent({ event: 'ticket.updated', data: '{}' })
      return () => {}
    })
    listStatuses
      .mockResolvedValueOnce({
        stages: [
          {
            id: 'stage-1',
            project_id: 'project-1',
            key: 'backlog',
            name: 'Backlog',
            position: 1,
            active_runs: 1,
            max_active_runs: null,
            description: '',
          },
        ],
        stage_groups: [],
        statuses: [],
      })
      .mockResolvedValueOnce({
        stages: [
          {
            id: 'stage-1',
            project_id: 'project-1',
            key: 'backlog',
            name: 'Backlog',
            position: 1,
            active_runs: 2,
            max_active_runs: null,
            description: '',
          },
        ],
        stage_groups: [],
        statuses: [],
      })

    const { findByText } = render(StatusSettings)

    expect(await findByText('1 active now, unlimited capacity')).toBeTruthy()
    expect(onEvent).toBeTypeOf('function')

    onEvent?.()

    expect(await findByText('2 active now, unlimited capacity')).toBeTruthy()
    await waitFor(() => {
      expect(listStatuses).toHaveBeenCalledTimes(2)
    })
  })
})
