import { cleanup, render } from '@testing-library/svelte'
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

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
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

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

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
      harness_ai: { state: 'available', reason: null },
      skill_ai: { state: 'available', reason: null },
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

describe('SkillEditorPage', () => {
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

  it('does not render a standalone fix-and-verify entry in the header', async () => {
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

    loadSkillEditorData.mockResolvedValue({
      skill: {
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
      workflows: [],
    })

    const { findByRole, queryByRole } = render(SkillEditorPage, {
      props: { skillId: 'skill-1' },
    })

    expect(await findByRole('button', { name: 'Save' })).toBeTruthy()
    expect(queryByRole('button', { name: 'Fix & verify' })).toBeNull()
  })
})
