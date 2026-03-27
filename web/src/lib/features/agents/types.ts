export type AgentInstance = {
  id: string
  name: string
  providerId: string
  providerName: string
  modelName: string
  status: 'idle' | 'claimed' | 'running' | 'paused' | 'failed' | 'terminated'
  runtimePhase: 'none' | 'launching' | 'ready' | 'executing' | 'failed'
  runtimeControlState: 'active' | 'pause_requested' | 'paused'
  activeRunCount: number
  currentTicket?: { id: string; identifier: string; title: string }
  lastHeartbeat?: string | null
  runtimeStartedAt?: string | null
  sessionId?: string
  lastError?: string
  todayCompleted: number
  todayCost: number
}

export type AgentRunInstance = {
  id: string
  agentId: string
  agentName: string
  providerId: string
  providerName: string
  modelName: string
  workflowId: string
  workflowName: string
  status: 'launching' | 'ready' | 'executing' | 'completed' | 'errored' | 'terminated'
  ticket: { id: string; identifier: string; title: string }
  lastHeartbeat: string | null
  runtimeStartedAt: string | null
  sessionId: string
  lastError: string
  createdAt: string
}

export type ProviderConfig = {
  id: string
  machineId: string
  machineName: string
  machineHost: string
  machineStatus: string
  machineWorkspaceRoot?: string | null
  name: string
  adapterType: string
  availabilityState: string
  available: boolean
  availabilityCheckedAt?: string | null
  availabilityReason?: string | null
  cliCommand: string
  cliArgs: string[]
  authConfig: Record<string, unknown>
  modelName: string
  modelTemperature: number
  modelMaxTokens: number
  costPerInputToken: number
  costPerOutputToken: number
  agentCount: number
  isDefault: boolean
}

export type ProviderAdapterType = 'claude-code-cli' | 'codex-app-server' | 'gemini-cli' | 'custom'

export type ProviderDraft = {
  machineId: string
  name: string
  adapterType: string
  cliCommand: string
  cliArgs: string
  authConfig: string
  modelName: string
  modelTemperature: string
  modelMaxTokens: string
  costPerInputToken: string
  costPerOutputToken: string
}

export type ProviderDraftField = keyof ProviderDraft

export type ProviderMutation = {
  machine_id: string
  name: string
  adapter_type: ProviderAdapterType
  cli_command: string
  cli_args: string[]
  auth_config: Record<string, unknown>
  model_name: string
  model_temperature: number
  model_max_tokens: number
  cost_per_input_token: number
  cost_per_output_token: number
}

export type ProviderDraftParseResult =
  | {
      ok: true
      value: ProviderMutation
    }
  | {
      ok: false
      error: string
    }
