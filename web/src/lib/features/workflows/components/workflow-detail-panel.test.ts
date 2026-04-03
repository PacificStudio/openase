import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import WorkflowDetailPanel from './workflow-detail-panel.svelte'

vi.mock('$lib/features/providers', () => ({
  adapterIconPath: vi.fn(() => ''),
  availabilityDotColor: vi.fn(() => 'bg-emerald-500'),
}))

const workflow = {
  id: 'wf-1',
  name: 'Coding Workflow',
  type: 'coding' as const,
  workflowFamily: 'coding' as const,
  classification: {
    family: 'coding' as const,
    confidence: 1,
    reasons: ['fixture'],
  },
  agentId: 'agent-1',
  harnessPath: '.openase/harnesses/coding.md',
  pickupStatusIds: ['todo'],
  pickupStatusLabel: 'To Do',
  finishStatusIds: ['done'],
  finishStatusLabel: 'Done',
  maxConcurrent: 1,
  maxRetry: 1,
  timeoutMinutes: 30,
  stallTimeoutMinutes: 10,
  isActive: true,
  lastModified: '2026-03-28T12:00:00Z',
  recentSuccessRate: 85,
  version: 3,
  history: [],
}

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

describe('WorkflowDetailPanel', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('allows saving pickup and finish bindings across any status stage', async () => {
    const onSave = vi.fn()
    const { getAllByRole, getByRole } = render(WorkflowDetailPanel, {
      props: {
        workflow,
        statuses,
        agentOptions,
        onSave,
      },
    })

    expect(getAllByRole('button', { name: 'Done' })).toHaveLength(2)
    expect(getAllByRole('button', { name: 'Doing' })).toHaveLength(2)

    await fireEvent.click(getAllByRole('button', { name: 'Done' })[0])
    await fireEvent.click(getAllByRole('button', { name: 'Doing' })[1])
    await fireEvent.click(getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith({
        agent_id: 'agent-1',
        finish_status_ids: ['done', 'doing'],
        hooks: undefined,
        is_active: true,
        max_concurrent: 1,
        max_retry_attempts: 1,
        name: 'Coding Workflow',
        type: 'coding',
        pickup_status_ids: ['todo', 'done'],
        stall_timeout_minutes: 10,
        timeout_minutes: 30,
      })
    })
  })
})
