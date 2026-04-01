import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
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

async function chooseNextProvider(trigger: HTMLElement) {
  await fireEvent.pointerDown(trigger)
  await fireEvent.keyDown(trigger, { key: 'ArrowDown' })
  await fireEvent.keyDown(trigger, { key: 'Enter' })
}

describe('EphemeralChatPanel provider switching', () => {
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

  it('switches providers by closing the old session and starting a fresh one', async () => {
    let turnCount = 0

    streamChatTurn.mockImplementation(async (request, handlers) => {
      turnCount += 1

      if (turnCount === 1) {
        expect(request.providerId).toBe('provider-1')
        expect(request.sessionId).toBeUndefined()

        handlers.onEvent({
          kind: 'session',
          payload: { sessionId: 'session-provider-1' },
        })
        handlers.onEvent({
          kind: 'message',
          payload: {
            type: 'text',
            content: 'Reply from the first provider.',
          },
        })
        handlers.onEvent({
          kind: 'done',
          payload: {
            sessionId: 'session-provider-1',
            turnsUsed: 1,
            turnsRemaining: 9,
          },
        })
        return
      }

      expect(request.providerId).toBe('provider-2')
      expect(request.sessionId).toBeUndefined()

      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: 'session-provider-2' },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: 'Reply from the second provider.',
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-provider-2',
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })
    closeChatSession.mockResolvedValue(undefined)

    const { getByLabelText, getByPlaceholderText, findByText, queryByText } = render(
      EphemeralChatPanel,
      {
        props: {
          source: 'ticket_detail',
          context: { projectId: 'project-1', ticketId: 'ticket-1' },
          providers: providerFixtures,
          defaultProviderId: 'provider-1',
          placeholder: 'Ask…',
        },
      },
    )

    const prompt = getByPlaceholderText('Ask…')
    await sendMessage(prompt, 'Use the first provider.')
    expect(await findByText('Reply from the first provider.')).toBeTruthy()

    await chooseNextProvider(getByLabelText('Chat model'))

    await waitFor(() => {
      expect(closeChatSession).toHaveBeenCalledWith('session-provider-1')
    })
    await waitFor(() => {
      expect(queryByText('Reply from the first provider.')).toBeNull()
    })

    await sendMessage(prompt, 'Use the second provider.')

    expect(await findByText('Reply from the second provider.')).toBeTruthy()
    expect(streamChatTurn.mock.calls[1][0]).toMatchObject({
      providerId: 'provider-2',
      sessionId: undefined,
    })
  })
})
