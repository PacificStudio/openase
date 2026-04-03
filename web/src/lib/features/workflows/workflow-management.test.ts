import { beforeEach, describe, expect, it, vi } from 'vitest'

import { saveWorkflowLifecycle } from './workflow-management'

const { updateWorkflow } = vi.hoisted(() => ({
  updateWorkflow: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  updateWorkflow,
  deleteWorkflow: vi.fn(),
}))

describe('saveWorkflowLifecycle', () => {
  const statuses = [
    { id: 'todo', name: 'To Do', stage: 'unstarted' as const },
    { id: 'done', name: 'Done', stage: 'completed' as const },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('merges supported hook edits with unsupported existing keys before update', async () => {
    updateWorkflow.mockResolvedValue({
      workflow: {
        id: 'wf-1',
        name: 'Workflow 1',
        type: 'coding',
        workflow_family: 'coding',
        workflow_classification: {
          family: 'coding',
          confidence: 1,
          reasons: ['fixture'],
        },
        agent_id: 'agent-1',
        harness_path: '.openase/harnesses/coding.md',
        pickup_status_ids: ['todo'],
        finish_status_ids: ['done'],
        max_concurrent: 0,
        max_retry_attempts: 1,
        timeout_minutes: 30,
        stall_timeout_minutes: 5,
        is_active: true,
        version: 2,
        hooks: {
          workflow_hooks: {
            on_activate: [{ cmd: 'claude --version', on_failure: 'block' }],
            on_deactivate: [{ cmd: 'echo old', on_failure: 'ignore' }],
          },
        },
      },
    })

    await saveWorkflowLifecycle(
      'wf-1',
      {
        agent_id: 'agent-1',
        finish_status_ids: ['done'],
        hooks: {
          workflow_hooks: {
            on_activate: [{ cmd: 'claude --version', on_failure: 'block' }],
          },
        },
        is_active: true,
        max_concurrent: 0,
        max_retry_attempts: 1,
        name: 'Workflow 1',
        type: 'coding',
        pickup_status_ids: ['todo'],
        stall_timeout_minutes: 5,
        timeout_minutes: 30,
      },
      statuses,
      {
        id: 'wf-1',
        name: 'Workflow 1',
        type: 'coding',
        workflowFamily: 'coding',
        classification: {
          family: 'coding',
          confidence: 1,
          reasons: ['fixture'],
        },
        agentId: 'agent-1',
        harnessPath: '.openase/harnesses/coding.md',
        pickupStatusIds: ['todo'],
        pickupStatusLabel: 'To Do',
        finishStatusIds: ['done'],
        finishStatusLabel: 'Done',
        maxConcurrent: 0,
        maxRetry: 1,
        timeoutMinutes: 30,
        stallTimeoutMinutes: 5,
        isActive: true,
        lastModified: '2026-04-01T10:00:00Z',
        recentSuccessRate: 0,
        version: 1,
        history: [],
        rawHooks: {
          workflow_hooks: {
            on_deactivate: [{ cmd: 'echo old', on_failure: 'ignore' }],
          },
        },
      },
    )

    expect(updateWorkflow).toHaveBeenCalledWith('wf-1', {
      agent_id: 'agent-1',
      finish_status_ids: ['done'],
      hooks: {
        workflow_hooks: {
          on_deactivate: [{ cmd: 'echo old', on_failure: 'ignore' }],
          on_activate: [{ cmd: 'claude --version', on_failure: 'block' }],
        },
      },
      is_active: true,
      max_concurrent: 0,
      max_retry_attempts: 1,
      name: 'Workflow 1',
      type: 'coding',
      pickup_status_ids: ['todo'],
      stall_timeout_minutes: 5,
      timeout_minutes: 30,
    })
  })
})
