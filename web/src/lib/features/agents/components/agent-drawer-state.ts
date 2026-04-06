import type { AgentInstance } from '../types'

export const agentStatusVariant: Record<AgentInstance['status'], string> = {
  idle: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400',
  claimed: 'bg-amber-500/15 text-amber-700 dark:text-amber-400',
  running: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
  paused: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
  failed: 'bg-red-500/15 text-red-700 dark:text-red-400',
  interrupted: 'bg-rose-500/15 text-rose-700 dark:text-rose-400',
  terminated: 'bg-slate-500/15 text-slate-600 dark:text-slate-400',
}

export const agentStatusDot: Record<AgentInstance['status'], string> = {
  idle: 'bg-emerald-500',
  claimed: 'bg-amber-500',
  running: 'bg-blue-500',
  paused: 'bg-orange-500',
  failed: 'bg-red-500',
  interrupted: 'bg-rose-500',
  terminated: 'bg-slate-500',
}

export const agentStatusLabel: Record<AgentInstance['status'], string> = {
  idle: 'Idle',
  claimed: 'Claimed',
  running: 'Running',
  paused: 'Paused',
  failed: 'Failed',
  interrupted: 'Interrupted',
  terminated: 'Terminated',
}

export function canInterruptAgent(agent: AgentInstance) {
  return (
    agent.runtimeControlState === 'active' &&
    agent.activeRunCount > 0 &&
    (agent.status === 'claimed' || agent.status === 'running')
  )
}

export function canPauseAgent(agent: AgentInstance) {
  return (
    agent.runtimeControlState === 'active' &&
    agent.activeRunCount > 0 &&
    (agent.status === 'claimed' || agent.status === 'running')
  )
}

export function canResumeAgent(agent: AgentInstance) {
  return agent.runtimeControlState === 'paused'
}

export function canRetireAgent(agent: AgentInstance) {
  return agent.runtimeControlState !== 'retired' && agent.activeRunCount === 0
}
