const STORAGE_KEY = 'openase:ticket-view'

type TicketViewMode = 'board' | 'list'

function loadMode(): TicketViewMode {
  if (typeof localStorage === 'undefined') return 'board'
  const stored = localStorage.getItem(STORAGE_KEY)
  return stored === 'list' ? 'list' : 'board'
}

class TicketViewStore {
  mode = $state<TicketViewMode>(loadMode())

  setMode(next: TicketViewMode) {
    this.mode = next
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, next)
    }
  }
}

export const ticketViewStore = new TicketViewStore()
