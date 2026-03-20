import { connectEventStream } from '$lib/api/sse'
import { frameReferencesTicket } from './context'

export function connectTicketDetailStreams(
  projectId: string,
  ticketId: string,
  onRelevantEvent: () => void,
) {
  const connect = (path: string, label: string) =>
    connectEventStream(path, {
      onEvent: (frame) => {
        if (frameReferencesTicket(frame.data, ticketId)) {
          onRelevantEvent()
        }
      },
      onError: (streamError) => {
        console.error(`Ticket detail ${label} stream error:`, streamError)
      },
    })

  const disconnectTicketStream = connect(`/api/v1/projects/${projectId}/tickets/stream`, 'tickets')
  const disconnectActivityStream = connect(
    `/api/v1/projects/${projectId}/activity/stream`,
    'activity',
  )

  return () => {
    disconnectTicketStream()
    disconnectActivityStream()
  }
}
