import { addTicketExternalLink, deleteTicketExternalLink } from '$lib/api/openase'
import { runTicketDrawerMutation } from './drawer-mutation'
import type { TicketDetail, TicketExternalLinkDraft } from './types'

type DrawerMutationBase = Omit<
  Parameters<typeof runTicketDrawerMutation>[0],
  'start' | 'finish' | 'optimisticUpdate' | 'mutate' | 'successMessage'
>

type ExternalLinkDrawerState = {
  ticket: TicketDetail | null
  creatingExternalLink: boolean
  deletingExternalLinkId: string | null
  setMutationError: (message: string) => void
}

export async function handleCreateExternalLinkAction({
  ticketId,
  drawerState,
  draft,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: ExternalLinkDrawerState
  draft: TicketExternalLinkDraft
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return false

  return await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.creatingExternalLink = true
    },
    finish: () => {
      drawerState.creatingExternalLink = false
    },
    optimisticUpdate: (currentTicket) => ({
      ...currentTicket,
      externalLinks: [
        ...currentTicket.externalLinks,
        {
          id: `pending-${Date.now()}`,
          type: draft.type,
          url: draft.url,
          externalId: draft.externalId,
          title: draft.title || undefined,
          status: draft.status || undefined,
          relation: draft.relation || 'references',
        },
      ],
    }),
    mutate: () =>
      addTicketExternalLink(ticketId, {
        type: draft.type,
        url: draft.url,
        external_id: draft.externalId,
        title: draft.title || null,
        status: draft.status || null,
        relation: draft.relation || null,
      }),
    successMessage: 'External link added.',
  })
}

export async function handleDeleteExternalLinkAction({
  ticketId,
  drawerState,
  linkId,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: ExternalLinkDrawerState
  linkId: string
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return

  const link = ticket.externalLinks.find((item) => item.id === linkId)
  if (!link) {
    drawerState.setMutationError('External link no longer exists in the current ticket state.')
    return
  }

  await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.deletingExternalLinkId = linkId
    },
    finish: () => {
      drawerState.deletingExternalLinkId = null
    },
    optimisticUpdate: (currentTicket) => ({
      ...currentTicket,
      externalLinks: currentTicket.externalLinks.filter((item) => item.id !== linkId),
    }),
    mutate: () => deleteTicketExternalLink(ticketId, linkId),
    successMessage: `Removed external link ${link.externalId}.`,
  })
}
