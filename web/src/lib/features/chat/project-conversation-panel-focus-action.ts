import type { ProjectAIFocus } from './project-ai-focus'

export function deriveFocusInterruptTarget(focus: ProjectAIFocus | null) {
  if (focus?.kind !== 'ticket') return null
  const agent = focus.ticketAssignedAgent
  const run = focus.ticketCurrentRun
  if (
    !agent?.id ||
    agent.runtimeControlState !== 'active' ||
    (run?.status !== 'launching' && run?.status !== 'ready' && run?.status !== 'executing')
  ) {
    return null
  }
  return {
    agentId: agent.id,
    agentName: agent.name || focus.ticketIdentifier,
  }
}
