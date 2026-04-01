import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import WorkflowCreationDialog from './workflow-creation-dialog.svelte'

const { createWorkflowWithBinding } = vi.hoisted(() => ({
  createWorkflowWithBinding: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock('../data', () => ({
  createWorkflowWithBinding,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

const statuses = [
  { id: 'backlog', name: 'Backlog', stage: 'backlog' as const },
  { id: 'todo', name: 'To Do', stage: 'unstarted' as const },
  { id: 'doing', name: 'Doing', stage: 'started' as const },
  { id: 'done', name: 'Done', stage: 'completed' as const },
  { id: 'canceled', name: 'Canceled', stage: 'canceled' as const },
]

const agentOptions = [
  {
    id: 'agent-1',
    label: 'Primary Agent',
    agentName: 'Primary Agent',
    providerName: 'OpenAI',
    modelName: 'gpt-5.4',
    machineName: 'Local',
    adapterType: 'openai',
    available: true,
  },
]

describe('WorkflowCreationDialog', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('allows selecting any status stage for pickup and finish', async () => {
    createWorkflowWithBinding.mockResolvedValue({
      workflow: {
        id: 'wf-1',
        name: 'Workflow 1',
        type: 'coding',
        agentId: 'agent-1',
        harnessPath: '',
        pickupStatusIds: ['backlog', 'done'],
        pickupStatusLabel: 'Backlog, Done',
        finishStatusIds: ['backlog', 'doing'],
        finishStatusLabel: 'Backlog, Doing',
        maxConcurrent: 1,
        maxRetry: 1,
        timeoutMinutes: 30,
        stallTimeoutMinutes: 10,
        isActive: true,
        lastModified: '2026-04-01T10:00:00Z',
        recentSuccessRate: 0,
        version: 1,
        history: [],
      },
      selectedId: 'wf-1',
    })

    const onCreated = vi.fn()
    const { getAllByRole, getByRole } = render(WorkflowCreationDialog, {
      props: {
        open: true,
        projectId: 'project-1',
        statuses,
        agentOptions,
        existingCount: 0,
        builtinRoleContent: 'role',
        onCreated,
      },
    })

    expect(getAllByRole('button', { name: 'Done' })).toHaveLength(2)
    expect(getAllByRole('button', { name: 'Doing' })).toHaveLength(2)

    await fireEvent.click(getAllByRole('button', { name: 'Done' })[0])
    await fireEvent.click(getAllByRole('button', { name: 'Doing' })[1])
    await fireEvent.click(getByRole('button', { name: 'Create workflow' }))

    await waitFor(() => {
      expect(createWorkflowWithBinding).toHaveBeenCalledWith(
        'project-1',
        expect.objectContaining({
          agentId: 'agent-1',
          name: 'Workflow 1',
          pickupStatusIds: ['backlog', 'done'],
          finishStatusIds: ['backlog', 'doing'],
        }),
        statuses,
        'role',
      )
    })
    expect(onCreated).toHaveBeenCalledTimes(1)
  })
})
