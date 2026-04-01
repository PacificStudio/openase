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

describe('EphemeralChatPanel actions and lifecycle', () => {
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

  it('confirms and executes an action proposal', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: 'session-ap-1' },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'action_proposal',
          summary: 'Create 1 child ticket',
          actions: [
            {
              method: 'POST',
              path: '/api/v1/projects/project-1/tickets',
              body: {
                title: 'Implement child ticket',
                priority: 'high',
              },
            },
          ],
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
    createTicket.mockResolvedValue({ ticket: { id: 'ticket-1' } })

    const { getByPlaceholderText, getByRole, findByText } = render(EphemeralChatPanel, {
      props: {
        source: 'ticket_detail',
        context: { projectId: 'project-1', ticketId: 'ticket-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask…',
      },
    })

    const prompt = getByPlaceholderText('Ask…')
    await sendMessage(prompt, 'Split this into a child ticket.')

    await findByText('Create 1 child ticket')
    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(createTicket).toHaveBeenCalledWith('project-1', {
        title: 'Implement child ticket',
        description: undefined,
        status_id: undefined,
        priority: 'high',
        type: undefined,
        workflow_id: undefined,
        created_by: undefined,
        parent_ticket_id: undefined,
        external_ref: undefined,
        budget_usd: undefined,
      })
    })
    expect(await findByText('POST /api/v1/projects/project-1/tickets succeeded.')).toBeTruthy()
  })

  it('resets the conversation by closing the active session and clearing the transcript', async () => {
    let turnCount = 0

    streamChatTurn.mockImplementation(async (request, handlers) => {
      turnCount += 1

      expect(request.sessionId).toBeUndefined()

      handlers.onEvent({
        kind: 'session',
        payload: { sessionId: `session-reset-${turnCount}` },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: turnCount === 1 ? 'First reply before reset.' : 'Second reply after reset.',
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: `session-reset-${turnCount}`,
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })
    closeChatSession.mockResolvedValue(undefined)

    const { getByPlaceholderText, getByRole, findByText, queryByText } = render(
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
    await sendMessage(prompt, 'Explain the failure.')
    expect(await findByText('First reply before reset.')).toBeTruthy()

    await fireEvent.click(getByRole('button', { name: 'Reset conversation' }))

    await waitFor(() => {
      expect(closeChatSession).toHaveBeenCalledWith('session-reset-1')
    })
    await waitFor(() => {
      expect(queryByText('Explain the failure.')).toBeNull()
      expect(queryByText('First reply before reset.')).toBeNull()
    })

    await sendMessage(prompt, 'Start over with a fresh answer.')

    expect(await findByText('Second reply after reset.')).toBeTruthy()
    expect(streamChatTurn.mock.calls[1][0]).toMatchObject({
      providerId: 'provider-1',
      sessionId: undefined,
    })
  })
})
