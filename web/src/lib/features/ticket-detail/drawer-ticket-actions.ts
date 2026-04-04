import {
  addTicketDependency,
  deleteTicketDependency,
  resetTicketWorkspace,
  resumeTicketRetry,
  updateTicket,
} from '$lib/api/openase'
import {
  buildAddDependencyMutation,
  buildDeleteDependencyMutation,
  buildFieldMutation,
} from './mutation-builders'
import { runTicketDrawerMutation } from './drawer-mutation'
import type { DependencyDraft, TicketFieldDraft } from './mutation-shared'
import type { TicketDetail, TicketReferenceOption, TicketStatusOption } from './types'

type DrawerMutationBase = Omit<
  Parameters<typeof runTicketDrawerMutation>[0],
  'start' | 'finish' | 'optimisticUpdate' | 'mutate' | 'successMessage'
>

type TicketMutationDrawerState = {
  ticket: TicketDetail | null
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  savingFields: boolean
  creatingDependency: boolean
  deletingDependencyId: string | null
  resumingRetry: boolean
  resettingWorkspace: boolean
  setMutationError: (message: string) => void
  setMutationNotice: (message: string) => void
}

export async function handleSaveFieldsAction({
  ticketId,
  drawerState,
  draft,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: TicketMutationDrawerState
  draft: TicketFieldDraft
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return

  const mutation = buildFieldMutation(ticket, drawerState.statuses, draft)
  if (!mutation.ok) return drawerState.setMutationError(mutation.error)
  if (Object.keys(mutation.value.body).length === 0) {
    return drawerState.setMutationNotice('No ticket field changes to save.')
  }

  await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.savingFields = true
    },
    finish: () => {
      drawerState.savingFields = false
    },
    optimisticUpdate: mutation.value.optimisticUpdate,
    mutate: () => updateTicket(ticketId, mutation.value.body),
    successMessage: mutation.value.successMessage,
  })
}

export async function handleAddDependencyAction({
  ticketId,
  drawerState,
  draft,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: TicketMutationDrawerState
  draft: DependencyDraft
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return false

  const mutation = buildAddDependencyMutation(ticket, drawerState.dependencyCandidates, draft)
  if (!mutation.ok) {
    drawerState.setMutationError(mutation.error)
    return false
  }

  return await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.creatingDependency = true
    },
    finish: () => {
      drawerState.creatingDependency = false
    },
    optimisticUpdate: mutation.value.optimisticUpdate,
    mutate: () => addTicketDependency(ticketId, mutation.value.body),
    successMessage: mutation.value.successMessage,
  })
}

export async function handleDeleteDependencyAction({
  ticketId,
  drawerState,
  dependencyId,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: TicketMutationDrawerState
  dependencyId: string
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return

  const mutation = buildDeleteDependencyMutation(ticket, dependencyId)
  if (!mutation.ok) return drawerState.setMutationError(mutation.error)

  await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.deletingDependencyId = dependencyId
    },
    finish: () => {
      drawerState.deletingDependencyId = null
    },
    optimisticUpdate: mutation.value.optimisticUpdate,
    mutate: () => deleteTicketDependency(ticketId, dependencyId),
    successMessage: mutation.value.successMessage,
  })
}

export async function handleResumeRetryAction({
  ticketId,
  drawerState,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: TicketMutationDrawerState
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return
  if (!ticket.retryPaused || ticket.pauseReason !== 'repeated_stalls') {
    return drawerState.setMutationError('Only stalled tickets can continue retry from this card.')
  }

  await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.resumingRetry = true
    },
    finish: () => {
      drawerState.resumingRetry = false
    },
    optimisticUpdate: (currentTicket) => ({
      ...currentTicket,
      retryPaused: false,
      pauseReason: undefined,
    }),
    mutate: () => resumeTicketRetry(ticketId),
    successMessage: 'Retry resumed.',
  })
}

export async function handleResetWorkspaceAction({
  ticketId,
  drawerState,
  buildDrawerMutation,
}: {
  ticketId?: string | null
  drawerState: TicketMutationDrawerState
  buildDrawerMutation: (ticket: TicketDetail) => DrawerMutationBase
}) {
  const ticket = drawerState.ticket
  if (!ticket || !ticketId) return
  if (ticket.currentRunId) {
    return drawerState.setMutationError('Workspace can only be reset after the current run stops.')
  }

  await runTicketDrawerMutation({
    ...buildDrawerMutation(ticket),
    start: () => {
      drawerState.resettingWorkspace = true
    },
    finish: () => {
      drawerState.resettingWorkspace = false
    },
    optimisticUpdate: (currentTicket) => currentTicket,
    mutate: () => resetTicketWorkspace(ticketId),
    successMessage: 'Workspace reset.',
  })
}
