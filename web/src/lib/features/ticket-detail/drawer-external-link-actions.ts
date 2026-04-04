import { addTicketExternalLink, deleteTicketExternalLink } from '$lib/api/openase'
import { buildAddExternalLinkMutation } from './mutation-builders'
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

  const mutation = buildAddExternalLinkMutation(draft)
  if (!mutation.ok) {
    drawerState.setMutationError(mutation.error)
    return false
  }

  return await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.creatingExternalLink = true
    },
    finish: () => {
      drawerState.creatingExternalLink = false
    },
    optimisticUpdate: mutation.value.optimisticUpdate,
    mutate: () => addTicketExternalLink(ticketId, mutation.value.body),
    successMessage: mutation.value.successMessage,
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
