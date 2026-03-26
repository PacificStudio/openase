import { ApiError } from '$lib/api/client'
import { cloneTicketDetail } from './mutation-shared'
import type { TicketDetail } from './types'

export async function runOptimisticTicketMutation({
  ticket,
  optimisticUpdate,
  mutate,
  reload,
  applyTicket,
  clearMessages,
  setError,
  setNotice,
  successMessage,
}: {
  ticket: TicketDetail
  optimisticUpdate: (currentTicket: TicketDetail) => TicketDetail
  mutate: () => Promise<unknown>
  reload: () => Promise<void>
  applyTicket: (ticket: TicketDetail) => void
  clearMessages: () => void
  setError: (message: string) => void
  setNotice: (message: string) => void
  successMessage: string
}) {
  const previousTicket = cloneTicketDetail(ticket)
  clearMessages()
  applyTicket(optimisticUpdate(previousTicket))

  try {
    await mutate()
    setNotice(successMessage)
    await reload()
    return true
  } catch (caughtError) {
    applyTicket(previousTicket)
    setError(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to update ticket detail.',
    )
    void reload()
    return false
  }
}
