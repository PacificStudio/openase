import { runOptimisticTicketMutation } from './optimistic'
import type { TicketDetail } from './types'

type LoadOptions = {
  background?: boolean
  preserveMessages?: boolean
}

export async function runTicketDrawerMutation({
  ticket,
  projectId,
  ticketId,
  load,
  applyTicket,
  clearMessages,
  setError,
  setNotice,
  start,
  finish,
  optimisticUpdate,
  mutate,
  successMessage,
}: {
  ticket: TicketDetail | null
  projectId?: string | null
  ticketId?: string | null
  load: (projectId: string, ticketId: string, options?: LoadOptions) => Promise<void>
  applyTicket: (ticket: TicketDetail) => void
  clearMessages: () => void
  setError: (message: string) => void
  setNotice: (message: string) => void
  start?: () => void
  finish?: () => void
  optimisticUpdate: (currentTicket: TicketDetail) => TicketDetail
  mutate: () => Promise<unknown>
  successMessage: string
}) {
  if (!ticket || !projectId || !ticketId) return

  start?.()

  try {
    await runOptimisticTicketMutation({
      ticket,
      optimisticUpdate,
      mutate,
      reload: () => load(projectId, ticketId, { background: true, preserveMessages: true }),
      applyTicket,
      clearMessages,
      setError,
      setNotice,
      successMessage,
    })
  } finally {
    finish?.()
  }
}
