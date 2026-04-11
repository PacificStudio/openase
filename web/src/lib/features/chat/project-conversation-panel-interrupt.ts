import { ApiError } from '$lib/api/client'

type InterruptFocusedAgentInput = {
  agentId: string
  agentName: string
  interruptAgent: (agentId: string) => Promise<unknown>
  onSuccess: (message: string) => void
  onError: (message: string) => void
}

export async function interruptFocusedProjectAgent(
  input: InterruptFocusedAgentInput | null,
): Promise<void> {
  if (!input) {
    return
  }

  const confirmed = window.confirm(
    `Interrupt "${input.agentName}"? This stops the current agent run. Use Close Runtime separately if you want to stop Project AI itself.`,
  )
  if (!confirmed) {
    return
  }

  try {
    await input.interruptAgent(input.agentId)
    input.onSuccess(`Interrupt requested for "${input.agentName}".`)
  } catch (error) {
    input.onError(
      error instanceof ApiError ? error.detail : 'Failed to interrupt the focused agent.',
    )
  }
}
