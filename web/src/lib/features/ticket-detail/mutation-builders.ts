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
    return { ok: false, error: 'That relationship already exists on this ticket.' }
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
            stage: 'unstarted',
          },
        ],
      }),
      successMessage: 'Relationship added.',
    },
  }
}

export function buildAddExternalLinkMutation(draft: {
  type: string
  url: string
  externalId: string
  title: string
  status: string
  relation: string
}): ParseResult<{
  body: {
    type: string
    url: string
    external_id: string
    title: string | null
    status: string | null
    relation: string | null
  }
  optimisticUpdate: (ticket: TicketDetail) => TicketDetail
  successMessage: string
}> {
  const type = parseRequiredText(draft.type, 'Link type is required.', (value) =>
    value.toLowerCase(),
  )
  if (!type.ok) return type

  const url = parseRequiredText(draft.url, 'Link URL is required.')
  if (!url.ok) return url

  const externalId = parseRequiredText(draft.externalId, 'External ID is required.')
  if (!externalId.ok) return externalId

  const title = normalizeOptionalText(draft.title)
  const status = normalizeOptionalText(draft.status)
  const relation = normalizeOptionalText(draft.relation) ?? 'references'

  return {
    ok: true,
    value: {
      body: {
        type: type.value,
        url: url.value,
        external_id: externalId.value,
        title,
        status,
        relation,
      },
      optimisticUpdate: (ticket) => ({
        ...ticket,
        externalLinks: [
          ...ticket.externalLinks,
          buildOptimisticExternalLink({
            type: type.value,
            url: url.value,
            externalId: externalId.value,
            title,
            status,
            relation,
          }),
        ],
      }),
      successMessage: `External link added for ${externalId.value}.`,
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
      successMessage: `Removed relationship ${dependency.identifier}.`,
    },
  }
}

function normalizeOptionalText(value: string) {
  const normalized = value.trim()
  return normalized ? normalized : null
}

function parseRequiredText(
  value: string,
  error: string,
  transform: (normalized: string) => string = (normalized) => normalized,
): ParseResult<string> {
  const normalized = value.trim()
  if (!normalized) {
    return { ok: false, error }
  }

  return { ok: true, value: transform(normalized) }
}

function buildOptimisticExternalLink({
  type,
  url,
  externalId,
  title,
  status,
  relation,
}: {
  type: string
  url: string
  externalId: string
  title: string | null
  status: string | null
  relation: string
}) {
  return {
    id: `pending-${Date.now()}`,
    type,
    url,
    externalId,
    title: title ?? undefined,
    status: status ?? undefined,
    relation,
  }
}
