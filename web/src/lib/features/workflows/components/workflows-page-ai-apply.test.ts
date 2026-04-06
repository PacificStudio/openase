import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import WorkflowsPage from './workflows-page.svelte'

const {
  loadWorkflowPageData,
  loadWorkflowHarness,
  saveWorkflowHarness,
  validateHarness,
  bindWorkflowSkills,
  unbindWorkflowSkills,
  listBuiltinRoles,
  getBuiltinRole,
} = vi.hoisted(() => ({
  loadWorkflowPageData: vi.fn(),
  loadWorkflowHarness: vi.fn(),
  saveWorkflowHarness: vi.fn(),
  validateHarness: vi.fn(),
  bindWorkflowSkills: vi.fn(),
  unbindWorkflowSkills: vi.fn(),
  listBuiltinRoles: vi.fn(),
  getBuiltinRole: vi.fn(),
}))

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
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
  loadWorkflowPageData,
  loadWorkflowHarness,
}))

vi.mock('$lib/api/openase', () => ({
  saveWorkflowHarness,
  validateHarness,
  bindWorkflowSkills,
  unbindWorkflowSkills,
  listBuiltinRoles,
  getBuiltinRole,
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

vi.mock('./workflow-creation-dialog.svelte', () => ({
  default: vi.fn(),
}))

const orgFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  status: 'active',
  default_agent_provider_id: null,
}

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'TestProject',
  slug: 'test-project',
  description: '',
  status: 'active',
  default_agent_provider_id: 'provider-1',
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const providerFixture: AgentProvider = {
  id: 'provider-1',
  organization_id: 'org-1',
  machine_id: 'machine-1',
  machine_name: 'Localhost',
  machine_host: '127.0.0.1',
  machine_status: 'online',
  machine_ssh_user: null,
  machine_workspace_root: '/workspace',
  name: 'Codex',
  adapter_type: 'codex-app-server',
  permission_profile: 'unrestricted',
  availability_state: 'available',
  available: true,
  availability_checked_at: '2026-03-28T12:00:00Z',
  availability_reason: null,
  capabilities: {
    ephemeral_chat: {
      state: 'available',
      reason: null,
    },
  },
  cli_command: 'codex',
  cli_args: [],
  auth_config: {},
  cli_rate_limit: null,
  cli_rate_limit_updated_at: null,
  model_name: 'gpt-5.4',
  model_temperature: 0,
  model_max_tokens: 4096,
  max_parallel_runs: 2,
  cost_per_input_token: 0,
  cost_per_output_token: 0,
  pricing_config: {},
}

const pageDataFixture = {
  workflows: [
    {
      id: 'wf-1',
      name: 'Coding Workflow',
      type: 'coding',
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
    },
  ],
  selectedWorkflowId: 'wf-1',
  agentOptions: [],
  providers: [providerFixture],
  skillStates: [],
  builtinRoleContent: '',
  statuses: [
    { id: 'todo', name: 'To Do' },
    { id: 'done', name: 'Done' },
  ],
  variableGroups: [],
  harness: {
    frontmatter: 'type: coding',
    body: 'You are a coding assistant.',
    rawContent: '---\ntype: coding\n---\nYou are a coding assistant.',
  },
}

describe('WorkflowsPage', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('does not render a standalone workflow AI entrypoint', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture
    loadWorkflowPageData.mockResolvedValue(pageDataFixture)
    const { findByRole, queryByRole, container } = render(WorkflowsPage)

    expect(await findByRole('heading', { name: 'Workflows' })).toBeTruthy()
    expect(queryByRole('button', { name: 'AI' })).toBeNull()
    expect(
      container.querySelector('textarea[placeholder="Ask AI to refine this harness…"]'),
    ).toBeNull()
  })
})
