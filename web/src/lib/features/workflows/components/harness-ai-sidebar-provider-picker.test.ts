import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('$lib/api/chat', () => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
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

describe('HarnessAiSidebar provider picker', () => {
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

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('filters out remote-only providers from the Harness AI picker', async () => {
    const { getByLabelText, getByText, queryByText } = render(HarnessAiSidebar, {
      props: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        draftContent: harnessContent,
        providers: [
          ...providerFixtures,
          {
            ...providerFixtures[0],
            id: 'provider-remote',
            machine_id: 'machine-2',
            machine_name: 'builder-01',
            machine_host: '10.0.0.20',
            machine_ssh_user: 'openase',
            machine_workspace_root: '/srv/workspace',
            name: 'Codex Remote',
            capabilities: {
              ephemeral_chat: { state: 'available', reason: null },
              harness_ai: { state: 'unsupported', reason: 'remote_machine_not_supported' },
              skill_ai: { state: 'unsupported', reason: 'remote_machine_not_supported' },
            },
          },
        ],
      },
    })

    expect(getByText('gpt-5.4')).toBeTruthy()

    const trigger = getByLabelText('Chat model')
    await fireEvent.pointerDown(trigger)
    await fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    await vi.runOnlyPendingTimersAsync()

    expect(getByText('Codex · codex-app-server')).toBeTruthy()
    expect(queryByText('Codex Remote · codex-app-server')).toBeNull()
  })
})
