import type { AgentPayload, AgentProvider, Ticket } from '$lib/api/contracts'
import { normalizeProviderAvailabilityState } from '$lib/features/providers'
import type {
  AgentInstance,
  ProviderDraft,
  ProviderDraftParseResult,
  ProviderConfig,
  ProviderAdapterType,
} from './types'
import { normalizeAgentStatus, normalizeRuntimeControlState, normalizeRuntimePhase } from './state'

export const providerAdapterOptions: Array<{ value: ProviderAdapterType; label: string }> = [
  { value: 'claude-code-cli', label: 'Claude Code CLI' },
  { value: 'codex-app-server', label: 'Codex App Server' },
  { value: 'gemini-cli', label: 'Gemini CLI' },
  { value: 'custom', label: 'Custom' },
]

export function createEmptyProviderDraft(): ProviderDraft {
  return {
    machineId: '',
    name: '',
    adapterType: 'custom',
    cliCommand: '',
    cliArgs: '',
    authConfig: '',
    modelName: '',
    modelTemperature: '0',
    modelMaxTokens: '1',
    costPerInputToken: '0',
    costPerOutputToken: '0',
  }
}

export function providerToDraft(provider: ProviderConfig): ProviderDraft {
  return {
    machineId: provider.machineId,
    name: provider.name,
    adapterType: provider.adapterType,
    cliCommand: provider.cliCommand,
    cliArgs: provider.cliArgs.join('\n'),
    authConfig:
      Object.keys(provider.authConfig).length > 0
        ? JSON.stringify(provider.authConfig, null, 2)
        : '',
    modelName: provider.modelName,
    modelTemperature: String(provider.modelTemperature),
    modelMaxTokens: String(provider.modelMaxTokens),
    costPerInputToken: String(provider.costPerInputToken),
    costPerOutputToken: String(provider.costPerOutputToken),
  }
}

export function parseProviderDraft(draft: ProviderDraft): ProviderDraftParseResult {
  const machineId = draft.machineId.trim()
  if (!machineId) {
    return { ok: false, error: 'Execution machine is required.' }
  }

  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Provider name is required.' }
  }

  const adapterType = parseProviderAdapterType(draft.adapterType)
  if (!adapterType) {
    return {
      ok: false,
      error: `Adapter type must be one of ${providerAdapterOptions.map((option) => option.value).join(', ')}.`,
    }
  }

  const modelName = draft.modelName.trim()
  if (!modelName) {
    return { ok: false, error: 'Model name is required.' }
  }

  const modelTemperature = parseNonNegativeNumber('Model temperature', draft.modelTemperature)
  if (!modelTemperature.ok) {
    return modelTemperature
  }

  const modelMaxTokens = parsePositiveInteger('Model max tokens', draft.modelMaxTokens)
  if (!modelMaxTokens.ok) {
    return modelMaxTokens
  }

  const costPerInputToken = parseNonNegativeNumber('Input token cost', draft.costPerInputToken)
  if (!costPerInputToken.ok) {
    return costPerInputToken
  }

  const costPerOutputToken = parseNonNegativeNumber('Output token cost', draft.costPerOutputToken)
  if (!costPerOutputToken.ok) {
    return costPerOutputToken
  }

  const authConfig = parseAuthConfig(draft.authConfig)
  if (!authConfig.ok) {
    return authConfig
  }

  return {
    ok: true,
    value: {
      machine_id: machineId,
      name,
      adapter_type: adapterType,
      cli_command: draft.cliCommand.trim(),
      cli_args: splitLines(draft.cliArgs),
      auth_config: authConfig.value,
      model_name: modelName,
      model_temperature: modelTemperature.value,
      model_max_tokens: modelMaxTokens.value,
      cost_per_input_token: costPerInputToken.value,
      cost_per_output_token: costPerOutputToken.value,
    },
  }
}

export function buildProviderCards(
  providerItems: AgentProvider[],
  agentItems: AgentPayload['agents'],
  defaultProviderId: string | null,
): ProviderConfig[] {
  return providerItems.map((provider) => ({
    id: provider.id,
    machineId: provider.machine_id,
    machineName: provider.machine_name,
    machineHost: provider.machine_host,
    machineStatus: provider.machine_status,
    machineWorkspaceRoot: provider.machine_workspace_root ?? null,
    name: provider.name,
    adapterType: provider.adapter_type,
    availabilityState: normalizeProviderAvailabilityState(provider.availability_state),
    available: provider.available,
    availabilityCheckedAt: provider.availability_checked_at ?? null,
    availabilityReason: provider.availability_reason ?? null,
    cliCommand: provider.cli_command,
    cliArgs: [...provider.cli_args],
    authConfig: { ...provider.auth_config },
    modelName: provider.model_name,
    modelTemperature: provider.model_temperature,
    modelMaxTokens: provider.model_max_tokens,
    costPerInputToken: provider.cost_per_input_token,
    costPerOutputToken: provider.cost_per_output_token,
    agentCount: agentItems.filter((agent) => agent.provider_id === provider.id).length,
    isDefault: defaultProviderId === provider.id,
  }))
}

export function buildAgentRows(
  providerItems: AgentProvider[],
  ticketItems: Ticket[],
  agentItems: AgentPayload['agents'],
): AgentInstance[] {
  const ticketMap = new Map(ticketItems.map((ticket) => [ticket.id, ticket]))
  const providerMap = new Map(providerItems.map((provider) => [provider.id, provider]))

  return agentItems.map((agent) => {
    const provider = providerMap.get(agent.provider_id)
    const runtime = agent.runtime ?? null
    const currentTicket = runtime?.current_ticket_id
      ? ticketMap.get(runtime.current_ticket_id)
      : null

    return {
      id: agent.id,
      name: agent.name,
      providerId: agent.provider_id,
      providerName: provider?.name ?? 'Unknown provider',
      modelName: provider?.model_name ?? 'Unknown model',
      status: normalizeAgentStatus(runtime?.status ?? 'idle'),
      runtimePhase: normalizeRuntimePhase(runtime?.runtime_phase ?? 'none'),
      runtimeControlState: normalizeRuntimeControlState(agent.runtime_control_state),
      currentTicket: currentTicket
        ? {
            id: currentTicket.id,
            identifier: currentTicket.identifier,
            title: currentTicket.title,
          }
        : undefined,
      lastHeartbeat: runtime?.last_heartbeat_at ?? null,
      runtimeStartedAt: runtime?.runtime_started_at ?? null,
      sessionId: runtime?.session_id ?? '',
      lastError: runtime?.last_error ?? '',
      currentStepStatus: runtime?.current_step_status ?? undefined,
      currentStepSummary: runtime?.current_step_summary ?? undefined,
      currentStepChangedAt: runtime?.current_step_changed_at ?? null,
      todayCompleted: agent.total_tickets_completed,
      todayCost: 0,
    }
  })
}

export function applyUpdatedProviderState(
  providers: ProviderConfig[],
  agents: AgentInstance[],
  updatedProvider: AgentProvider,
) {
  const nextProviders = providers.map((provider) =>
    provider.id === updatedProvider.id
      ? {
          ...provider,
          machineId: updatedProvider.machine_id,
          machineName: updatedProvider.machine_name,
          machineHost: updatedProvider.machine_host,
          machineStatus: updatedProvider.machine_status,
          machineWorkspaceRoot: updatedProvider.machine_workspace_root ?? null,
          name: updatedProvider.name,
          adapterType: updatedProvider.adapter_type,
          availabilityState: normalizeProviderAvailabilityState(updatedProvider.availability_state),
          available: updatedProvider.available,
          availabilityCheckedAt: updatedProvider.availability_checked_at ?? null,
          availabilityReason: updatedProvider.availability_reason ?? null,
          cliCommand: updatedProvider.cli_command,
          cliArgs: [...updatedProvider.cli_args],
          authConfig: { ...updatedProvider.auth_config },
          modelName: updatedProvider.model_name,
          modelTemperature: updatedProvider.model_temperature,
          modelMaxTokens: updatedProvider.model_max_tokens,
          costPerInputToken: updatedProvider.cost_per_input_token,
          costPerOutputToken: updatedProvider.cost_per_output_token,
        }
      : provider,
  )
  const nextAgents = agents.map((agent) =>
    agent.providerId === updatedProvider.id
      ? {
          ...agent,
          providerName: updatedProvider.name,
          modelName: updatedProvider.model_name,
        }
      : agent,
  )

  return {
    providers: nextProviders,
    agents: nextAgents,
    provider: nextProviders.find((provider) => provider.id === updatedProvider.id) ?? null,
  }
}

function parseProviderAdapterType(raw: string): ProviderAdapterType | null {
  const value = raw.trim().toLowerCase()
  return providerAdapterOptions.some((option) => option.value === value)
    ? (value as ProviderAdapterType)
    : null
}

function splitLines(raw: string): string[] {
  return raw
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)
}

function parseNonNegativeNumber(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  const value = Number(raw.trim())
  if (!Number.isFinite(value) || value < 0) {
    return { ok: false, error: `${label} must be a number greater than or equal to 0.` }
  }
  return { ok: true, value }
}

function parsePositiveInteger(
  label: string,
  raw: string,
): { ok: true; value: number } | { ok: false; error: string } {
  const value = Number(raw.trim())
  if (!Number.isInteger(value) || value <= 0) {
    return { ok: false, error: `${label} must be a positive integer.` }
  }
  return { ok: true, value }
}

function parseAuthConfig(
  raw: string,
): { ok: true; value: Record<string, unknown> } | { ok: false; error: string } {
  const text = raw.trim()
  if (!text) {
    return { ok: true, value: {} }
  }

  try {
    const parsed = JSON.parse(text) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { ok: false, error: 'Auth config must be a JSON object.' }
    }
    return { ok: true, value: parsed as Record<string, unknown> }
  } catch {
    return { ok: false, error: 'Auth config must be valid JSON.' }
  }
}
