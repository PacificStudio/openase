import type { SSEFrame, StreamConnectionState } from '$lib/api/sse'
import {
  isTicketRunProjectEvent,
  projectEventAffectsTicketDetailReferences,
  projectEventReferencesTicket,
  subscribeProjectEvents,
  subscribeProjectEventBusState,
  toProjectEventFrame,
} from '$lib/features/project-events'

export function connectTicketDetailStreams(
  projectId: string,
  ticketId: string,
  handlers: {
    onRelevantEvent: () => void
    onReferenceEvent?: () => void
    onRunFrame: (frame: SSEFrame) => void
    onRunStateChange?: (state: StreamConnectionState) => void
  },
) {
  const disconnectProjectEvents = subscribeProjectEvents(projectId, (event) => {
    if (projectEventReferencesTicket(event, ticketId)) {
      handlers.onRelevantEvent()
    }
    if (isTicketRunProjectEvent(event) && projectEventReferencesTicket(event, ticketId)) {
      handlers.onRunFrame(toProjectEventFrame(event))
    }
    if (projectEventAffectsTicketDetailReferences(event, ticketId)) {
      handlers.onReferenceEvent?.()
    }
  })
  const disconnectRunState = subscribeProjectEventBusState(projectId, (state) => {
    handlers.onRunStateChange?.(state)
  })

  return () => {
    disconnectProjectEvents()
    disconnectRunState()
  }
}
