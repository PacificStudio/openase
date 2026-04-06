import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

import type { AgentProvider } from '$lib/api/contracts'
import EphemeralChatPanel from './ephemeral-chat-panel.svelte'

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
    pricing_config: {},
  },
]

async function sendMessage(prompt: HTMLElement, message: string) {
  await fireEvent.input(prompt, { target: { value: message } })
  await fireEvent.keyDown(prompt, { key: 'Enter' })
}

describe('EphemeralChatPanel legacy proposal handling', () => {
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

  it('renders proposal stream payloads as plain assistant text without confirm controls', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: 'session-ap-1' },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: 'Create 1 child ticket',
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-ap-1',
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })

    const { getByPlaceholderText, findByText, queryByRole } = render(EphemeralChatPanel, {
      props: {
        source: 'ticket_detail',
        context: { projectId: 'project-1', ticketId: 'ticket-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask…',
      },
    })

    await sendMessage(getByPlaceholderText('Ask…'), 'Split this into a child ticket.')

    expect(await findByText('Create 1 child ticket')).toBeTruthy()
    expect(queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(queryByRole('button', { name: 'Cancel' })).toBeNull()
  })
})
