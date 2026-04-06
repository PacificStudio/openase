import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeChatSession,
  streamChatTurn,
  addTicketDependency,
  addTicketExternalLink,
  bindWorkflowSkills,
  createTicket,
  createTicketComment,
  createWorkflow,
  deleteTicketComment,
  deleteTicketDependency,
  deleteTicketExternalLink,
  deleteWorkflow,
  listProviders,
  saveWorkflowHarness,
  unbindWorkflowSkills,
  updateProject,
  updateTicket,
  updateTicketComment,
  updateWorkflow,
} = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
  addTicketDependency: vi.fn(),
  addTicketExternalLink: vi.fn(),
  bindWorkflowSkills: vi.fn(),
  createTicket: vi.fn(),
  createTicketComment: vi.fn(),
  createWorkflow: vi.fn(),
  deleteTicketComment: vi.fn(),
  deleteTicketDependency: vi.fn(),
  deleteTicketExternalLink: vi.fn(),
  deleteWorkflow: vi.fn(),
  listProviders: vi.fn(),
  saveWorkflowHarness: vi.fn(),
  unbindWorkflowSkills: vi.fn(),
  updateProject: vi.fn(),
  updateTicket: vi.fn(),
  updateTicketComment: vi.fn(),
  updateWorkflow: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  addTicketDependency,
  addTicketExternalLink,
  bindWorkflowSkills,
  createTicket,
  createTicketComment,
  createWorkflow,
  deleteTicketComment,
  deleteTicketDependency,
  deleteTicketExternalLink,
  deleteWorkflow,
  listProviders,
  saveWorkflowHarness,
  unbindWorkflowSkills,
  updateProject,
  updateTicket,
  updateTicketComment,
  updateWorkflow,
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
  {
    id: 'provider-2',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Claude',
    adapter_type: 'claude-code',
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
    cli_command: 'claude',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'claude-sonnet-4',
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

describe('EphemeralChatPanel', () => {
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

  it('completes a two-turn ticket_detail conversation with session reuse', async () => {
    let turnCount = 0

    streamChatTurn.mockImplementation(async (request, handlers) => {
      turnCount += 1

      if (turnCount === 1) {
        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-ticket-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content: 'This ticket failed because the test assertion timed out after 30 seconds.',
          },
        })
        handlers.onEvent({
          kind: 'done',
          payload: {
            sessionId: 'session-ticket-1',
            turnsUsed: 1,
            turnsRemaining: 9,
          },
        })
      } else {
        expect(request.sessionId).toBe('session-ticket-1')

        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-ticket-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content:
              'You should increase the timeout to 60 seconds and add a retry with exponential backoff.',
          },
        })
        handlers.onEvent({
          kind: 'done',
          payload: {
            sessionId: 'session-ticket-1',
            turnsUsed: 2,
            turnsRemaining: 8,
          },
        })
      }
    })

    const { getByPlaceholderText, findByText } = render(EphemeralChatPanel, {
      props: {
        source: 'ticket_detail',
        context: { projectId: 'project-1', ticketId: 'ticket-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        title: 'Ticket AI',
        placeholder: 'Ask about failures…',
      },
    })

    const prompt = getByPlaceholderText('Ask about failures…')

    // Turn 1
    await sendMessage(prompt, 'Why did this ticket fail?')

    expect(
      await findByText('This ticket failed because the test assertion timed out after 30 seconds.'),
    ).toBeTruthy()

    expect(streamChatTurn).toHaveBeenCalledTimes(1)
    expect(streamChatTurn.mock.calls[0][0]).toMatchObject({
      message: 'Why did this ticket fail?',
      source: 'ticket_detail',
      providerId: 'provider-1',
      sessionId: undefined,
      context: { projectId: 'project-1', ticketId: 'ticket-1' },
    })

    // Turn 2 — follow-up should reuse the session
    await sendMessage(prompt, 'How should I fix it?')

    expect(
      await findByText(
        'You should increase the timeout to 60 seconds and add a retry with exponential backoff.',
      ),
    ).toBeTruthy()

    expect(streamChatTurn).toHaveBeenCalledTimes(2)
    expect(streamChatTurn.mock.calls[1][0]).toMatchObject({
      message: 'How should I fix it?',
      source: 'ticket_detail',
      providerId: 'provider-1',
      sessionId: 'session-ticket-1',
      context: { projectId: 'project-1', ticketId: 'ticket-1' },
    })

    // Both user messages visible in transcript
    expect(await findByText('Why did this ticket fail?')).toBeTruthy()
    expect(await findByText('How should I fix it?')).toBeTruthy()
  })
})
