import type { TicketDetail, TicketReferenceOption, TicketStatusOption } from './types'
import {
  dependencyRelationOptions,
  type DependencyDraft,
  type ParseResult,
  type TicketFieldDraft,
} from './mutation-shared'

export function buildFieldMutation(
  currentTicket: TicketDetail,
  availableStatuses: TicketStatusOption[],
  draft: TicketFieldDraft,
): ParseResult<{
  body: { title?: string; description?: string; status_id?: string }
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const title = draft.title.trim()
  if (!title) {
    return { ok: false, error: 'Title is required.' }
  }

  const status = availableStatuses.find((item) => item.id === draft.statusId)
  if (!status) {
    return { ok: false, error: 'Select a valid ticket status.' }
  }

  const description = draft.description.trim()
  const body: { title?: string; description?: string; status_id?: string } = {}

  if (title !== currentTicket.title) body.title = title
  if (description !== currentTicket.description) body.description = description
  if (status.id !== currentTicket.status.id) body.status_id = status.id

  return {
    ok: true,
    value: {
      body,
      optimisticUpdate: (ticket) => ({
        ...ticket,
        title,
        description,
        status,
      }),
      successMessage: 'Ticket fields saved.',
    },
  }
}

export function buildAddDependencyMutation(
  currentTicket: TicketDetail,
  availableTickets: TicketReferenceOption[],
  draft: DependencyDraft,
): ParseResult<{
  body: { target_ticket_id: string; type: string }
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const target = availableTickets.find((item) => item.id === draft.targetTicketId)
  if (!target) {
    return { ok: false, error: 'Select a valid dependency target ticket.' }
  }
  if (
    currentTicket.dependencies.some((dependency) => dependency.targetId === draft.targetTicketId)
  ) {
    return { ok: false, error: 'That dependency already exists on this ticket.' }
  }
  if (!dependencyRelationOptions.some((option) => option.value === draft.relation)) {
    return { ok: false, error: 'Select a valid dependency relation.' }
  }

  return {
    ok: true,
    value: {
      body: {
        target_ticket_id: target.id,
        type: draft.relation,
      },
      optimisticUpdate: (ticket) => ({
        ...ticket,
        dependencies: [
          ...ticket.dependencies,
          {
            id: `pending-${target.id}`,
            targetId: target.id,
            identifier: target.identifier,
            title: target.title,
            relation: draft.relation,
          },
        ],
      }),
      successMessage: 'Dependency added.',
    },
  }
}

export function buildDeleteDependencyMutation(
  currentTicket: TicketDetail,
  dependencyId: string,
): ParseResult<{
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const dependency = currentTicket.dependencies.find((item) => item.id === dependencyId)
  if (!dependency) {
    return { ok: false, error: 'Dependency no longer exists in the current ticket state.' }
  }

  return {
    ok: true,
    value: {
      optimisticUpdate: (ticket) => ({
        ...ticket,
        dependencies: ticket.dependencies.filter((item) => item.id !== dependencyId),
      }),
      successMessage: `Removed dependency ${dependency.identifier}.`,
    },
  }
}
