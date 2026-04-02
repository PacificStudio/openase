import { connectEventStream } from '$lib/api/sse'
import { subscribeProjectEvents } from '$lib/features/project-events/project-event-bus'

export function connectAgentsPageStreams(
  projectId: string,
  orgId: string,
  onEvent: () => void,
): () => void {
  const disconnectAgents = subscribeProjectEvents(projectId, onEvent)
  const disconnectProviders = connectEventStream(`/api/v1/orgs/${orgId}/providers/stream`, {
    onEvent,
    onError: (streamError) => {
      console.error('Providers stream error:', streamError)
    },
  })

  return () => {
    disconnectAgents()
    disconnectProviders()
  }
}
