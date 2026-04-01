import { describe, expect, it, vi, afterEach } from 'vitest'

const {
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
  saveWorkflowHarness,
  unbindWorkflowSkills,
  updateProject,
  updateTicket,
  updateTicketComment,
  updateWorkflow,
} = vi.hoisted(() => ({
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
  saveWorkflowHarness: vi.fn(),
  unbindWorkflowSkills: vi.fn(),
  updateProject: vi.fn(),
  updateTicket: vi.fn(),
  updateTicketComment: vi.fn(),
  updateWorkflow: vi.fn(),
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
  saveWorkflowHarness,
  unbindWorkflowSkills,
  updateProject,
  updateTicket,
  updateTicketComment,
  updateWorkflow,
}))

import type { ChatActionProposalPayload } from '$lib/api/chat'
import { executeActionProposal } from './action-proposal-executor'

describe('executeActionProposal', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('executes supported actions with parsed request bodies', async () => {
    createTicket.mockResolvedValue({ ticket: { id: 'ticket-1' } })

    const proposal: ChatActionProposalPayload = {
      type: 'action_proposal',
      summary: 'Create a ticket',
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
    }

    const results = await executeActionProposal(proposal)

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
    expect(results).toEqual([
      expect.objectContaining({
        ok: true,
        summary: 'POST /api/v1/projects/project-1/tickets succeeded.',
      }),
    ])
  })

  it('reports unsupported actions without throwing', async () => {
    const proposal: ChatActionProposalPayload = {
      type: 'action_proposal',
      summary: 'Unsupported change',
      actions: [
        {
          method: 'POST',
          path: '/api/v1/projects/project-1/repos/import-github',
          body: {
            name: 'repo-1',
          },
        },
      ],
    }

    const results = await executeActionProposal(proposal)

    expect(results).toEqual([
      expect.objectContaining({
        ok: false,
        summary:
          'POST /api/v1/projects/project-1/repos/import-github is not supported by the chat executor yet.',
      }),
    ])
  })

  it('turns invalid action bodies into per-action failures instead of aborting the batch', async () => {
    const proposal: ChatActionProposalPayload = {
      type: 'action_proposal',
      summary: 'Broken action payload',
      actions: [
        {
          method: 'POST',
          path: '/api/v1/projects/project-1/tickets',
          body: {
            title: '',
          },
        },
      ],
    }

    await expect(executeActionProposal(proposal)).resolves.toEqual([
      expect.objectContaining({
        ok: false,
        summary: 'POST /api/v1/projects/project-1/tickets failed.',
        detail: 'Proposed action field title must be a non-empty string.',
      }),
    ])
    expect(createTicket).not.toHaveBeenCalled()
  })
})
