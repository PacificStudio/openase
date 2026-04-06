import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { loadSkillEditorData } = vi.hoisted(() => ({
  loadSkillEditorData: vi.fn(),
}))

const { bindSkill, deleteSkill, disableSkill, enableSkill, unbindSkill, updateSkill } = vi.hoisted(
  () => ({
    bindSkill: vi.fn(),
    deleteSkill: vi.fn(),
    disableSkill: vi.fn(),
    enableSkill: vi.fn(),
    unbindSkill: vi.fn(),
    updateSkill: vi.fn(),
  }),
)

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

vi.mock('$app/navigation', () => ({
  goto,
  beforeNavigate: vi.fn(),
}))

vi.mock('./skill-editor-page.helpers', async () => {
  const actual = await vi.importActual<typeof import('./skill-editor-page.helpers')>(
    './skill-editor-page.helpers',
  )
  return {
    ...actual,
    loadSkillEditorData,
  }
})

vi.mock('$lib/api/openase', () => ({
  bindSkill,
  deleteSkill,
  disableSkill,
  enableSkill,
  unbindSkill,
  updateSkill,
}))

vi.mock('$lib/stores/toast.svelte', () => ({ toastStore }))

import { appStore } from '$lib/stores/app.svelte'
import type { AgentProvider } from '$lib/api/contracts'
import SkillEditorPage from './skill-editor-page.svelte'

const providerFixtures: AgentProvider[] = [
  {
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
      ephemeral_chat: { state: 'available', reason: null },
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
  },
]

const initialContent = [
  '---',
  'name: "deploy"',
  'description: "Deploy safely"',
  '---',
  '',
  'Use safe steps.',
].join('\n')

function buildSkillEditorData(overrides: Record<string, unknown> = {}) {
  const { skill: rawSkill, workflows: rawWorkflows, ...restOverrides } = overrides
  const baseSkill = {
    id: 'skill-1',
    name: 'deploy',
    description: 'Deploy safely',
    path: '.openase/skills/deploy/SKILL.md',
    current_version: 3,
    is_builtin: false,
    is_enabled: true,
    created_by: 'user:manual',
    created_at: '2026-04-01T12:00:00Z',
    bound_workflows: [],
  }
  return {
    skill: {
      ...baseSkill,
      ...((rawSkill as Record<string, unknown> | undefined) ?? {}),
    },
    content: initialContent,
    files: [
      {
        path: 'SKILL.md',
        file_kind: 'entrypoint',
        media_type: 'text/markdown; charset=utf-8',
        encoding: 'utf8',
        is_executable: false,
        size_bytes: initialContent.length,
        sha256: 'sha-entry',
        content: initialContent,
        content_base64: 'ignored',
      },
    ],
    history: [],
    workflows: (rawWorkflows as Record<string, unknown>[] | undefined) ?? [],
    ...restOverrides,
  }
}

async function renderPage(overrides: Record<string, unknown> = {}) {
  appStore.currentOrg = {
    id: 'org-1',
    name: 'OpenAI',
    slug: 'openai',
    status: 'active',
    default_agent_provider_id: 'provider-1',
  }
  appStore.currentProject = {
    id: 'project-1',
    organization_id: 'org-1',
    name: 'OpenASE',
    slug: 'openase',
    description: '',
    status: 'active',
    default_agent_provider_id: 'provider-1',
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
  }
  appStore.providers = providerFixtures
  loadSkillEditorData.mockResolvedValue(buildSkillEditorData(overrides))
  return render(SkillEditorPage, {
    props: { skillId: 'skill-1' },
  })
}

describe('SkillEditorPage actions', () => {
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
    appStore.providers = []
    vi.clearAllMocks()
  })

  it('toggles the enabled state and updates the header action', async () => {
    loadSkillEditorData.mockResolvedValue(
      buildSkillEditorData({
        skill: {
          is_enabled: true,
        },
      }),
    )
    disableSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        is_enabled: false,
      },
    })
    enableSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        is_enabled: true,
      },
    })

    const { findByTitle } = await renderPage({
      skill: {
        is_enabled: true,
      },
    })

    const disableButton = await findByTitle('Disable')
    await fireEvent.click(disableButton)

    await waitFor(() => {
      expect(disableSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Disabled deploy.')
    })

    const enableButton = await findByTitle('Enable')
    await fireEvent.click(enableButton)

    await waitFor(() => {
      expect(enableSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Enabled deploy.')
    })
  })

  it('binds and unbinds workflows from the metadata bar', async () => {
    loadSkillEditorData.mockResolvedValue(
      buildSkillEditorData({
        workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
      }),
    )
    bindSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        bound_workflows: [{ id: 'workflow-1' }],
      },
    })
    unbindSkill.mockResolvedValue({
      skill: {
        ...buildSkillEditorData().skill,
        bound_workflows: [],
      },
    })

    const { findByTitle } = await renderPage({
      workflows: [{ id: 'workflow-1', name: 'Coding Workflow' }],
    })

    await fireEvent.click(await findByTitle('Bind to Coding Workflow'))

    await waitFor(() => {
      expect(bindSkill).toHaveBeenCalledWith('skill-1', ['workflow-1'])
      expect(toastStore.success).toHaveBeenCalledWith('Bound deploy to Coding Workflow.')
    })

    await fireEvent.click(await findByTitle('Unbind from Coding Workflow'))

    await waitFor(() => {
      expect(unbindSkill).toHaveBeenCalledWith('skill-1', ['workflow-1'])
      expect(toastStore.success).toHaveBeenCalledWith('Unbound deploy from Coding Workflow.')
    })
  })

  it('deletes the skill after confirmation and navigates back to the skills page', async () => {
    loadSkillEditorData.mockResolvedValue(buildSkillEditorData())
    deleteSkill.mockResolvedValue({ skill: { id: 'skill-1' } })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const { findByTitle } = await renderPage()

    await fireEvent.click(await findByTitle('Delete skill'))

    await waitFor(() => {
      expect(deleteSkill).toHaveBeenCalledWith('skill-1')
      expect(toastStore.success).toHaveBeenCalledWith('Deleted deploy.')
      expect(goto).toHaveBeenCalledWith('/orgs/org-1/projects/project-1/skills')
    })
  })
})
