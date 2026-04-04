import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

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

const { subscribeProjectEvents } = vi.hoisted(() => ({
  subscribeProjectEvents: vi.fn(() => () => {}),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  createStatus,
  deleteStatus,
  listStatuses,
  resetStatuses,
  updateStatus,
}))

vi.mock('$lib/features/project-events/project-event-bus', () => ({
  subscribeProjectEvents,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

function buildPayload() {
  return {
    statuses: [
      {
        id: 'status-1',
        project_id: 'project-1',
        name: 'Todo',
        stage: 'unstarted',
        color: '#94a3b8',
        icon: '',
        position: 0,
        active_runs: 0,
        max_active_runs: null,
        is_default: true,
        description: '',
      },
      {
        id: 'status-2',
        project_id: 'project-1',
        name: 'Doing',
        stage: 'started',
        color: '#fbbf24',
        icon: '',
        position: 1,
        active_runs: 1,
        max_active_runs: 1,
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

  it('renders the status editor with stage management', async () => {
    const { findByText } = render(StatusSettings)

    expect(await findByText('Statuses')).toBeTruthy()
    expect(await findByText('Todo')).toBeTruthy()
    expect(await findByText('Doing')).toBeTruthy()
    expect(await findByText('1/1')).toBeTruthy()
    expect(await findByText('Backlog')).toBeTruthy()
    expect(await findByText('Started')).toBeTruthy()
  })

  it('creates a status from the management panel', async () => {
    createStatus.mockResolvedValue({
      status: {
        id: 'status-3',
        project_id: 'project-1',
        name: 'Review',
        stage: 'backlog',
        color: '#6366f1',
        icon: '',
        position: 2,
        active_runs: 0,
        max_active_runs: 2,
        is_default: false,
        description: '',
      },
    })

    const { findByLabelText, findByPlaceholderText } = render(StatusSettings)

    const backlogGroup = await findByLabelText('Backlog stage')
    await fireEvent.click(within(backlogGroup).getByRole('button', { name: 'Add' }))
    await fireEvent.input(await findByPlaceholderText('Status name'), {
      target: { value: 'Review' },
    })
    const createCapacityInput = await findByPlaceholderText('Unlimited')
    await fireEvent.input(createCapacityInput, {
      target: { value: '2' },
    })
    await fireEvent.click(within(backlogGroup).getAllByRole('button', { name: 'Add' }).at(-1)!)

    await waitFor(() =>
      expect(createStatus).toHaveBeenCalledWith('project-1', {
        name: 'Review',
        stage: 'backlog',
        color: '#94a3b8',
        is_default: false,
        max_active_runs: 2,
      }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Created status "Review".')
  })

  it('keeps the status draft when creation fails', async () => {
    createStatus.mockRejectedValue(new Error('conflict'))

    const { findByLabelText, findByPlaceholderText } = render(StatusSettings)

    const backlogGroup = await findByLabelText('Backlog stage')
    await fireEvent.click(within(backlogGroup).getByRole('button', { name: 'Add' }))
    const nameInput = await findByPlaceholderText('Status name')
    await fireEvent.input(nameInput, {
      target: { value: 'Review' },
    })
    await fireEvent.click(within(backlogGroup).getAllByRole('button', { name: 'Add' }).at(-1)!)

    await waitFor(() => expect(createStatus).toHaveBeenCalledTimes(1))
    expect((nameInput as HTMLInputElement).value).toBe('Review')
    expect(toastStore.error).toHaveBeenCalledWith('Failed to create status.')
  })

  it('updates status concurrency from the management row', async () => {
    updateStatus.mockResolvedValue({
      status: {
        ...buildPayload().statuses[1],
        max_active_runs: 2,
      },
    })

    const { findAllByRole, findByDisplayValue, findByRole } = render(StatusSettings)

    await fireEvent.click((await findAllByRole('button', { name: 'Actions' }))[1])
    await fireEvent.click(await findByRole('menuitem', { name: 'Edit' }))

    const capacityInput = await findByDisplayValue('1')
    await fireEvent.input(capacityInput, { target: { value: '2' } })
    await fireEvent.click(await findByRole('button', { name: 'Save' }))

    await waitFor(() =>
      expect(updateStatus).toHaveBeenCalledWith('status-2', { max_active_runs: 2 }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Updated status "Doing".')
  })

  it('deletes a status after confirmation', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    deleteStatus.mockResolvedValue({
      deleted_status_id: 'status-2',
      replacement_status_id: 'status-1',
      moved_ticket_count: 1,
    })

    const { findAllByRole, findByRole } = render(StatusSettings)

    await fireEvent.click((await findAllByRole('button', { name: 'Actions' }))[1])
    await fireEvent.click(await findByRole('menuitem', { name: 'Delete' }))

    await waitFor(() => expect(deleteStatus).toHaveBeenCalledWith('status-2'))
    expect(confirmSpy).toHaveBeenCalledWith(
      'Delete "Doing"? Tickets assigned to it will be moved to a replacement status.',
    )

    confirmSpy.mockRestore()
  })

  it('loads statuses once when the project settings page mounts', async () => {
    const { findByText } = render(StatusSettings)

    expect(await findByText('Statuses')).toBeTruthy()
    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(subscribeProjectEvents).toHaveBeenCalledTimes(1)
  })
})
