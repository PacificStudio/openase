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

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
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

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
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

  it('applies an AI multi-file suggestion back into the skill bundle editor', async () => {
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

    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: 'session-skill-1' },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'bundle_diff',
          files: [
            {
              file: 'SKILL.md',
              hunks: [
                {
                  oldStart: 5,
                  oldLines: 2,
                  newStart: 5,
                  newLines: 4,
                  lines: [
                    { op: 'context', text: '' },
                    { op: 'context', text: 'Use safe steps.' },
                    { op: 'add', text: '' },
                    { op: 'add', text: 'Verify rollback steps before production deploys.' },
                  ],
                },
              ],
            },
            {
              file: 'references/runbook.md',
              hunks: [
                {
                  oldStart: 1,
                  oldLines: 0,
                  newStart: 1,
                  newLines: 3,
                  lines: [
                    { op: 'add', text: '# Runbook' },
                    { op: 'add', text: '' },
                    { op: 'add', text: '1. Verify rollback before deploy.' },
                  ],
                },
              ],
            },
          ],
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-skill-1',
          turnsUsed: 1,
          turnsRemaining: 9,
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

    await fireEvent.click(await findByRole('button', { name: 'AI' }))

    const prompt = await findByPlaceholderText('Ask AI to refine SKILL.md…')
    await fireEvent.input(prompt, { target: { value: 'Make the deploy skill safer.' } })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    await fireEvent.click(await findByRole('button', { name: 'references/runbook.md' }))
    await fireEvent.click(await findByRole('button', { name: 'Apply All' }))

    const saveButton = getByRole('button', { name: 'Save' }) as HTMLButtonElement
    await waitFor(() => {
      expect(resolvedEditor.value).toBe(runbookContent)
      expect(saveButton.disabled).toBe(false)
    })

    expect(streamChatTurn).toHaveBeenCalledWith(
      expect.objectContaining({
        source: 'skill_editor',
        context: expect.objectContaining({
          skillId: 'skill-1',
          skillFilePath: 'SKILL.md',
        }),
      }),
      expect.any(Object),
    )
  })
})
