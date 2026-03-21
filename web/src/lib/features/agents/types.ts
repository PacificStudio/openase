export type AgentInstance = {
  id: string
  name: string
  providerId: string
  providerName: string
  modelName: string
  status: 'idle' | 'claimed' | 'running' | 'failed' | 'terminated'
  runtimePhase: 'none' | 'launching' | 'ready' | 'failed'
  runtimeControlState: 'active' | 'pause_requested' | 'paused'
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
