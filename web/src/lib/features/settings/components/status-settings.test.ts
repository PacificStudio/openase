import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import StatusSettings from './status-settings.svelte'

const {
  createStage,
  createStatus,
  deleteStage,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStage,
  updateStatus,
} = vi.hoisted(() => ({
  createStage: vi.fn(),
  createStatus: vi.fn(),
  deleteStage: vi.fn(),
  deleteStatus: vi.fn(),
  listStatuses: vi.fn(),
  resetStatuses: vi.fn(),
  updateStage: vi.fn(),
  updateStatus: vi.fn(),
}))

const { connectEventStream } = vi.hoisted(() => ({
  connectEventStream: vi.fn(() => () => {}),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  createStage,
  createStatus,
  deleteStage,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStage,
  updateStatus,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

function buildPayload() {
  return {
    stages: [
      {
        id: 'stage-1',
        project_id: 'project-1',
        key: 'backlog',
        name: 'Backlog',
        position: 0,
        active_runs: 0,
        max_active_runs: null,
        description: '',
      },
      {
        id: 'stage-2',
        project_id: 'project-1',
        key: 'in-progress',
        name: 'In Progress',
        position: 1,
        active_runs: 1,
        max_active_runs: 1,
        description: '',
      },
    ],
    stage_groups: [],
    statuses: [
      {
        id: 'status-1',
        project_id: 'project-1',
        stage_id: 'stage-1',
        stage: null,
        name: 'Todo',
        color: '#94a3b8',
        icon: '',
        position: 0,
        is_default: true,
        description: '',
      },
      {
        id: 'status-2',
        project_id: 'project-1',
        stage_id: 'stage-2',
        stage: null,
        name: 'Doing',
        color: '#fbbf24',
        icon: '',
        position: 1,
        is_default: false,
        description: '',
      },
    ],
  }
}

function seedProject() {
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
}

describe('Status settings', () => {
  beforeEach(() => {
    seedProject()
    listStatuses.mockResolvedValue(buildPayload())
  })

  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the status editor with stage management instead of the old runtime-only card', async () => {
    const { findByText, queryByText } = render(StatusSettings)

    expect(await findByText('Stages')).toBeTruthy()
    expect(await findByText('Backlog')).toBeTruthy()
    expect(await findByText('0 active now, unlimited capacity')).toBeTruthy()
    expect(queryByText('Stage Concurrency')).toBeNull()
  })

  it('creates a stage from the management panel', async () => {
    createStage.mockResolvedValue({
      stage: {
        id: 'stage-3',
        project_id: 'project-1',
        key: 'review',
        name: 'Review',
        position: 2,
        active_runs: 0,
        max_active_runs: null,
        description: '',
      },
    })

    const { findByPlaceholderText, getByRole } = render(StatusSettings)

    await fireEvent.input(await findByPlaceholderText('New stage name'), {
      target: { value: 'Review' },
    })
    await fireEvent.click(getByRole('button', { name: 'Add stage' }))

    await waitFor(() =>
      expect(createStage).toHaveBeenCalledWith('project-1', {
        key: 'review',
        name: 'Review',
        max_active_runs: null,
      }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Created stage "Review".')
  })

  it('updates stage concurrency from the management row', async () => {
    updateStage.mockResolvedValue({
      stage: {
        id: 'stage-2',
        project_id: 'project-1',
        key: 'in-progress',
        name: 'In Progress',
        position: 1,
        active_runs: 1,
        max_active_runs: 2,
        description: '',
      },
    })

    const { findByDisplayValue } = render(StatusSettings)

    const capacityInput = await findByDisplayValue('1')
    const stageRow = capacityInput.closest('.border-border.rounded-md.border.px-3.py-3')
    expect(stageRow).toBeTruthy()
    await fireEvent.input(capacityInput, { target: { value: '2' } })
    await fireEvent.click(within(stageRow as HTMLElement).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updateStage).toHaveBeenCalledWith('stage-2', { max_active_runs: 2 }))
    expect(toastStore.success).toHaveBeenCalledWith('Updated stage "In Progress".')
  })

  it('deletes a stage and confirms detachment semantics', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    deleteStage.mockResolvedValue({
      deleted_stage_id: 'stage-2',
      detached_statuses: 1,
    })

    const { findByDisplayValue } = render(StatusSettings)

    const capacityInput = await findByDisplayValue('1')
    const stageRow = capacityInput.closest('.border-border.rounded-md.border.px-3.py-3')
    expect(stageRow).toBeTruthy()
    await fireEvent.click(within(stageRow as HTMLElement).getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(deleteStage).toHaveBeenCalledWith('stage-2'))
    expect(confirmSpy).toHaveBeenCalledWith(
      'Delete "In Progress"? Statuses in this stage will become ungrouped until you reassign them.',
    )

    confirmSpy.mockRestore()
  })

  it('loads statuses once when the project settings page mounts', async () => {
    const { findByText } = render(StatusSettings)

    expect(await findByText('Stages')).toBeTruthy()
    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(connectEventStream).toHaveBeenCalledTimes(1)
  })
})
