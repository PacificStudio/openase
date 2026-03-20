export type AgentInstance = {
  id: string
  name: string
  providerName: string
  modelName: string
  status: 'idle' | 'claimed' | 'running' | 'failed' | 'terminated'
  runtimePhase: 'none' | 'launching' | 'ready' | 'failed'
  currentTicket?: { id: string; identifier: string; title: string }
  lastHeartbeat?: string | null
  runtimeStartedAt?: string | null
  sessionId?: string
  lastError?: string
  todayCompleted: number
  todayCost: number
  capabilities: string[]
}

export type ProviderConfig = {
  id: string
  name: string
  adapterType: string
  modelName: string
  agentCount: number
  isDefault: boolean
}
