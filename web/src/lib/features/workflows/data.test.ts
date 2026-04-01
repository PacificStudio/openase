import { beforeEach, describe, expect, it, vi } from 'vitest'

import { createWorkflowWithBinding } from './data'

const { createWorkflow } = vi.hoisted(() => ({
  createWorkflow: vi.fn(),
}))

vi.mock('$lib/api/openase', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/openase')>('$lib/api/openase')
  return {
    ...actual,
    createWorkflow,
  }
})

describe('createWorkflowWithBinding', () => {
  const statuses = [
    { id: 'todo', name: 'To Do', stage: 'unstarted' as const },
    { id: 'done', name: 'Done', stage: 'completed' as const },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('forwards parsed hooks to workflow creation payloads', async () => {
    createWorkflow.mockResolvedValue({
      workflow: {
        id: 'wf-1',
        name: 'Workflow 1',
        type: 'coding',
        agent_id: 'agent-1',
        harness_path: '.openase/harnesses/coding.md',
        pickup_status_ids: ['todo'],
        finish_status_ids: ['done'],
        max_concurrent: 0,
        max_retry_attempts: 1,
        timeout_minutes: 30,
        stall_timeout_minutes: 5,
        is_active: true,
        version: 1,
        hooks: {
          workflow_hooks: {
            on_activate: [{ cmd: 'claude --version', timeout: 30, on_failure: 'block' }],
          },
        },
      },
    })

    await createWorkflowWithBinding(
      'project-1',
      {
        agentId: 'agent-1',
        name: 'Workflow 1',
        workflowType: 'coding',
        pickupStatusIds: ['todo'],
        finishStatusIds: ['done'],
        hooks: {
          workflow_hooks: {
            on_activate: [{ cmd: 'claude --version', timeout: 30, on_failure: 'block' }],
          },
        },
      },
      statuses,
      'role',
    )

    expect(createWorkflow).toHaveBeenCalledWith(
      'project-1',
      expect.objectContaining({
        hooks: {
          workflow_hooks: {
            on_activate: [{ cmd: 'claude --version', timeout: 30, on_failure: 'block' }],
          },
        },
      }),
    )
  })
})
