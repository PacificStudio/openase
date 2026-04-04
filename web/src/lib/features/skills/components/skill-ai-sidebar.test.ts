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
      handlers.onEvent({
        kind: 'session_anchor',
        payload: {
          providerThreadId: 'thread-1',
          providerTurnId: 'turn-1',
          providerAnchorId: 'thread-1',
          providerAnchorKind: 'thread',
          providerTurnSupported: true,
        },
      })
      handlers.onEvent({
        kind: 'thread_status',
        payload: {
          threadId: 'thread-1',
          status: 'active',
          activeFlags: ['running'],
          entryId: 'entry-thread-1',
        },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'task_progress',
          raw: {
            stream: 'command',
            command: 'bash -n scripts/redeploy.sh',
            text: 'bash -n scripts/redeploy.sh\n./scripts/redeploy.sh',
            snapshot: true,
          },
        },
      })
      handlers.onEvent({
        kind: 'plan_updated',
        payload: {
          threadId: 'thread-1',
          turnId: 'turn-1',
          explanation: 'Inspect and verify the skill bundle.',
          plan: [
            { step: 'Inspect', status: 'completed' },
            { step: 'Verify', status: 'completed' },
          ],
          entryId: 'entry-plan-1',
        },
      })
      handlers.onEvent({
        kind: 'reasoning_updated',
        payload: {
          threadId: 'thread-1',
          turnId: 'turn-1',
          itemId: 'item-1',
          kind: 'summary_text_delta',
          delta: 'Checking the verification transcript.',
          entryId: 'entry-reasoning-1',
        },
      })
      handlers.onEvent({
        kind: 'interrupt_requested',
        payload: {
          requestId: 'req-1',
          kind: 'command_execution',
          payload: { command: 'git status' },
          options: [{ id: 'approve_once', label: 'Approve once' }],
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
    await findByText('Provider Thread')
    await findByText('thread-1')
    await findByText('Command approval required')
    await findByText('Ran `bash -n scripts/redeploy.sh`')
    await fireEvent.click(await findByText('Ran `bash -n scripts/redeploy.sh`'))
    await findByText('Codex thread status')
    await findByText('Plan updated')
    await findByText('Reasoning update')
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

  it('renders Claude session anchors and requires_action transcript events', async () => {
    const claudeProviders: AgentProvider[] = [
      {
        ...providerFixtures[0],
        id: 'provider-claude',
        name: 'Claude Code',
        adapter_type: 'claude-code',
        cli_command: 'claude',
        model_name: 'claude-opus-4-6',
      },
    ]

    streamSkillRefinement.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-claude-1',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-claude-1/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-claude-1',
          phase: 'testing',
          attempt: 1,
          message: 'Claude is resuming the draft bundle session.',
        },
      })
      handlers.onEvent({
        kind: 'session_anchor',
        payload: {
          providerAnchorId: 'claude-session-1',
          providerAnchorKind: 'session',
          providerTurnSupported: false,
        },
      })
      handlers.onEvent({
        kind: 'session_state',
        payload: {
          status: 'requires_action',
          activeFlags: ['waiting_for_input'],
          detail: 'Claude is waiting for an approval decision.',
          entryId: 'entry-session-state-1',
        },
      })
      handlers.onEvent({
        kind: 'interrupt_requested',
        payload: {
          requestId: 'req-claude-1',
          kind: 'user_input',
          payload: {
            question: 'Proceed with updating the runbook?',
          },
          options: [
            { id: 'yes', label: 'Yes' },
            { id: 'no', label: 'No' },
          ],
        },
      })
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-claude-1',
          status: 'verified',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-claude-1/workspace',
          providerId: 'provider-claude',
          providerName: 'Claude Code',
          attempts: 1,
          transcriptSummary: 'Claude updated the runbook after approval.',
          commandOutputSummary: '',
          candidateBundleHash: 'bundle-hash-claude-1',
          candidateFiles: fileFixtures,
        },
      })
    })

    const { getByPlaceholderText, getByRole, findAllByText, findByText } = render(SkillAiSidebar, {
      props: {
        projectId: 'project-1',
        providers: claudeProviders,
        skillId: 'skill-1',
        files: fileFixtures,
      },
    })

    const prompt = getByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, {
      target: { value: 'Update the runbook and ask before editing.' },
    })
    await fireEvent.click(getByRole('button', { name: 'Fix and verify' }))

    await findByText('Provider Session')
    await findByText('claude-session-1')
    const systemGroups = await findAllByText('System activity')
    await fireEvent.click(systemGroups[0]!)
    await findByText('Claude session status')
    await findByText('User input required')
    await waitFor(() => {
      expect(
        document.body.textContent?.includes('Claude updated the runbook after approval.'),
      ).toBe(true)
    })
  })

  it('renders diff updates, compaction, and anchor detail cards inside the transcript', async () => {
    streamSkillRefinement.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-3',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-3/workspace',
        },
      })
      handlers.onEvent({
        kind: 'status',
        payload: {
          sessionId: 'session-3',
          phase: 'testing',
          attempt: 1,
          message: 'Codex is replaying the latest diff and status updates.',
        },
      })
      handlers.onEvent({
        kind: 'session_anchor',
        payload: {
          providerThreadId: 'thread-9',
          providerTurnId: 'turn-9',
          providerAnchorId: 'thread-9',
          providerAnchorKind: 'thread',
          providerTurnSupported: true,
        },
      })
      handlers.onEvent({
        kind: 'thread_compacted',
        payload: {
          threadId: 'thread-9',
          turnId: 'turn-9',
          entryId: 'entry-compact-9',
        },
      })
      handlers.onEvent({
        kind: 'diff_updated',
        payload: {
          threadId: 'thread-9',
          turnId: 'turn-9',
          entryId: 'entry-diff-9',
          diff: [
            'diff --git a/SKILL.md b/SKILL.md',
            '--- a/SKILL.md',
            '+++ b/SKILL.md',
            '@@ -1,1 +1,2 @@',
            ' Use safe steps.',
            '+Add a pre-deploy rollback verification checklist.',
          ].join('\n'),
        },
      })
      handlers.onEvent({
        kind: 'result',
        payload: {
          sessionId: 'session-3',
          status: 'verified',
          workspacePath: '/tmp/skill-tests/openase/deploy/session-3/workspace',
          providerId: 'provider-1',
          providerName: 'Codex',
          providerThreadId: 'thread-9',
          providerTurnId: 'turn-9',
          attempts: 1,
          transcriptSummary: 'Codex replayed the diff and compacted the thread.',
          commandOutputSummary: '',
          candidateBundleHash: 'bundle-hash-3',
          candidateFiles: fileFixtures,
        },
      })
    })

    const { getAllByRole, getByPlaceholderText, getByRole, findAllByText, findByText } = render(
      SkillAiSidebar,
      {
        props: {
          projectId: 'project-1',
          providers: providerFixtures,
          skillId: 'skill-1',
          files: fileFixtures,
        },
      },
    )

    const prompt = getByPlaceholderText(
      'Describe what Codex should improve and verify for this draft bundle…',
    )
    await fireEvent.input(prompt, { target: { value: 'Show the latest diff and status details.' } })
    await fireEvent.click(getByRole('button', { name: 'Fix and verify' }))

    await findByText('Provider Thread')
    await findByText('thread-9')
    expect((await findAllByText('SKILL.md')).length).toBeGreaterThan(0)
    await findByText('+Add a pre-deploy rollback verification checklist.')

    for (const button of getAllByRole('button', { name: /System activity/i })) {
      await fireEvent.click(button)
    }
    await findByText('Thread compacted')
    await findByText('Provider thread anchored')

    await fireEvent.click(getByRole('button', { name: /Thread compacted/i }))
    await findByText('Thread thread-9 compacted')

    await fireEvent.click(getByRole('button', { name: /Provider thread anchored/i }))
    await waitFor(() => {
      const pageText = document.body.textContent ?? ''
      expect(pageText.includes('anchor: thread-9')).toBe(true)
      expect(pageText.includes('turn: turn-9')).toBe(true)
      expect(pageText.includes('turn support: yes')).toBe(true)
    })
  })
})
