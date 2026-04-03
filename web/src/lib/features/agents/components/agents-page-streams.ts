import { connectEventStream } from '$lib/api/sse'
import {
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
  type ProjectEventEnvelope,
} from '$lib/features/project-events'

export function connectAgentsPageStreams(
  projectId: string,
  orgId: string,
  onEvent: () => void,
): () => void {
  const disconnectAgents = subscribeProjectEvents(projectId, (event) => {
    if (isImmediateRuntimeRefreshEvent(event)) {
      onEvent()
      return
    }

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

function isImmediateRuntimeRefreshEvent(event: Pick<ProjectEventEnvelope, 'topic' | 'type'>) {
  if (event.topic === 'agent.events') {
    return event.type !== 'agent.heartbeat'
  }

  return event.topic === 'ticket.run.events' && event.type === 'ticket.run.lifecycle'
}
