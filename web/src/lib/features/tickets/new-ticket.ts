import type { TicketStatus, Workflow } from '$lib/api/contracts'

export const ticketPriorityOptions = ['urgent', 'high', 'medium', 'low'] as const

export type TicketPriorityOption = (typeof ticketPriorityOptions)[number]

export type TicketOption = {
  id: string
  label: string
}

export type NewTicketDraft = {
  title: string
  description: string
  statusId: string
  priority: TicketPriorityOption
  workflowId: string
}

export type NewTicketPayload = {
  title: string
  description?: string
  status_id?: string | null
  priority: TicketPriorityOption
  workflow_id?: string | null
}

type ParsedDraft = { ok: true; payload: NewTicketPayload } | { ok: false; error: string }

export function mapTicketStatusOptions(statuses: TicketStatus[]): TicketOption[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({
      id: status.id,
      label: status.name,
    }))
}

export function mapWorkflowOptions(workflows: Workflow[]): TicketOption[] {
  return workflows
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((workflow) => ({
      id: workflow.id,
      label:
        workflow.name === workflow.type ? workflow.name : `${workflow.name} (${workflow.type})`,
    }))
}

export function createNewTicketDraft(
  statusOptions: TicketOption[],
  workflowOptions: TicketOption[],
): NewTicketDraft {
  return {
    title: '',
    description: '',
    statusId: statusOptions[0]?.id ?? '',
    priority: 'medium',
    workflowId: workflowOptions[0]?.id ?? '',
  }
}

export function parseNewTicketDraft(draft: NewTicketDraft): ParsedDraft {
  const title = draft.title.trim()
  if (!title) {
    return {
      ok: false,
      error: 'Title is required.',
    }
  }

  const description = draft.description.trim()

  return {
    ok: true,
    payload: {
      title,
      description: description || undefined,
      status_id: draft.statusId || null,
      priority: draft.priority,
      workflow_id: draft.workflowId || null,
    },
  }
}
