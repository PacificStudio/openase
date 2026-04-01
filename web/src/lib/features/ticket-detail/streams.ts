import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
import { frameReferencesTicket } from './context'

export function connectTicketDetailStreams(
  projectId: string,
  ticketId: string,
  handlers: {
    onRelevantEvent: () => void
    onRunFrame: (frame: SSEFrame) => void
    onRunStateChange?: (state: StreamConnectionState) => void
  },
) {
  const connect = (path: string, label: string) =>
    connectEventStream(path, {
      onEvent: (frame) => {
        if (frameReferencesTicket(frame.data, ticketId)) {
          handlers.onRelevantEvent()
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
  const disconnectRunStream = connectEventStream(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/runs/stream`,
    {
      onEvent: (frame) => {
        handlers.onRunFrame(frame)
      },
      onStateChange: (state) => {
        handlers.onRunStateChange?.(state)
      },
      onError: (streamError) => {
        console.error('Ticket detail runs stream error:', streamError)
      },
    },
  )

  return () => {
    disconnectTicketStream()
    disconnectActivityStream()
    disconnectRunStream()
  }
}
