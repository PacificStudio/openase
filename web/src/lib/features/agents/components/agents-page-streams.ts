import { connectEventStream } from '$lib/api/sse'
import {
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
} from '$lib/features/project-events'

export function connectAgentsPageStreams(
  projectId: string,
  orgId: string,
  onEvent: () => void,
): () => void {
  const disconnectAgents = subscribeProjectEvents(projectId, (event) => {
    if (!isProjectDashboardRefreshEvent(event)) {
      return
    }

    const sections = readProjectDashboardRefreshSections(event)
    if (sections.includes('agents') || sections.includes('tickets')) {
      onEvent()
    }
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
