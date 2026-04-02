import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
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
import HarnessAiSidebar from './harness-ai-sidebar.svelte'

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

const harnessContent = [
  '---',
  'name: Coding Workflow',
  'type: coding',
  '---',
  '',
  'Write clean, tested code.',
].join('\n')

describe('HarnessAiSidebar dismiss suggestion', () => {
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
    vi.clearAllMocks()
  })

  it('dismisses the current suggestion without applying it', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({ kind: 'session', payload: { sessionId: 'session-harness-dismiss-1' } })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'diff',
          file: 'harness content',
          hunks: [
            {
              oldStart: 6,
              oldLines: 1,
              newStart: 6,
              newLines: 1,
              lines: [
                { op: 'remove', text: 'Write clean, tested code.' },
                { op: 'add', text: 'Write clean, tested code with guardrails.' },
              ],
            },
          ],
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: { sessionId: 'session-harness-dismiss-1', turnsUsed: 1, turnsRemaining: 9 },
      })
    })

    const appliedSuggestions: string[] = []
    const { getByPlaceholderText, getByRole, findByText, queryByText } = render(HarnessAiSidebar, {
      props: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        providers: providerFixtures,
        draftContent: harnessContent,
        onApplySuggestion: (content: string) => appliedSuggestions.push(content),
      },
    })

    const prompt = getByPlaceholderText('Ask AI to refine this harness…')
    await fireEvent.input(prompt, { target: { value: 'Tighten this harness.' } })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    expect(await findByText('Apply')).toBeTruthy()
    await fireEvent.click(getByRole('button', { name: 'Dismiss suggestion' }))

    await waitFor(() => {
      expect(queryByText('Apply')).toBeNull()
      expect(queryByText('Dismiss suggestion')).toBeNull()
    })
    expect(appliedSuggestions).toEqual([])
  })
})
