import { connectEventStream } from '$lib/api/sse'
import { createAgentOutputState } from './agent-output-state.svelte'

type AgentOutputState = ReturnType<typeof createAgentOutputState>

export function wireAgentOutputStream(input: {
  projectId: () => string | undefined
  isOpen: () => boolean
  selectedAgentId: () => string | null
  outputState: AgentOutputState
}) {
  $effect(() => {
    const projectId = input.projectId()
    const agentId = input.selectedAgentId()

    if (!input.isOpen() || !projectId || !agentId) {
      if (!input.isOpen()) input.outputState.reset()
      return
    }

    let cancelled = false
    let disconnectOutput: (() => void) | null = null
    let disconnectSteps: (() => void) | null = null

    // Load initial data first, then open SSE streams.
    // Opening all connections simultaneously exceeds the browser's per-origin
    // connection limit (6 for HTTP/1.1), causing ERR_INSUFFICIENT_RESOURCES.
    void input.outputState.load(projectId, agentId, true).then(() => {
      if (cancelled) return

      disconnectOutput = connectEventStream(
        `/api/v1/projects/${projectId}/agents/${agentId}/output/stream`,
        {
          onEvent: (frame) => input.outputState.handleFrame(agentId, frame),
          onStateChange: (state) => {
            input.outputState.setTraceStreamState(state)
          },
          onError: (streamError) => {
            console.error('Agent output stream error:', streamError)
          },
        },
      )
      disconnectSteps = connectEventStream(
        `/api/v1/projects/${projectId}/agents/${agentId}/steps/stream`,
        {
          onEvent: (frame) => input.outputState.handleFrame(agentId, frame),
          onStateChange: (state) => {
            input.outputState.setStepStreamState(state)
          },
          onError: (streamError) => {
            console.error('Agent step stream error:', streamError)
          },
        },
      )
    })

    return () => {
      cancelled = true
      input.outputState.invalidate()
      disconnectOutput?.()
      disconnectSteps?.()
    }
  })
}
