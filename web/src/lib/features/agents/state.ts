import type { AgentInstance } from './types'

export function normalizeAgentStatus(status: string): AgentInstance['status'] {
  if (
    status === 'idle' ||
    status === 'claimed' ||
    status === 'running' ||
    status === 'paused' ||
    status === 'failed' ||
    status === 'interrupted' ||
    status === 'terminated'
  ) {
    return status
  }

  return status === 'active' ? 'running' : 'idle'
}

export function normalizeRuntimePhase(runtimePhase: string): AgentInstance['runtimePhase'] {
  if (
    runtimePhase === 'none' ||
    runtimePhase === 'launching' ||
    runtimePhase === 'ready' ||
    runtimePhase === 'executing' ||
    runtimePhase === 'failed'
  ) {
    return runtimePhase
  }

  return 'none'
}

export function normalizeRuntimeControlState(
  runtimeControlState: string,
): AgentInstance['runtimeControlState'] {
  if (
    runtimeControlState === 'active' ||
    runtimeControlState === 'interrupt_requested' ||
    runtimeControlState === 'pause_requested' ||
    runtimeControlState === 'paused' ||
    runtimeControlState === 'retired'
  ) {
    return runtimeControlState
  }

  return 'active'
}
