import { describe, expect, it } from 'vitest'

import { createWorkflowLifecycleDraft, parseWorkflowLifecycleDraft } from './workflow-lifecycle'

describe('workflow lifecycle draft', () => {
  it('renders unlimited max concurrent as a blank field and parses blank back to zero', () => {
    const draft = createWorkflowLifecycleDraft({
      id: 'wf-1',
      name: 'Coding Workflow',
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
      pickupStatusLabel: 'Todo',
      finishStatusIds: ['done'],
      finishStatusLabel: 'Done',
      maxConcurrent: 0,
      maxRetry: 0,
      timeoutMinutes: 30,
      stallTimeoutMinutes: 5,
      isActive: true,
      lastModified: '2026-04-01T10:00:00Z',
      recentSuccessRate: 100,
      version: 1,
      history: [],
    })

    expect(draft.maxConcurrent).toBe('')
    expect(draft.maxRetryAttempts).toBe('0')

    const parsed = parseWorkflowLifecycleDraft(draft)
    expect(parsed).toEqual({
      ok: true,
      value: {
        agent_id: 'agent-1',
        finish_status_ids: ['done'],
        is_active: true,
        max_concurrent: 0,
        max_retry_attempts: 0,
        name: 'Coding Workflow',
        platform_access_allowed: [],
        type: 'coding',
        pickup_status_ids: ['todo'],
        role_description: '',
        role_name: 'Coding Workflow',
        stall_timeout_minutes: 5,
        timeout_minutes: 30,
      },
    })
  })

  it('rejects non-positive explicit max concurrent values', () => {
    const parsed = parseWorkflowLifecycleDraft({
      agentId: 'agent-1',
      name: 'Coding Workflow',
      typeLabel: 'coding',
      pickupStatusIds: ['todo'],
      finishStatusIds: ['done'],
      maxConcurrent: '0',
      maxRetryAttempts: '0',
      timeoutMinutes: '30',
      stallTimeoutMinutes: '5',
      isActive: true,
    })

    expect(parsed).toEqual({
      ok: false,
      error: 'Max concurrent must be a positive integer.',
    })
  })

  it('rejects overlapping pickup and finish statuses', () => {
    const parsed = parseWorkflowLifecycleDraft({
      agentId: 'agent-1',
      name: 'Coding Workflow',
      typeLabel: 'coding',
      pickupStatusIds: ['todo', 'doing'],
      finishStatusIds: ['doing', 'done'],
      maxConcurrent: '1',
      maxRetryAttempts: '0',
      timeoutMinutes: '30',
      stallTimeoutMinutes: '5',
      isActive: true,
    })

    expect(parsed).toEqual({
      ok: false,
      error: 'Pickup and finish statuses must be mutually exclusive.',
    })
  })
})
