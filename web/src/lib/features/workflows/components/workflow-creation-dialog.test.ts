import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '$lib/api/client'
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
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('blocks selecting the same status in both pickup and finish bindings', async () => {
    const { getAllByRole } = render(WorkflowCreationDialog, {
      props: {
        open: true,
        projectId: 'project-1',
        statuses,
        agentOptions,
        existingCount: 0,
        builtinRoleContent: 'role',
      },
    })

    expect(getAllByRole('button', { name: 'Doing' })).toHaveLength(2)

    await fireEvent.click(getAllByRole('button', { name: 'Doing' })[1])

    await waitFor(() => {
      expect(getAllByRole('button', { name: 'Doing' })[0].hasAttribute('disabled')).toBe(true)
    })
  })

  it('preselects pickup and finish statuses from structured template metadata', async () => {
    createWorkflowWithBinding.mockResolvedValue({
      workflow: {
        id: 'wf-2',
        name: 'Dispatcher',
        type: 'custom',
        workflowFamily: 'dispatcher',
        classification: {
          family: 'dispatcher',
          confidence: 1,
          reasons: ['fixture'],
        },
        agentId: 'agent-1',
        harnessPath: '.openase/harnesses/roles/dispatcher.md',
        pickupStatusIds: ['backlog'],
        pickupStatusLabel: 'Backlog',
        finishStatusIds: ['backlog'],
        finishStatusLabel: 'Backlog',
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
      selectedId: 'wf-2',
    })

    const { getByRole } = render(WorkflowCreationDialog, {
      props: {
        open: true,
        projectId: 'project-1',
        statuses,
        agentOptions,
        existingCount: 1,
        builtinRoleContent: 'role',
        templateDraft: {
          name: 'Dispatcher',
          content: '# Dispatcher\n\nRoute backlog tickets.\n',
          workflowType: 'Dispatcher',
          workflowFamily: 'dispatcher',
          roleSlug: 'dispatcher',
          pickupStatusNames: ['Backlog'],
          finishStatusNames: ['Todo'],
          harnessPath: '.openase/harnesses/roles/dispatcher.md',
        },
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Create workflow' }))

    await waitFor(() => {
      expect(createWorkflowWithBinding).toHaveBeenCalledWith(
        'project-1',
        expect.objectContaining({
          name: 'Dispatcher',
          workflowType: 'Dispatcher',
          harnessPath: '.openase/harnesses/roles/dispatcher.md',
          pickupStatusIds: ['backlog'],
          finishStatusIds: ['todo'],
        }),
        statuses,
        '# Dispatcher\n\nRoute backlog tickets.\n',
      )
    })
  })

  it('shows a precise error when template statuses are missing from the project', async () => {
    const { findByText, getByRole } = render(WorkflowCreationDialog, {
      props: {
        open: true,
        projectId: 'project-1',
        statuses,
        agentOptions,
        existingCount: 1,
        builtinRoleContent: 'role',
        templateDraft: {
          name: 'Dispatcher',
          content: '# Dispatcher\n\nRoute backlog tickets.\n',
          workflowType: 'Dispatcher',
          workflowFamily: 'dispatcher',
          pickupStatusNames: ['Inbox'],
          finishStatusNames: ['Inbox'],
          harnessPath: '.openase/harnesses/roles/dispatcher.md',
        },
      },
    })

    expect(
      await findByText('Template status bindings are not configured in this project: Inbox.'),
    ).toBeTruthy()
    expect(getByRole('button', { name: 'Create workflow' }).hasAttribute('disabled')).toBe(true)
    expect(createWorkflowWithBinding).not.toHaveBeenCalled()
  })

  it('shows a friendly toast when the workflow name conflicts', async () => {
    createWorkflowWithBinding.mockRejectedValue(
      new ApiError(
        409,
        'workflow name "Workflow 1" is already used in this project',
        'WORKFLOW_NAME_CONFLICT',
      ),
    )

    const { getByRole } = render(WorkflowCreationDialog, {
      props: {
        open: true,
        projectId: 'project-1',
        statuses,
        agentOptions,
        existingCount: 0,
        builtinRoleContent: 'role',
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Create workflow' }))

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith(
        'A workflow with this name already exists in the project.',
      )
    })
  })
})
