export type AgentInstance = {
  id: string
  name: string
  providerId: string
  providerName: string
  modelName: string
  adapterType: string
  permissionProfile: ProviderPermissionProfile
  status: 'idle' | 'claimed' | 'running' | 'paused' | 'failed' | 'terminated'
  runtimePhase: 'none' | 'launching' | 'ready' | 'executing' | 'failed'
  runtimeControlState: 'active' | 'pause_requested' | 'paused' | 'retired'
  activeRunCount: number
  currentTicket?: { id: string; identifier: string; title: string }
  lastHeartbeat?: string | null
  runtimeStartedAt?: string | null
  sessionId?: string
  lastError?: string
  currentStepStatus?: string
  currentStepSummary?: string
  currentStepChangedAt?: string | null
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
  permissionProfile: ProviderPermissionProfile
  availabilityState: string
  available: boolean
  availabilityCheckedAt?: string | null
  availabilityReason?: string | null
  cliCommand: string
  cliArgs: string[]
  authConfig: Record<string, unknown>
  cliRateLimit?: ProviderCLIRateLimit | null
  cliRateLimitUpdatedAt?: string | null
  modelName: string
  modelTemperature: number
  modelMaxTokens: number
  maxParallelRuns: number
  costPerInputToken: number
  costPerOutputToken: number
  pricingConfig: ProviderPricingConfig
  agentCount: number
  isDefault: boolean
}

export type ProviderPricingRates = {
  input_per_token?: number
  output_per_token?: number
  cached_input_read_per_token?: number
  cache_write_5m_per_token?: number
  cache_write_1h_per_token?: number
  cache_storage_per_token_hour?: number
}

export type ProviderPricingTier = {
  label?: string
  max_prompt_tokens?: number
  rates?: ProviderPricingRates
}

export type ProviderPricingConfig = {
  version?: string
  source_kind?: string
  pricing_mode?: string
  provider?: string
  model_id?: string
  source_url?: string
  source_verified_at?: string
  default_cache_write_window?: string
  notes?: string[]
  rates?: ProviderPricingRates
  tiers?: ProviderPricingTier[]
}

export type ProviderCLIRateLimit = {
  provider: string
  raw: Record<string, unknown>
  claudeCode?: {
    status: string
    rateLimitType?: string | null
    resetsAt?: string | null
    utilization?: number | null
    surpassedThreshold?: number | null
    overageStatus?: string | null
    overageDisabledReason?: string | null
    isUsingOverage?: boolean | null
  } | null
  codex?: {
    limitId?: string | null
    limitName?: string | null
    planType?: string | null
    primary?: {
      usedPercent?: number | null
      windowMinutes?: number | null
      resetsAt?: string | null
    } | null
    secondary?: {
      usedPercent?: number | null
      windowMinutes?: number | null
      resetsAt?: string | null
    } | null
  } | null
  gemini?: {
    authType?: string | null
    remaining?: number | null
    limit?: number | null
    resetTime?: string | null
    buckets?: Array<{
      modelId?: string | null
      tokenType?: string | null
      remainingAmount?: string | null
      remainingFraction?: number | null
      resetTime?: string | null
    }>
  } | null
}

export type ProviderAdapterType = 'claude-code-cli' | 'codex-app-server' | 'gemini-cli' | 'custom'
export type ProviderPermissionProfile = 'standard' | 'unrestricted'

export type ProviderDraft = {
  machineId: string
  name: string
  adapterType: string
  permissionProfile: string
  cliCommand: string
  cliArgs: string
  authConfig: string
  modelName: string
  modelTemperature: string
  modelMaxTokens: string
  maxParallelRuns: string
  costPerInputToken: string
  costPerOutputToken: string
  pricingConfig: string
}

export type ProviderDraftField = keyof ProviderDraft

export type ProviderMutation = {
  machine_id: string
  name: string
  adapter_type: ProviderAdapterType
  permission_profile: ProviderPermissionProfile
  cli_command: string
  cli_args: string[]
  auth_config: Record<string, unknown>
  model_name: string
  model_temperature: number
  model_max_tokens: number
  max_parallel_runs: number
  cost_per_input_token: number
  cost_per_output_token: number
  pricing_config: ProviderPricingConfig
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
