import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import ProjectPipelinePresetPanel from './project-pipeline-preset-panel.svelte'

const { applyPipelinePreset, listAgents, listPipelinePresets, listStatuses, listWorkflows } =
  vi.hoisted(() => ({
    applyPipelinePreset: vi.fn(),
    listAgents: vi.fn(),
    listPipelinePresets: vi.fn(),
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
  applyPipelinePreset,
  listAgents,
  listPipelinePresets,
  listStatuses,
  listWorkflows,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('ProjectPipelinePresetPanel', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('shows the active-ticket gate when preset apply is blocked', async () => {
    appStore.currentProject = currentProject()
    listPipelinePresets.mockResolvedValue({
      active_ticket_count: 2,
      can_apply: false,
      presets: [fullstackPreset()],
    })
    listAgents.mockResolvedValue({ agents: [agentFixture()] })
    listStatuses.mockResolvedValue({ statuses: [] })
    listWorkflows.mockResolvedValue({ workflows: [] })

    const { findByText, getByRole } = render(ProjectPipelinePresetPanel)

    expect(await findByText('Preset apply is currently blocked')).toBeTruthy()
    expect(
      await findByText(
        'This project currently has 2 active tickets. Pipeline presets only apply when active ticket count is 0.',
      ),
    ).toBeTruthy()
    expect(getByRole('button', { name: 'Apply preset' }).hasAttribute('disabled')).toBe(true)
  })

  it('applies the selected preset with the default agent binding', async () => {
    appStore.currentProject = currentProject()
    listPipelinePresets.mockResolvedValue({
      active_ticket_count: 0,
      can_apply: true,
      presets: [fullstackPreset()],
    })
    listAgents.mockResolvedValue({ agents: [agentFixture()] })
    listStatuses.mockResolvedValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted',
          color: '#3B82F6',
          icon: 'list-todo',
          position: 0,
          active_runs: 0,
          is_default: true,
          description: 'Ready',
        },
      ],
    })
    listWorkflows.mockResolvedValue({ workflows: [] })
    applyPipelinePreset.mockResolvedValue({
      result: {
        preset: fullstackPreset(),
        active_ticket_count: 0,
        statuses: [],
        workflows: [],
      },
    })

    const { findAllByText, getByRole } = render(ProjectPipelinePresetPanel)

    expect((await findAllByText('Fullstack Delivery Pipeline')).length).toBeGreaterThan(0)
    await fireEvent.click(getByRole('button', { name: 'Apply preset' }))

    await waitFor(() =>
      expect(applyPipelinePreset).toHaveBeenCalledWith('project-1', 'fullstack-default', {
        workflow_agent_bindings: [
          {
            workflow_key: 'fullstack-developer',
            agent_id: 'agent-1',
          },
        ],
      }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('Applied Fullstack Delivery Pipeline.')
  })
})

function currentProject(): Project {
  return {
    id: 'project-1',
    organization_id: 'org-1',
    name: 'OpenASE',
    slug: 'openase',
    description: '',
    status: 'active',
    default_agent_provider_id: null,
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
    agent_run_summary_prompt: '',
    effective_agent_run_summary_prompt: '',
    agent_run_summary_prompt_source: 'builtin',
    project_ai_retention: {
      enabled: false,
      keep_latest_n: 0,
      keep_recent_days: 0,
    },
  }
}

function agentFixture() {
  return {
    id: 'agent-1',
    provider_id: 'provider-1',
    project_id: 'project-1',
    name: 'Backend Engineer',
    runtime_control_state: 'active',
    total_tokens_used: 0,
    total_tickets_completed: 0,
    runtime: null,
  }
}

function fullstackPreset() {
  return {
    version: 1,
    preset: {
      key: 'fullstack-default',
      name: 'Fullstack Delivery Pipeline',
      description: 'Starter pipeline.',
    },
    statuses: [
      {
        name: 'Todo',
        stage: 'unstarted',
        color: '#3B82F6',
      },
      {
        name: 'Done',
        stage: 'completed',
        color: '#10B981',
      },
    ],
    workflows: [
      {
        key: 'fullstack-developer',
        name: 'Fullstack Developer Workflow',
        type: 'coding',
        role_slug: 'fullstack-developer',
        role_name: 'Fullstack Developer',
        role_description: 'Implement product changes end to end.',
        max_concurrent: 0,
        max_retry_attempts: 1,
        timeout_minutes: 60,
        stall_timeout_minutes: 5,
        pickup_statuses: ['Todo'],
        finish_statuses: ['Done'],
      },
    ],
  }
}
