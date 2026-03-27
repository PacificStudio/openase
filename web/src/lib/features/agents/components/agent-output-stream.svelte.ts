import { connectEventStream } from '$lib/api/sse'
import { createAgentOutputState } from './agent-output-state.svelte'

type AgentOutputState = ReturnType<typeof createAgentOutputState>

export function wireAgentOutputStream(input: {
  projectId: () => string | undefined
  isOpen: () => boolean
  outputState: AgentOutputState
}) {
  $effect(() => {
    const projectId = input.projectId()
    const agentId = input.outputState.selectedAgentId

    if (!input.isOpen() || !projectId || !agentId) {
      if (!input.isOpen()) input.outputState.reset()
      return
    }

    void input.outputState.load(projectId, agentId, true)

    const disconnectOutput = connectEventStream(
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
    const disconnectSteps = connectEventStream(
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

    return () => {
      input.outputState.invalidate()
      disconnectOutput()
      disconnectSteps()
    }
  })
}
