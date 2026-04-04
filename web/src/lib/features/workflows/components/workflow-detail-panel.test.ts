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

  it('blocks selecting the same status in both pickup and finish bindings', async () => {
    const { getAllByRole } = render(WorkflowDetailPanel, {
      props: {
        workflow,
        statuses,
        agentOptions,
      },
    })

    expect(getAllByRole('button', { name: 'Doing' })).toHaveLength(2)

    await fireEvent.click(getAllByRole('button', { name: 'Doing' })[1])

    await waitFor(() => {
      expect(getAllByRole('button', { name: 'Doing' })[0].hasAttribute('disabled')).toBe(true)
    })
  })
})
