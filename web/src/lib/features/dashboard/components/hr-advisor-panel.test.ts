import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { BuiltinRolePayload, HRAdvisorActivationResponse } from '$lib/api/contracts'
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
    active_workflow_types: ['coding'],
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
      workflow_type: 'qa',
      summary: 'Add a QA lane before more coding tickets pile up.',
      harness_path: '.openase/harnesses/roles/qa-engineer.md',
      priority: 'high',
      reason: 'Coding throughput has increased without matching test coverage.',
      evidence: ['8 coding tickets are active.', 'No QA workflow is currently active.'],
      suggested_headcount: 1,
      suggested_workflow_name: 'QA Workflow',
      activation_ready: true,
      active_workflow_name: null,
    },
    {
      role_slug: 'technical-writer',
      role_name: 'Technical Writer',
      workflow_type: 'docs',
      summary: 'Documentation updates are lagging behind recent merges.',
      harness_path: '.openase/harnesses/roles/technical-writer.md',
      priority: 'medium',
      reason: 'Merged changes keep landing without synchronized docs updates.',
      evidence: ['Recent PRs landed without docs follow-up.'],
      suggested_headcount: 1,
      suggested_workflow_name: 'Docs Workflow',
      activation_ready: true,
      active_workflow_name: null,
    },
  ],
}

const builtinRolesFixture: BuiltinRolePayload = {
  roles: [
    {
      slug: 'qa-engineer',
      name: 'QA Engineer',
      summary: 'Owns regression and test expansion.',
      workflow_type: 'qa',
      harness_path: '.openase/harnesses/roles/qa-engineer.md',
      content: `---
workflow:
  role: qa-engineer
---
Validate the changed code paths and expand test coverage.
`,
    },
    {
      slug: 'technical-writer',
      name: 'Technical Writer',
      summary: 'Owns release notes and operational documentation.',
      workflow_type: 'docs',
      harness_path: '.openase/harnesses/roles/technical-writer.md',
      content: 'Document the shipped behavior.',
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
    type: 'qa',
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

  it('groups recommendations by priority and supports harness preview, activation, and deferral', async () => {
    listBuiltinRoles.mockResolvedValue(builtinRolesFixture)
    activateHRRecommendation.mockResolvedValue(activationFixture)
    getHRAdvisor.mockResolvedValue({
      summary: refreshedAdvisorFixture.summary,
      staffing: refreshedAdvisorFixture.staffing,
      recommendations: refreshedAdvisorFixture.recommendations,
    })

    const { findByText, findByRole, queryByText } = render(HRAdvisorPanel, {
      props: {
        projectId: 'project-1',
        advisor: advisorFixture,
      },
    })

    expect(await findByText('高优先级')).toBeTruthy()
    expect(await findByText('中优先级')).toBeTruthy()

    const qaCard = (await findByText('QA Engineer')).closest('article')
    if (!qaCard) {
      throw new Error('QA recommendation card not found.')
    }

    await fireEvent.click(within(qaCard).getByRole('button', { name: '查看角色 Harness' }))

    expect(listBuiltinRoles).toHaveBeenCalledTimes(1)
    expect(await findByText('Owns regression and test expansion.')).toBeTruthy()
    expect(await findByText(/Validate the changed code paths/)).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Close' }))
    await fireEvent.click(within(qaCard).getByRole('button', { name: '一键激活' }))

    expect(activateHRRecommendation).toHaveBeenCalledWith('project-1', {
      role_slug: 'qa-engineer',
      create_bootstrap_ticket: true,
    })

    await waitFor(() => {
      expect(getHRAdvisor).toHaveBeenCalledWith('project-1')
    })

    expect(await findByText('已通过 QA Workflow 激活。')).toBeTruthy()

    const activatedQaCard = (await findByText('QA Engineer')).closest('article')
    if (!activatedQaCard) {
      throw new Error('Activated QA recommendation card not found.')
    }
    expect(
      (within(activatedQaCard).getByRole('button', { name: '一键激活' }) as HTMLButtonElement)
        .disabled,
    ).toBe(true)

    const writerCard = (await findByText('Technical Writer')).closest('article')
    if (!writerCard) {
      throw new Error('Writer recommendation card not found.')
    }

    await fireEvent.click(within(writerCard).getByRole('button', { name: '稍后再说' }))

    expect(await findByText('已延后')).toBeTruthy()
    expect(await findByRole('button', { name: '重新显示' })).toBeTruthy()
    expect(queryByText('中优先级')).toBeNull()

    await fireEvent.click(await findByRole('button', { name: '重新显示' }))

    expect(await findByText('中优先级')).toBeTruthy()
  })
})
