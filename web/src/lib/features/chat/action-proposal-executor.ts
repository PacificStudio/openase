import { ApiError } from '$lib/api/client'
import type { ChatActionProposalAction, ChatActionProposalPayload } from '$lib/api/chat'
import {
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
} from '$lib/api/openase'
import {
  parseCreateCommentBody,
  parseCreateTicketBody,
  parseCreateWorkflowBody,
  parseDependencyBody,
  parseExternalLinkBody,
  parseHarnessBody,
  parseProjectUpdateBody,
  parseSkillsBody,
  parseUpdateCommentBody,
  parseUpdateTicketBody,
  parseUpdateWorkflowBody,
} from './action-proposal-bodies'
import { matchActionProposalPath } from './action-proposal-paths'

export type ChatActionExecutionResult = {
  actionIndex: number
  action: ChatActionProposalAction
  ok: boolean
  summary: string
  detail?: string
}

type SupportedAction = {
  action: ChatActionProposalAction
  execute: () => Promise<unknown>
}

export async function executeActionProposal(
  proposal: ChatActionProposalPayload,
): Promise<ChatActionExecutionResult[]> {
  const results: ChatActionExecutionResult[] = []

  for (const [index, action] of proposal.actions.entries()) {
    const supportedAction = parseSupportedAction(action)
    if (!supportedAction) {
      results.push({
        actionIndex: index,
        action,
        ok: false,
        summary: `${action.method} ${action.path} is not supported by the chat executor yet.`,
      })
      continue
    }

    try {
      await supportedAction.execute()
      results.push({
        actionIndex: index,
        action,
        ok: true,
        summary: `${action.method} ${action.path} succeeded.`,
      })
    } catch (caughtError) {
      results.push({
        actionIndex: index,
        action,
        ok: false,
        summary: `${action.method} ${action.path} failed.`,
        detail: formatExecutionError(caughtError),
      })
    }
  }

  return results
}

function parseSupportedAction(action: ChatActionProposalAction): SupportedAction | null {
  const path = matchActionProposalPath(action.path)

  if (action.method === 'POST' && path.kind === 'projectTickets') {
    return {
      action,
      execute: () => createTicket(path.projectId, parseCreateTicketBody(action.body)),
    }
  }

  if (action.method === 'PATCH' && path.kind === 'ticket') {
    return {
      action,
      execute: () => updateTicket(path.ticketId, parseUpdateTicketBody(action.body)),
    }
  }

  if (action.method === 'POST' && path.kind === 'ticketComments') {
    return {
      action,
      execute: () => createTicketComment(path.ticketId, parseCreateCommentBody(action.body)),
    }
  }

  if (action.method === 'PATCH' && path.kind === 'ticketComment') {
    return {
      action,
      execute: () =>
        updateTicketComment(path.ticketId, path.commentId, parseUpdateCommentBody(action.body)),
    }
  }

  if (action.method === 'DELETE' && path.kind === 'ticketComment') {
    return {
      action,
      execute: () => deleteTicketComment(path.ticketId, path.commentId),
    }
  }

  if (action.method === 'POST' && path.kind === 'ticketDependencies') {
    return {
      action,
      execute: () => addTicketDependency(path.ticketId, parseDependencyBody(action.body)),
    }
  }

  if (action.method === 'DELETE' && path.kind === 'ticketDependency') {
    return {
      action,
      execute: () => deleteTicketDependency(path.ticketId, path.dependencyId),
    }
  }

  if (action.method === 'POST' && path.kind === 'ticketExternalLinks') {
    return {
      action,
      execute: () => addTicketExternalLink(path.ticketId, parseExternalLinkBody(action.body)),
    }
  }

  if (action.method === 'DELETE' && path.kind === 'ticketExternalLink') {
    return {
      action,
      execute: () => deleteTicketExternalLink(path.ticketId, path.externalLinkId),
    }
  }

  if (action.method === 'PATCH' && path.kind === 'project') {
    return {
      action,
      execute: () => updateProject(path.projectId, parseProjectUpdateBody(action.body)),
    }
  }

  if (action.method === 'POST' && path.kind === 'projectWorkflows') {
    return {
      action,
      execute: () => createWorkflow(path.projectId, parseCreateWorkflowBody(action.body)),
    }
  }

  if (action.method === 'PATCH' && path.kind === 'workflow') {
    return {
      action,
      execute: () => updateWorkflow(path.workflowId, parseUpdateWorkflowBody(action.body)),
    }
  }

  if (action.method === 'DELETE' && path.kind === 'workflow') {
    return {
      action,
      execute: () => deleteWorkflow(path.workflowId),
    }
  }

  if (action.method === 'PUT' && path.kind === 'workflowHarness') {
    return {
      action,
      execute: () => saveWorkflowHarness(path.workflowId, parseHarnessBody(action.body)),
    }
  }

  if (action.method === 'POST' && path.kind === 'workflowBindSkills') {
    return {
      action,
      execute: () => bindWorkflowSkills(path.workflowId, parseSkillsBody(action.body)),
    }
  }

  if (action.method === 'POST' && path.kind === 'workflowUnbindSkills') {
    return {
      action,
      execute: () => unbindWorkflowSkills(path.workflowId, parseSkillsBody(action.body)),
    }
  }

  return null
}

function formatExecutionError(error: unknown) {
  if (error instanceof ApiError) {
    return error.detail
  }

  if (error instanceof Error) {
    return error.message
  }

  return 'Unknown action execution error.'
}
