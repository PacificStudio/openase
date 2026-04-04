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
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
      harness_ai: {
        state: 'available',
        reason: null,
      },
      skill_ai: {
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

const updatedHarnessContent = [
  '---',
  'name: Coding Workflow',
  'type: coding',
  '---',
  '',
  'Write clean, tested code with full coverage.',
  '',
  'Ensure every public function has at least one unit test.',
].join('\n')

describe('HarnessAiSidebar', () => {
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

  it('completes a two-turn conversation and displays a structured diff suggestion', async () => {
    let turnCount = 0

    streamChatTurn.mockImplementation(async (request, handlers) => {
      turnCount += 1

      if (turnCount === 1) {
        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-harness-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content: 'I can add a test coverage section. Want me to proceed?',
          },
        })
        handlers.onEvent({
          kind: 'done',
          payload: {
            sessionId: 'session-harness-1',
            turnsUsed: 1,
            turnsRemaining: 9,
          },
        })
      } else {
        // Second turn reuses session and returns a structured diff
        expect(request.sessionId).toBe('session-harness-1')

        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-harness-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content: 'Here is the updated harness with test coverage guidance.',
          },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'diff',
            file: 'harness content',
            hunks: [
              {
                oldStart: 5,
                oldLines: 2,
                newStart: 5,
                newLines: 4,
                lines: [
                  { op: 'context', text: '' },
                  { op: 'remove', text: 'Write clean, tested code.' },
                  { op: 'add', text: 'Write clean, tested code with full coverage.' },
                  { op: 'add', text: '' },
                  { op: 'add', text: 'Ensure every public function has at least one unit test.' },
                ],
              },
            ],
          },
        })
        handlers.onEvent({
          kind: 'done',
          payload: {
            sessionId: 'session-harness-1',
            turnsUsed: 2,
            turnsRemaining: 8,
          },
        })
      }
    })

    const appliedSuggestions: string[] = []

    const { getByPlaceholderText, getByRole, findByText } = render(HarnessAiSidebar, {
      props: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        providers: providerFixtures,
        draftContent: harnessContent,
        onApplySuggestion: (content: string) => appliedSuggestions.push(content),
      },
    })

    // Turn 1: ask about test coverage
    const prompt = getByPlaceholderText('Ask AI to refine this harness…')
    await fireEvent.input(prompt, {
      target: { value: 'Add a section about test coverage requirements.' },
    })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    expect(await findByText('I can add a test coverage section. Want me to proceed?')).toBeTruthy()

    // Verify turn 1 sent with harness_editor source
    expect(streamChatTurn).toHaveBeenCalledTimes(1)
    expect(streamChatTurn.mock.calls[0][0]).toMatchObject({
      message: 'Add a section about test coverage requirements.',
      source: 'harness_editor',
      providerId: 'provider-1',
      sessionId: undefined,
      context: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        harnessDraft: harnessContent,
      },
    })

    // Turn 2: confirm — assistant returns diff
    await fireEvent.input(prompt, { target: { value: 'Yes, proceed.' } })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    expect(
      await findByText('Here is the updated harness with test coverage guidance.'),
    ).toBeTruthy()

    // Verify turn 2 reused the session
    expect(streamChatTurn).toHaveBeenCalledTimes(2)
    expect(streamChatTurn.mock.calls[1][0]).toMatchObject({
      sessionId: 'session-harness-1',
    })

    // The diff entry should render in the transcript
    expect(await findByText('Structured Diff')).toBeTruthy()

    // The suggestion card should appear with an Apply button
    expect(await findByText('Apply')).toBeTruthy()

    // Click Apply
    await fireEvent.click(getByRole('button', { name: 'Apply' }))

    expect(appliedSuggestions).toEqual([updatedHarnessContent])

    // After applying, the applied label should appear
    await waitFor(() => {
      expect(document.body.textContent).toContain('Applied')
    })
  })
})
