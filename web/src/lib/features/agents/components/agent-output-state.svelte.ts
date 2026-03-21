import { ApiError } from '$lib/api/client'
import type { AgentOutputEntry } from '$lib/api/contracts'
import { listAgentOutput } from '$lib/api/openase'
import type { SSEFrame, StreamConnectionState } from '$lib/api/sse'

const agentOutputLimit = 200

export function createAgentOutputState() {
  let selectedAgentId = $state<string | null>(null)
  let entries = $state<AgentOutputEntry[]>([])
  let loading = $state(false)
  let error = $state('')
  let streamState = $state<StreamConnectionState>('idle')
  let loadRequestId = 0

  return {
    get selectedAgentId() {
      return selectedAgentId
    },
    get entries() {
      return entries
    },
    get loading() {
      return loading
    },
    get error() {
      return error
    },
    get streamState() {
      return streamState
    },
    set streamState(value) {
      streamState = value
    },
    open(agentId: string) {
      selectedAgentId = agentId
      error = ''
    },
    invalidate() {
      loadRequestId += 1
    },
    async load(projectId: string, agentId: string, showLoading: boolean) {
      const requestId = ++loadRequestId
      if (showLoading) {
        loading = true
      }
      error = ''

      try {
        const payload = await listAgentOutput(projectId, agentId, { limit: agentOutputLimit })
        if (requestId !== loadRequestId || selectedAgentId !== agentId) return

        entries = [...payload.entries].reverse()
      } catch (caughtError) {
        if (requestId !== loadRequestId || selectedAgentId !== agentId) return
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load agent output.'
      } finally {
        if (requestId === loadRequestId && selectedAgentId === agentId && showLoading) {
          loading = false
        }
      }
    },
    handleFrame(agentId: string, frame: SSEFrame) {
      if (frame.event !== 'agent.output' || selectedAgentId !== agentId) {
        return
      }

      try {
        const envelope = JSON.parse(frame.data) as {
          payload?: { entry?: AgentOutputEntry }
        }
        const entry = envelope.payload?.entry
        if (!entry) {
          return
        }

        entries = mergeAgentOutputEntry(entries, entry)
      } catch (caughtError) {
        console.error('Failed to parse agent output frame:', caughtError)
      }
    },
    reset() {
      loadRequestId += 1
      selectedAgentId = null
      entries = []
      loading = false
      error = ''
      streamState = 'idle'
    },
  }
}

function mergeAgentOutputEntry(entries: AgentOutputEntry[], entry: AgentOutputEntry) {
  if (entries.some((item) => item.id === entry.id)) {
    return entries
  }

  const nextEntries = [...entries, entry]
  if (nextEntries.length > agentOutputLimit) {
    return nextEntries.slice(nextEntries.length - agentOutputLimit)
  }

  return nextEntries
}
