import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/stores/app.svelte', () => ({
  appStore: {
    currentProject: { default_agent_provider_id: 'provider-1' },
  },
}))

import type { AgentProvider } from '$lib/api/contracts'
import SkillAiSidebar from './skill-ai-sidebar.svelte'

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

const fileContent = [
  '---',
  'name: "deploy"',
  'description: "Deploy safely"',
  '---',
  '',
  'Use safe steps.',
].join('\n')

const updatedFileContent = [
  '---',
  'name: "deploy"',
  'description: "Deploy safely"',
  '---',
  '',
  'Use safe steps.',
  '',
  'Verify rollback steps before production deploys.',
].join('\n')

describe('SkillAiSidebar', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('reuses the session and applies a structured diff suggestion for the selected file', async () => {
    let turnCount = 0

    streamChatTurn.mockImplementation(async (request, handlers) => {
      turnCount += 1

      if (turnCount === 1) {
        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-skill-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content: 'I can tighten the deploy instructions. Want a concrete patch?',
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
        return
      }

      expect(request.sessionId).toBe('session-skill-1')

      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: 'session-skill-1' },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'diff',
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
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-skill-1',
          turnsUsed: 2,
          turnsRemaining: 8,
        },
      })
    })

    const appliedSuggestions: Array<{ path: string; content: string }> = []

    const { getByPlaceholderText, getByRole, queryByRole, findByText } = render(SkillAiSidebar, {
      props: {
        projectId: 'project-1',
        providers: providerFixtures,
        skillId: 'skill-1',
        selectedFilePath: 'SKILL.md',
        selectedFileContent: fileContent,
        selectedFileIsText: true,
        onApplySuggestion: (path: string, content: string) =>
          appliedSuggestions.push({ path, content }),
      },
    })

    const prompt = getByPlaceholderText('Ask AI to refine SKILL.md…')
    await fireEvent.input(prompt, {
      target: { value: 'Make the deploy instructions safer.' },
    })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    expect(
      await findByText('I can tighten the deploy instructions. Want a concrete patch?'),
    ).toBeTruthy()
    expect(streamChatTurn).toHaveBeenCalledTimes(1)
    expect(streamChatTurn.mock.calls[0][0]).toMatchObject({
      message: 'Make the deploy instructions safer.',
      source: 'skill_editor',
      providerId: 'provider-1',
      context: {
        projectId: 'project-1',
        skillId: 'skill-1',
        skillFilePath: 'SKILL.md',
        skillFileDraft: fileContent,
      },
    })

    await fireEvent.input(prompt, { target: { value: 'Yes, show the patch.' } })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    expect(await findByText('Structured Diff')).toBeTruthy()
    expect((await screen.findAllByText('SKILL.md')).length).toBeGreaterThanOrEqual(2)

    await fireEvent.click(getByRole('button', { name: 'Apply' }))

    expect(appliedSuggestions).toEqual([{ path: 'SKILL.md', content: updatedFileContent }])

    await waitFor(() => {
      expect(queryByRole('button', { name: 'Apply' })).toBeNull()
    })
  })
})
