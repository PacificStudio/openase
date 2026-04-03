import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HRAdvisorActivationResponse } from '$lib/api/contracts'
import type { HRAdvisorSnapshot } from '../types'
import HRAdvisorPanel from './hr-advisor-panel.svelte'

const { activateHRRecommendation, getHRAdvisor, listBuiltinRoles } = vi.hoisted(() => ({
  activateHRRecommendation: vi.fn(),
  getHRAdvisor: vi.fn(),
  listBuiltinRoles: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  activateHRRecommendation,
  getHRAdvisor,
  listBuiltinRoles,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

const advisorFixture: HRAdvisorSnapshot = {
  summary: {
    open_tickets: 12,
    coding_tickets: 8,
    failing_tickets: 1,
    blocked_tickets: 2,
    active_agents: 1,
    workflow_count: 2,
    recent_activity_count: 5,
    active_workflow_families: ['coding'],
  },
  staffing: {
    developers: 1,
    qa: 1,
    docs: 1,
    security: 0,
    product: 0,
    research: 0,
  },
  recommendations: [
    {
      role_slug: 'qa-engineer',
      role_name: 'QA Engineer',
      workflow_type: 'QA Engineer',
      workflow_family: 'test',
      summary: 'Add a QA lane before more coding tickets pile up.',
      harness_path: '.openase/harnesses/roles/qa-engineer.md',
      priority: 'high',
      reason: 'Coding throughput has increased without matching test coverage.',
      evidence: ['8 coding tickets are active.', 'No QA workflow is currently active.'],
      suggested_headcount: 1,
      suggested_workflow_name: 'QA Workflow',
      suggested_workflow_type: 'QA Engineer',
      suggested_workflow_family: 'test',
      activation_ready: true,
      active_workflow_name: null,
    },
    {
      role_slug: 'technical-writer',
      role_name: 'Technical Writer',
      workflow_type: 'Technical Writer',
      workflow_family: 'docs',
      summary: 'Documentation updates are lagging behind recent merges.',
      harness_path: '.openase/harnesses/roles/technical-writer.md',
      priority: 'medium',
      reason: 'Merged changes keep landing without synchronized docs updates.',
      evidence: ['Recent PRs landed without docs follow-up.'],
      suggested_headcount: 1,
      suggested_workflow_name: 'Docs Workflow',
      suggested_workflow_type: 'Technical Writer',
      suggested_workflow_family: 'docs',
      activation_ready: true,
      active_workflow_name: null,
    },
  ],
}

const activationFixture = {
  project_id: 'project-1',
  role_slug: 'qa-engineer',
  agent: {
    id: 'agent-1',
    name: 'QA Engineer',
    project_id: 'project-1',
    provider_id: 'provider-1',
    runtime: null,
    runtime_control_state: 'active',
    total_tickets_completed: 0,
    total_tokens_used: 0,
  },
  workflow: {
    id: 'workflow-1',
    agent_id: 'agent-1',
    finish_status_ids: ['done'],
    harness_content: null,
    harness_path: '.openase/harnesses/roles/qa-engineer.md',
    hooks: {},
    is_active: true,
    max_concurrent: 1,
    max_retry_attempts: 1,
    name: 'QA Workflow',
    pickup_status_ids: ['todo'],
    project_id: 'project-1',
    stall_timeout_minutes: 10,
    timeout_minutes: 30,
    type: 'QA Engineer',
    workflow_family: 'test',
    workflow_classification: {
      family: 'test',
      confidence: 1,
      reasons: ['matched explicit built-in role slug'],
    },
    version: 1,
  },
  bootstrap_ticket: {
    requested: true,
    status: 'created',
    message: 'Bootstrap ticket created.',
  },
} as unknown as HRAdvisorActivationResponse

const refreshedAdvisorFixture: HRAdvisorSnapshot = {
  ...advisorFixture,
  summary: {
    ...advisorFixture.summary,
    workflow_count: 3,
  },
  recommendations: advisorFixture.recommendations.map((recommendation) =>
    recommendation.role_slug === 'qa-engineer'
      ? {
          ...recommendation,
          activation_ready: false,
          active_workflow_name: 'QA Workflow',
        }
      : recommendation,
  ),
}

describe('HRAdvisorPanel', () => {
  afterEach(() => {
    cleanup()
    window.localStorage.clear()
    vi.clearAllMocks()
  })

  it('renders compact recommendation rows with expand/collapse', async () => {
    const { findByText, queryByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    // Both recommendations visible as compact rows
    expect(await findByText('QA Engineer')).toBeTruthy()
    expect(await findByText('Technical Writer')).toBeTruthy()

    // Details not visible before expand
    expect(queryByText('Add a QA lane before more coding tickets pile up.')).toBeNull()

    // Expand QA card
    const qaRow = (await findByText('QA Engineer')).closest('article')!
    await fireEvent.click(qaRow.querySelector('button[type="button"]')!)

    // Details now visible
    expect(await findByText('Add a QA lane before more coding tickets pile up.')).toBeTruthy()
    expect(await findByText('8 coding tickets are active.')).toBeTruthy()
  })

  it('activates a recommendation via inline button', async () => {
    activateHRRecommendation.mockResolvedValue(activationFixture)
    getHRAdvisor.mockResolvedValue({
      summary: refreshedAdvisorFixture.summary,
      staffing: refreshedAdvisorFixture.staffing,
      recommendations: refreshedAdvisorFixture.recommendations,
    })

    const { findByText, findAllByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    // Click the inline activate button (first one = QA)
    const activateButtons = await findAllByText('激活')
    await fireEvent.click(activateButtons[0].closest('button')!)

    expect(activateHRRecommendation).toHaveBeenCalledWith('project-1', {
      role_slug: 'qa-engineer',
      create_bootstrap_ticket: true,
    })

    await waitFor(() => {
      expect(getHRAdvisor).toHaveBeenCalledWith('project-1')
    })

    // Expand QA row to see activation status
    const qaRow = (await findByText('QA Engineer')).closest('article')!
    await fireEvent.click(qaRow.querySelector('button[type="button"]')!)

    expect(await findByText('已通过 QA Workflow 激活。')).toBeTruthy()
  })

  it('shows activatable count in header', async () => {
    const { findByText } = render(HRAdvisorPanel, {
      props: { projectId: 'project-1', advisor: advisorFixture },
    })

    expect(await findByText('2 可激活')).toBeTruthy()
    expect(await findByText('2 workflows')).toBeTruthy()
  })
})
