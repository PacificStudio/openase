import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('$lib/api/skill-refinement', () => ({
  closeSkillRefinementSession: vi.fn(),
  streamSkillRefinement: vi.fn(),
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

import type { AgentProvider, SkillFile } from '$lib/api/contracts'
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

const fileFixtures: SkillFile[] = [
  {
    path: 'SKILL.md',
    file_kind: 'entrypoint',
    media_type: 'text/markdown; charset=utf-8',
    encoding: 'utf8',
    is_executable: false,
    size_bytes: 12,
    sha256: 'sha-entry',
    content: 'Use safe steps.',
    content_base64: 'ignored',
  },
]

describe('SkillAiSidebar provider picker', () => {
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

  it('shows only Codex-capable providers in the Skill AI picker', async () => {
    const { getByLabelText, getByText, queryByText } = render(SkillAiSidebar, {
      props: {
        projectId: 'project-1',
        skillId: 'skill-1',
        files: fileFixtures,
        providers: [
          ...providerFixtures,
          {
            ...providerFixtures[0],
            id: 'provider-claude',
            name: 'Claude',
            adapter_type: 'claude-code-cli',
            model_name: 'claude-sonnet-4',
            cli_command: 'claude',
            capabilities: {
              ephemeral_chat: { state: 'available', reason: null },
              harness_ai: { state: 'available', reason: null },
              skill_ai: { state: 'unsupported', reason: 'skill_ai_requires_codex' },
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
    expect(queryByText('Claude · claude-code-cli')).toBeNull()
  })
})
