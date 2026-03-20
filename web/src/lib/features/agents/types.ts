export type AgentInstance = {
  id: string
  name: string
  providerName: string
  modelName: string
  status: 'idle' | 'running' | 'offline' | 'stalled'
  currentTicket?: { id: string; identifier: string; title: string }
  lastHeartbeat: string
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
