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
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 4096,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
  },
]

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

  it('renders, confirms, and executes an action proposal through the existing API client', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
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
          sessionId: 'session-1',
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })
    createTicket.mockResolvedValue({
      ticket: {
        id: 'ticket-1',
      },
    })

    const { getByPlaceholderText, getByRole, findByText } = render(EphemeralChatPanel, {
      props: {
        source: 'project_sidebar',
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
      },
    })

    expect(
      await findByText(
        'Session cap: 10 turns / $2.00. The first reply starts a new ephemeral chat session.',
      ),
    ).toBeTruthy()

    const prompt = getByPlaceholderText('Ask a question about this project.')
    await fireEvent.input(prompt, { target: { value: 'Split this work into a child ticket.' } })
    await fireEvent.click(getByRole('button', { name: 'Send' }))

    await findByText('Create 1 child ticket')
    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(createTicket).toHaveBeenCalledWith('project-1', {
        title: 'Implement child ticket',
        priority: 'high',
        description: undefined,
        status_id: undefined,
        type: undefined,
        workflow_id: undefined,
        created_by: undefined,
        parent_ticket_id: undefined,
        external_ref: undefined,
        budget_usd: undefined,
      })
    })

    expect(await findByText('POST /api/v1/projects/project-1/tickets succeeded.')).toBeTruthy()
    expect(await findByText('Executed 1 proposed platform action.')).toBeTruthy()
  })
})
