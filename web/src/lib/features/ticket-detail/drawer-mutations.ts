import { runOptimisticTicketMutation } from './optimistic'
import type { TicketDetail } from './types'

type DrawerMutationOptions = {
  ticket: TicketDetail
  mutate: () => Promise<unknown>
  reload: () => Promise<void>
  applyTicket: (ticket: TicketDetail) => void
  clearMessages: () => void
  setError: (message: string) => void
  setNotice: (message: string) => void
  optimisticUpdate: (currentTicket: TicketDetail) => TicketDetail
  successMessage: string
  start?: () => void
  finish?: () => void
}

export async function runTicketDrawerMutation({
  ticket,
  mutate,
  reload,
  applyTicket,
  clearMessages,
  setError,
  setNotice,
  optimisticUpdate,
  successMessage,
  start,
  finish,
}: DrawerMutationOptions) {
  start?.()

  try {
    await runOptimisticTicketMutation({
      ticket,
      optimisticUpdate,
      mutate,
      reload,
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
