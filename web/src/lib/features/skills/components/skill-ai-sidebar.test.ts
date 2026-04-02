/* eslint-disable max-lines */
import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { closeSkillRefinementSession, streamSkillRefinement } = vi.hoisted(() => ({
  closeSkillRefinementSession: vi.fn(),
  streamSkillRefinement: vi.fn(),
}))

vi.mock('$lib/api/skill-refinement', () => ({
  closeSkillRefinementSession,
  streamSkillRefinement,
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

const fileContent = [
  '---',
  'name: "deploy"',
  'description: "Deploy safely"',
  '---',
  '',
  'Use safe steps.',
].join('\n')

const verifiedContent = [
  '---',
  'name: "deploy"',
  'description: "Deploy safely"',
  '---',
  '',
  'Use safe steps.',
  '',
  'Verify rollback steps before production deploys.',
].join('\n')

const fileFixtures: SkillFile[] = [
  {
    path: 'SKILL.md',
    file_kind: 'entrypoint',
    media_type: 'text/markdown; charset=utf-8',
    encoding: 'utf8',
    is_executable: false,
    size_bytes: fileContent.length,
    sha256: 'sha-entry',
    content: fileContent,
    content_base64: 'ignored',
  },
  {
    path: 'scripts/redeploy.sh',
    file_kind: 'script',
    media_type: 'text/x-shellscript; charset=utf-8',
    encoding: 'utf8',
    is_executable: true,
    size_bytes: 29,
    sha256: 'sha-script',
    content: '#!/usr/bin/env bash\necho old\n',
    content_base64: 'ignored',
  },
]

describe('SkillAiSidebar', () => {
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

    expect(getByText('Codex · codex-app-server')).toBeTruthy()
    expect(queryByText('Claude · claude-code-cli')).toBeNull()
  })

  it('transitions from idle to running to verified and applies the verified bundle', async () => {
    streamSkillRefinement.mockImplementation(async (request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-1',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-1/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-1',
          phase: 'editing',
          attempt: 1,
          message: 'Codex is editing the draft bundle.',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-1',
          phase: 'testing',
          attempt: 1,
          message: 'Codex is running verification commands.',
        },
      })
      await new Promise((resolve) => setTimeout(resolve, 0))
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-1',
          status: 'verified',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-1/workspace',
          providerId: 'provider-1',
          providerName: 'Codex',
          providerThreadId: 'thread-1',
          providerTurnId: 'turn-1',
          attempts: 1,
          transcriptSummary: 'Bundle verified after tightening the deploy instructions.',
          commandOutputSummary: 'bash -n scripts/redeploy.sh\n./scripts/redeploy.sh',
          candidateBundleHash: 'bundle-hash-1',
          candidateFiles: [
            {
              ...fileFixtures[0],
              content: verifiedContent,
              size_bytes: verifiedContent.length,
              sha256: 'sha-entry-verified',
            },
            fileFixtures[1],
          ],
        },
      })

      expect(request).toMatchObject({
        projectId: 'project-1',
        skillId: 'skill-1',
        providerId: 'provider-1',
        message: 'Make the deploy skill safer.',
      })
    })

    const appliedBundles: SkillFile[][] = []

    const { getByPlaceholderText, getByRole, findByText } = render(SkillAiSidebar, {
      props: {
        projectId: 'project-1',
        providers: providerFixtures,
        skillId: 'skill-1',
        files: fileFixtures,
        onApplySuggestion: (bundle: SkillFile[]) => appliedBundles.push(bundle),
      },
    })

    await findByText('Idle')

    const prompt = getByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, { target: { value: 'Make the deploy skill safer.' } })
    await fireEvent.click(getByRole('button', { name: 'Fix and verify' }))

    await findByText('Testing')
    await findByText('Verified')
    await waitFor(() => {
      expect(
        document.body.textContent?.includes(
          'Bundle verified after tightening the deploy instructions.',
        ),
      ).toBe(true)
    })
    await fireEvent.click(await findByText('Apply All'))

    expect(appliedBundles).toHaveLength(1)
    expect(appliedBundles[0][0]?.content).toBe(verifiedContent)
  })

  it('transitions from idle to running to blocked and surfaces failure evidence', async () => {
    streamSkillRefinement.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-2',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-2/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-2',
          phase: 'editing',
          attempt: 1,
          message: 'Codex is editing the draft bundle.',
        },
      })
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-2',
          status: 'blocked',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-2/workspace',
          providerId: 'provider-1',
          providerName: 'Codex',
          attempts: 2,
          transcriptSummary: 'Codex could not make the script pass shell validation.',
          commandOutputSummary: 'bash -n scripts/redeploy.sh\nscripts/redeploy.sh: syntax error',
          failureReason: 'shell script still has a syntax error near line 4',
          candidateFiles: fileFixtures,
        },
      })
    })

    const { getByPlaceholderText, getByRole, findByText, queryByText } = render(SkillAiSidebar, {
      props: {
        projectId: 'project-1',
        providers: providerFixtures,
        skillId: 'skill-1',
        files: fileFixtures,
      },
    })

    const prompt = getByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, { target: { value: 'Fix the broken deploy script.' } })
    await fireEvent.click(getByRole('button', { name: 'Fix and verify' }))

    await findByText('Blocked')
    await waitFor(() => {
      expect(
        document.body.textContent?.includes('shell script still has a syntax error near line 4'),
      ).toBe(true)
    })
    expect(queryByText('Apply All')).toBeNull()
  })
})
