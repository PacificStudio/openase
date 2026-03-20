import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
import { api, toErrorMessage } from '$lib/features/workspace/api'
import { agentConsoleLimit } from '$lib/features/workspace/constants'
import {
  chooseAgentSelection,
  dedupeActivityEvents,
  hasAutomationSignal,
  stalledAgentCount,
} from '$lib/features/workspace/metrics'
import {
  parseActivityEvent,
  parseAgentPatch,
  parseStreamEnvelope,
} from '$lib/features/workspace/stream'
import type {
  ActivityEvent,
  ActivityPayload,
  Agent,
  AgentPayload,
  HRAdvisorPayload,
  HRAdvisorResponse,
  Ticket,
} from '$lib/features/workspace/types'

export function createDashboardStore() {
  let activeProjectId = $state('')
  let agents = $state<Agent[]>([])
  let activityEvents = $state<ActivityEvent[]>([])
  let selectedAgentId = $state('')
  let agentConsoleBusy = $state(false)
  let agentConsoleError = $state('')
  let agentStreamState = $state<StreamConnectionState>('idle')
  let activityStreamState = $state<StreamConnectionState>('idle')
  let heartbeatNow = $state(Date.now())
  let hrAdvisor = $state<HRAdvisorPayload | null>(null)
  let hrAdvisorBusy = $state(false)
  let hrAdvisorError = $state('')

  function setProject(projectId: string) {
    activeProjectId = projectId
  }

  function reset() {
    activeProjectId = ''
    agents = []
    activityEvents = []
    selectedAgentId = ''
    agentConsoleBusy = false
    agentConsoleError = ''
    agentStreamState = 'idle'
    activityStreamState = 'idle'
    hrAdvisor = null
    hrAdvisorBusy = false
    hrAdvisorError = ''
  }

  function tickHeartbeat() {
    heartbeatNow = Date.now()
  }

  async function loadAgents(projectId: string) {
    activeProjectId = projectId
    agentConsoleBusy = true
    agentConsoleError = ''

    try {
      const payload = await api<AgentPayload>(`/api/v1/projects/${projectId}/agents`)
      if (projectId !== activeProjectId) {
        return
      }

      agents = payload.agents
      selectedAgentId = chooseAgentSelection(payload.agents, selectedAgentId)
    } catch (error) {
      if (projectId === activeProjectId) {
        agents = []
        selectedAgentId = ''
        agentConsoleError = toErrorMessage(error)
      }
    } finally {
      if (projectId === activeProjectId) {
        agentConsoleBusy = false
      }
    }
  }

  async function loadActivityEvents(projectId: string, agentId = selectedAgentId) {
    activeProjectId = projectId
    agentConsoleError = ''

    try {
      const query = new URLSearchParams({ limit: String(agentConsoleLimit) })
      if (agentId) {
        query.set('agent_id', agentId)
      }

      const payload = await api<ActivityPayload>(
        `/api/v1/projects/${projectId}/activity?${query.toString()}`,
      )
      if (projectId !== activeProjectId) {
        return
      }

      activityEvents = payload.events
    } catch (error) {
      if (projectId === activeProjectId) {
        activityEvents = []
        agentConsoleError = toErrorMessage(error)
      }
    }
  }

  async function loadHRAdvisor(projectId: string) {
    activeProjectId = projectId
    hrAdvisorBusy = true
    hrAdvisorError = ''

    try {
      const payload = await api<HRAdvisorResponse>(`/api/v1/projects/${projectId}/hr-advisor`)
      if (projectId === activeProjectId) {
        hrAdvisor = payload
      }
    } catch (error) {
      if (projectId === activeProjectId) {
        hrAdvisor = null
        hrAdvisorError = toErrorMessage(error)
      }
    } finally {
      if (projectId === activeProjectId) {
        hrAdvisorBusy = false
      }
    }
  }

  async function selectAgent(agentId: string) {
    selectedAgentId = agentId
    if (!activeProjectId) {
      return
    }

    await loadActivityEvents(activeProjectId, agentId)
  }

  function connect(projectId: string) {
    activeProjectId = projectId

    const closeAgentStream = connectEventStream(`/api/v1/projects/${projectId}/agents/stream`, {
      onEvent: (frame) => handleAgentFrame(projectId, frame),
      onStateChange: (state) => {
        agentStreamState = state
      },
      onError: (error) => {
        agentConsoleError = toErrorMessage(error)
      },
    })

    const closeActivityStream = connectEventStream(
      `/api/v1/projects/${projectId}/activity/stream`,
      {
        onEvent: (frame) => handleActivityFrame(projectId, frame),
        onStateChange: (state) => {
          activityStreamState = state
        },
        onError: (error) => {
          agentConsoleError = toErrorMessage(error)
        },
      },
    )

    return () => {
      closeAgentStream()
      closeActivityStream()
    }
  }

  function currentAgent() {
    return agents.find((item) => item.id === selectedAgentId) ?? null
  }

  function runningAgentCount() {
    return agents.filter((item) => item.status === 'running').length
  }

  function selectedAgentName() {
    return currentAgent()?.name ?? 'All agents'
  }

  function hasSignal(tickets: Ticket[]) {
    return hasAutomationSignal(agents, tickets, activityEvents)
  }

  function stalledCount() {
    return stalledAgentCount(agents, heartbeatNow)
  }

  function handleAgentFrame(projectId: string, frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || projectId !== activeProjectId) {
      return
    }

    const patch = parseAgentPatch(envelope.payload)
    if (!patch) {
      void loadAgents(projectId)
      return
    }

    if (frame.event.includes('heartbeat') && !patch.last_heartbeat_at) {
      patch.last_heartbeat_at = envelope.published_at
    }

    agents = upsertAgent(agents, patch)
    selectedAgentId = chooseAgentSelection(agents, selectedAgentId)
  }

  function handleActivityFrame(projectId: string, frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || projectId !== activeProjectId) {
      return
    }

    const activityEvent = parseActivityEvent(envelope.payload, envelope.published_at)
    if (!activityEvent) {
      void loadActivityEvents(projectId, selectedAgentId)
      return
    }
    if (selectedAgentId && activityEvent.agent_id !== selectedAgentId) {
      return
    }

    activityEvents = dedupeActivityEvents([activityEvent, ...activityEvents]).slice(
      0,
      agentConsoleLimit,
    )
  }

  return {
    get agents() {
      return agents
    },
    get activityEvents() {
      return activityEvents
    },
    get selectedAgentId() {
      return selectedAgentId
    },
    get busy() {
      return agentConsoleBusy
    },
    get error() {
      return agentConsoleError
    },
    get agentStreamState() {
      return agentStreamState
    },
    get activityStreamState() {
      return activityStreamState
    },
    get heartbeatNow() {
      return heartbeatNow
    },
    get hrAdvisor() {
      return hrAdvisor
    },
    get hrAdvisorBusy() {
      return hrAdvisorBusy
    },
    get hrAdvisorError() {
      return hrAdvisorError
    },
    setProject,
    reset,
    tickHeartbeat,
    loadAgents,
    loadActivityEvents,
    loadHRAdvisor,
    selectAgent,
    connect,
    currentAgent,
    runningAgentCount,
    selectedAgentName,
    hasSignal,
    stalledCount,
  }
}

function upsertAgent(items: Agent[], patch: Partial<Agent> & { id: string }) {
  const index = items.findIndex((item) => item.id === patch.id)
  if (index === -1) {
    if (!patch.name || !patch.project_id || !patch.provider_id || !patch.status) {
      return items
    }

    return [
      ...items,
      {
        id: patch.id,
        provider_id: patch.provider_id,
        project_id: patch.project_id,
        name: patch.name,
        status: patch.status,
        current_ticket_id: patch.current_ticket_id ?? null,
        session_id: patch.session_id ?? '',
        workspace_path: patch.workspace_path ?? '',
        capabilities: patch.capabilities ?? [],
        total_tokens_used: patch.total_tokens_used ?? 0,
        total_tickets_completed: patch.total_tickets_completed ?? 0,
        last_heartbeat_at: patch.last_heartbeat_at ?? null,
      },
    ].sort((left, right) => left.name.localeCompare(right.name))
  }

  const next = [...items]
  next[index] = { ...items[index], ...patch }
  return next
}
