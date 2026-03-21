import type { AgentPayload, AgentProvider, Ticket } from '$lib/api/contracts'
import type {
  AgentInstance,
  ProviderDraft,
  ProviderDraftParseResult,
  ProviderConfig,
  ProviderAdapterType,
} from './types'

export const providerAdapterOptions: Array<{ value: ProviderAdapterType; label: string }> = [
  { value: 'claude-code-cli', label: 'Claude Code CLI' },
  { value: 'codex-app-server', label: 'Codex App Server' },
  { value: 'gemini-cli', label: 'Gemini CLI' },
  { value: 'custom', label: 'Custom' },
]

export function createEmptyProviderDraft(): ProviderDraft {
  return {
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
    name: provider.name,
    adapterType: provider.adapter_type,
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
    const currentTicket = agent.current_ticket_id ? ticketMap.get(agent.current_ticket_id) : null

    return {
      id: agent.id,
      name: agent.name,
      providerId: agent.provider_id,
      providerName: provider?.name ?? 'Unknown provider',
      modelName: provider?.model_name ?? 'Unknown model',
      status: normalizeAgentStatus(agent.status),
      runtimePhase: normalizeRuntimePhase(agent.runtime_phase),
      currentTicket: currentTicket
        ? {
            id: currentTicket.id,
            identifier: currentTicket.identifier,
            title: currentTicket.title,
          }
        : undefined,
      lastHeartbeat: agent.last_heartbeat_at,
      todayCompleted: agent.total_tickets_completed,
      todayCost: 0,
      capabilities: agent.capabilities,
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
          name: updatedProvider.name,
          adapterType: updatedProvider.adapter_type,
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

function normalizeAgentStatus(status: string): AgentInstance['status'] {
  if (
    status === 'idle' ||
    status === 'claimed' ||
    status === 'running' ||
    status === 'paused' ||
    status === 'failed' ||
    status === 'terminated'
  ) {
    return status
  }

  return status === 'active' ? 'running' : 'idle'
}

function normalizeRuntimePhase(runtimePhase: string): AgentInstance['runtimePhase'] {
  if (
    runtimePhase === 'none' ||
    runtimePhase === 'launching' ||
    runtimePhase === 'ready' ||
    runtimePhase === 'failed'
  ) {
    return runtimePhase
  }

  return 'none'
}
