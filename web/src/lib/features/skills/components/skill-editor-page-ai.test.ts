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

const { closeSkillRefinementSession, streamSkillRefinement } = vi.hoisted(() => ({
  closeSkillRefinementSession: vi.fn(),
  streamSkillRefinement: vi.fn(),
}))

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

vi.mock('$lib/api/skill-refinement', () => ({
  closeSkillRefinementSession,
  streamSkillRefinement,
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

const runbookContent = ['# Runbook', '', '1. Verify rollback before deploy.'].join('\n')

describe('SkillEditorPage AI bundle apply', () => {
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

  it('applies an AI multi-file suggestion back into the skill bundle editor', async () => {
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

    streamSkillRefinement.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-skill-1',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-skill-1/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-skill-1',
          phase: 'testing',
          attempt: 1,
          message: 'Codex is running verification commands.',
        },
      })
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-skill-1',
          status: 'verified',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-skill-1/workspace',
          providerId: 'provider-1',
          providerName: 'Codex',
          attempts: 1,
          transcriptSummary: 'Bundle verified after tightening the deploy instructions.',
          commandOutputSummary: 'bash -n scripts/redeploy.sh\n./scripts/redeploy.sh',
          candidateBundleHash: 'bundle-hash-1',
          candidateFiles: [
            {
              path: 'SKILL.md',
              file_kind: 'entrypoint',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: initialContent.length + 49,
              sha256: 'sha-entry-verified',
              content: [
                '---',
                'name: "deploy"',
                'description: "Deploy safely"',
                '---',
                '',
                'Use safe steps.',
                '',
                'Verify rollback steps before production deploys.',
              ].join('\n'),
              content_base64: 'ignored',
            },
            {
              path: 'references/runbook.md',
              file_kind: 'reference',
              media_type: 'text/markdown; charset=utf-8',
              encoding: 'utf8',
              is_executable: false,
              size_bytes: runbookContent.length,
              sha256: 'sha-runbook',
              content: runbookContent,
              content_base64: 'ignored',
            },
          ],
        },
      })
    })

    const { container, findByRole, findByPlaceholderText, getByRole } = render(SkillEditorPage, {
      props: { skillId: 'skill-1' },
    })

    let editor: HTMLTextAreaElement | undefined
    await waitFor(() => {
      const candidates = [...container.querySelectorAll('textarea')]
      editor =
        candidates.find(
          (item): item is HTMLTextAreaElement =>
            item instanceof HTMLTextAreaElement && item.value === initialContent,
        ) ?? undefined
      expect(editor).toBeDefined()
    })

    if (!editor) {
      throw new Error('expected skill editor textarea to render')
    }
    const resolvedEditor = editor
    expect(resolvedEditor.value).toBe(initialContent)

    await fireEvent.click(await findByRole('button', { name: 'Fix & verify' }))

    const prompt = await findByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, { target: { value: 'Make the deploy skill safer.' } })
    await fireEvent.click(await findByRole('button', { name: 'Fix and verify' }))

    await fireEvent.click(await findByRole('button', { name: 'references/runbook.md' }))
    await fireEvent.click(await findByRole('button', { name: 'Apply All' }))

    const saveButton = getByRole('button', { name: 'Save' }) as HTMLButtonElement
    await waitFor(() => {
      expect(resolvedEditor.value).toBe(runbookContent)
      expect(saveButton.disabled).toBe(false)
    })

    expect(streamSkillRefinement).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        skillId: 'skill-1',
        providerId: 'provider-1',
        message: 'Make the deploy skill safer.',
      }),
      expect.any(Object),
    )
  })
})
