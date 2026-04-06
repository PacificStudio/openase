import { afterEach, describe, expect, it, vi } from 'vitest'

const { streamChatTurn } = vi.hoisted(() => ({
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession: vi.fn(),
  streamChatTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

import type { AgentProvider } from '$lib/api/contracts'
import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'

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
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
    max_parallel_runs: 1,
  },
]

describe('createEphemeralChatSessionController diff handling', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('keeps structured diff entries distinct from render-only assistant text', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: 'I tightened the test guidance.',
        },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'diff',
          file: 'harness content',
          hunks: [
            {
              oldStart: 1,
              oldLines: 1,
              newStart: 1,
              newLines: 2,
              lines: [
                { op: 'context', text: '---' },
                { op: 'add', text: 'new line' },
              ],
            },
          ],
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-1',
          turnsUsed: 1,
        },
      })
    })

    const controller = createEphemeralChatSessionController({
      getSource: () => 'project_sidebar',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'Tighten the harness.',
      context: {
        projectId: 'project-1',
      },
    })

    expect(controller.entries).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: 'assistant',
          kind: 'text',
          content: 'I tightened the test guidance.',
        }),
        expect.objectContaining({
          role: 'assistant',
          kind: 'diff',
          diff: expect.objectContaining({
            file: 'harness content',
          }),
        }),
      ]),
    )
  })
})
