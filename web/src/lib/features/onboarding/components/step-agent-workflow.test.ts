import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Agent, TicketStatus, Workflow } from '$lib/api/contracts'
import StepAgentWorkflow from './step-agent-workflow.svelte'

const { createAgent, createWorkflow, listAgents, listBuiltinRoles, listStatuses, listWorkflows } =
  vi.hoisted(() => ({
    createAgent: vi.fn(),
    createWorkflow: vi.fn(),
    listAgents: vi.fn(),
    listBuiltinRoles: vi.fn(),
    listStatuses: vi.fn(),
    listWorkflows: vi.fn(),
  }))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  createAgent,
  createWorkflow,
  listAgents,
  listBuiltinRoles,
  listStatuses,
  listWorkflows,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('StepAgentWorkflow', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
  })

  beforeEach(() => {
    listStatuses.mockResolvedValue({
      statuses: [
        makeStatus({ id: 'status-backlog', name: 'Backlog', stage: 'backlog' }),
        makeStatus({ id: 'status-todo', name: 'Todo', stage: 'unstarted' }),
        makeStatus({ id: 'status-done', name: 'Done', stage: 'completed' }),
      ],
    })
    listBuiltinRoles.mockResolvedValue({
      roles: [{ slug: 'product-manager', content: '# Product Manager' }],
    })
    createAgent.mockResolvedValue({
      agent: { id: 'agent-1', name: 'product-manager-01' },
    })
    listAgents.mockResolvedValue({
      agents: [makeAgent()],
    })
    listWorkflows.mockResolvedValue({
      workflows: [makeWorkflow()],
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('shows preset picker before the preview, then creates after selection', async () => {
    const onComplete = vi.fn()
    const { getByText } = render(StepAgentWorkflow, {
      props: {
        projectId: 'project-1',
        providerId: 'provider-1',
        projectStatus: 'Planned',
        initialState: {
          agents: [],
          workflows: [],
          statuses: [],
        },
        onComplete,
      },
    })

    // Picker is shown first — all three preset titles visible
    expect(getByText('Write code')).toBeTruthy()
    expect(getByText('Plan the project')).toBeTruthy()
    expect(getByText('Explore ideas')).toBeTruthy()

    // Select the PM preset
    await fireEvent.click(getByText('Plan the project'))

    // Preview is now shown with a Change link and the status flow
    expect(getByText('Pickup: Backlog → Finish: Done')).toBeTruthy()
    expect(getByText('Change')).toBeTruthy()

    await fireEvent.click(getByText('Create agent and workflow'))

    await waitFor(() => {
      expect(createWorkflow).toHaveBeenCalledWith('project-1', {
        agent_id: 'agent-1',
        name: 'Product Manager Workflow',
        type: 'Product Manager',
        role_slug: 'product-manager',
        role_name: 'Product Manager',
        role_description: 'Product Manager',
        platform_access_allowed: [],
        skill_names: [],
        pickup_status_ids: ['status-backlog'],
        finish_status_ids: ['status-done'],
        harness_content: '# Product Manager',
        is_active: true,
        max_concurrent: 0,
        max_retry_attempts: 1,
        timeout_minutes: 30,
      })
      expect(onComplete).toHaveBeenCalledWith(expect.any(Array), expect.any(Array), 'pm')
    })
  })

  it('allows going back to the picker from the preview', async () => {
    const { getByText } = render(StepAgentWorkflow, {
      props: {
        projectId: 'project-1',
        providerId: 'provider-1',
        projectStatus: 'Planned',
        initialState: { agents: [], workflows: [], statuses: [] },
        onComplete: vi.fn(),
      },
    })

    await fireEvent.click(getByText('Plan the project'))
    expect(getByText('Change')).toBeTruthy()

    await fireEvent.click(getByText('Change'))
    expect(getByText('Write code')).toBeTruthy()
  })
})

function makeStatus(overrides: Partial<TicketStatus> = {}): TicketStatus {
  const status: TicketStatus = {
    id: 'status-1',
    project_id: 'project-1',
    name: 'Todo',
    stage: 'unstarted',
    color: '#3B82F6',
    icon: '',
    position: 1,
    is_default: false,
    active_runs: 0,
    max_active_runs: 0,
    description: '',
    ...overrides,
  }
  status.active_runs = overrides.active_runs ?? 0
  return status
}

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    id: 'agent-1',
    project_id: 'project-1',
    provider_id: 'provider-1',
    name: 'product-manager-01',
    runtime: null,
    runtime_control_state: 'active',
    total_tickets_completed: 0,
    total_tokens_used: 0,
    ...overrides,
  }
}

function makeWorkflow(overrides: Partial<Workflow> = {}): Workflow {
  return {
    id: 'workflow-1',
    project_id: 'project-1',
    agent_id: 'agent-1',
    name: 'Product Manager Workflow',
    type: 'Product Manager',
    workflow_family: 'planning',
    workflow_classification: {
      family: 'planning',
      confidence: 1,
      reasons: ['fixture'],
    },
    pickup_status_ids: ['status-backlog'],
    finish_status_ids: ['status-done'],
    max_concurrent: 0,
    max_retry_attempts: 1,
    timeout_minutes: 30,
    stall_timeout_minutes: 10,
    is_active: true,
    harness_path: '',
    harness_content: null,
    hooks: {},
    version: 1,
    ...overrides,
  }
}
