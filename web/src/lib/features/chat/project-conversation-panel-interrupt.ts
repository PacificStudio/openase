import { ApiError } from '$lib/api/client'
import { chatT } from './i18n'

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
    chatT('chat.confirmInterruptAgent', { agentName: input.agentName }),
  )
  if (!confirmed) {
    return
  }

  try {
    await input.interruptAgent(input.agentId)
    input.onSuccess(chatT('chat.interruptRequestedNotification', { agentName: input.agentName }))
  } catch (error) {
    input.onError(error instanceof ApiError ? error.detail : chatT('chat.interruptFailed'))
  }
}
