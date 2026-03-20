import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
import {
  api,
  orderTicketStatuses,
  orderTickets,
  parseStreamEnvelope,
  toErrorMessage,
  type Ticket,
  type TicketPayload,
  type TicketStatus,
} from '$lib/features/workspace'

type LoadOptions = { silent?: boolean }
type TicketMutationCallback = (projectId: string) => void

export function createBoardStore(onTicketMutation?: TicketMutationCallback) {
  let activeProjectId = $state('')
  let ticketStatuses = $state<TicketStatus[]>([])
  let tickets = $state<Ticket[]>([])
  let ticketBoardBusy = $state(false)
  let ticketBoardError = $state('')
  let ticketStreamState = $state<StreamConnectionState>('idle')
  let draggingTicketId = $state('')
  let dragTargetStatusId = $state('')
  let ticketMutationIds = $state<string[]>([])
  let ticketLoadInFlight = false
  let ticketReloadQueued = false

  function setProject(projectId: string) {
    activeProjectId = projectId
  }

  function setStatuses(statuses: TicketStatus[]) {
    ticketStatuses = orderTicketStatuses(statuses)
  }

  function reset() {
    activeProjectId = ''
    ticketStatuses = []
    tickets = []
    ticketBoardBusy = false
    ticketBoardError = ''
    ticketStreamState = 'idle'
    draggingTicketId = ''
    dragTargetStatusId = ''
    ticketMutationIds = []
    ticketLoadInFlight = false
    ticketReloadQueued = false
  }

  async function load(projectId: string, options: LoadOptions = {}) {
    const silent = options.silent ?? false
    activeProjectId = projectId
    if (!silent) {
      ticketBoardBusy = true
    }

    ticketLoadInFlight = true
    try {
      const payload = await api<TicketPayload>(`/api/v1/projects/${projectId}/tickets`)
      if (projectId !== activeProjectId) {
        return
      }

      tickets = orderTickets(payload.tickets)
      ticketBoardError = ''
    } catch (error) {
      if (projectId === activeProjectId) {
        ticketBoardError = toErrorMessage(error)
      }
    } finally {
      const shouldReload = ticketReloadQueued
      ticketReloadQueued = false
      ticketLoadInFlight = false

      if (projectId === activeProjectId && !silent) {
        ticketBoardBusy = false
      }
      if (shouldReload && projectId === activeProjectId) {
        void load(projectId, { silent: true })
      }
    }
  }

  function connect(projectId: string) {
    activeProjectId = projectId

    return connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: (frame) => handleTicketFrame(projectId, frame),
      onStateChange: (state) => {
        ticketStreamState = state
      },
      onError: (error) => {
        ticketBoardError = toErrorMessage(error)
      },
    })
  }

  function ticketsForStatus(statusID: string) {
    return tickets.filter((ticket) => ticket.status_id === statusID)
  }

  function statusName(statusID?: string | null) {
    if (!statusID) {
      return 'No finish state'
    }

    return ticketStatuses.find((status) => status.id === statusID)?.name ?? 'Unknown'
  }

  function isTicketMutationPending(ticketID: string) {
    return ticketMutationIds.includes(ticketID)
  }

  function handleTicketDragStart(event: DragEvent, ticket: Ticket) {
    draggingTicketId = ticket.id
    dragTargetStatusId = ticket.status_id
    event.dataTransfer?.setData('text/plain', ticket.id)
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
    }
  }

  function handleStatusDragOver(event: DragEvent, statusID: string) {
    event.preventDefault()
    dragTargetStatusId = statusID
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move'
    }
  }

  async function handleStatusDrop(event: DragEvent, statusID: string) {
    event.preventDefault()
    const ticketID = draggingTicketId || event.dataTransfer?.getData('text/plain') || ''
    dragTargetStatusId = ''
    const ticket = tickets.find((item) => item.id === ticketID)
    if (!ticket) {
      return
    }

    await moveTicketToStatus(ticket, statusID)
  }

  async function moveTicketToStatus(ticket: Ticket, statusID: string) {
    if (ticket.status_id === statusID || isTicketMutationPending(ticket.id)) {
      return
    }

    const previousStatusID = ticket.status_id
    const previousStatusName = ticket.status_name
    ticketBoardError = ''
    ticketMutationIds = [...ticketMutationIds, ticket.id]
    tickets = orderTickets(
      tickets.map((item) =>
        item.id === ticket.id
          ? { ...item, status_id: statusID, status_name: statusName(statusID) }
          : item,
      ),
    )

    try {
      const payload = await api<{ ticket: Ticket }>(`/api/v1/tickets/${ticket.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ status_id: statusID }),
      })
      tickets = orderTickets(
        tickets.map((item) => (item.id === payload.ticket.id ? payload.ticket : item)),
      )
      if (activeProjectId) {
        onTicketMutation?.(activeProjectId)
      }
    } catch (error) {
      tickets = orderTickets(
        tickets.map((item) =>
          item.id === ticket.id
            ? { ...item, status_id: previousStatusID, status_name: previousStatusName }
            : item,
        ),
      )
      ticketBoardError = toErrorMessage(error)
    } finally {
      ticketMutationIds = ticketMutationIds.filter((itemID) => itemID !== ticket.id)
      draggingTicketId = ''
      dragTargetStatusId = ''
    }
  }

  function handleTicketFrame(projectId: string, frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || projectId !== activeProjectId) {
      return
    }

    queueTicketReload(projectId)
  }

  function queueTicketReload(projectId: string) {
    if (projectId !== activeProjectId) {
      return
    }
    if (ticketLoadInFlight) {
      ticketReloadQueued = true
      return
    }

    void load(projectId, { silent: true })
  }

  return {
    get projectId() {
      return activeProjectId
    },
    get statuses() {
      return ticketStatuses
    },
    get tickets() {
      return tickets
    },
    get busy() {
      return ticketBoardBusy
    },
    get error() {
      return ticketBoardError
    },
    get streamState() {
      return ticketStreamState
    },
    get dragTargetStatusId() {
      return dragTargetStatusId
    },
    setProject,
    setStatuses,
    reset,
    load,
    connect,
    ticketsForStatus,
    statusName,
    isTicketMutationPending,
    handleTicketDragStart,
    handleStatusDragOver,
    handleStatusDrop,
    moveTicketToStatus,
  }
}
