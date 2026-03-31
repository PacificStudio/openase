import { connectEventStream } from '$lib/api/sse'

export function connectAgentsPageStreams(
  projectId: string,
  orgId: string,
  onEvent: () => void,
): () => void {
  const disconnectAgents = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
    onEvent,
    onError: (streamError) => {
      console.error('Agents stream error:', streamError)
    },
  })
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
