import type { Agent, AgentProvider } from '$lib/api/contracts'

export type ProviderOption = {
  id: string
  name: string
  adapterType: string
  modelName: string
  available: boolean
  agentCount: number
}

export type GovernanceAgent = {
  id: string
  name: string
  providerName: string
  status: 'idle' | 'claimed' | 'running' | 'paused' | 'failed' | 'terminated'
  runtimePhase: 'none' | 'launching' | 'ready' | 'failed'
  workspacePath: string
  lastHeartbeat?: string | null
}

export type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export const governanceAgentStatusLabels: Record<GovernanceAgent['status'], string> = {
  idle: 'Idle',
  claimed: 'Claimed',
  running: 'Running',
  paused: 'Paused',
  failed: 'Failed',
  terminated: 'Terminated',
}

export const governanceAgentStatusClasses: Record<GovernanceAgent['status'], string> = {
  idle: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  claimed: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300',
  running: 'border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300',
  paused: 'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300',
  failed: 'border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300',
  terminated: 'border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-300',
}

export function buildProviderOptions(
  providerItems: AgentProvider[],
  agentItems: Agent[],
): ProviderOption[] {
  return providerItems.map((provider) => ({
    id: provider.id,
    name: provider.name,
    adapterType: provider.adapter_type,
    modelName: provider.model_name,
    available: provider.available,
    agentCount: agentItems.filter((agent) => agent.provider_id === provider.id).length,
  }))
}

export function buildGovernanceAgents(
  agentItems: Agent[],
  providerItems: AgentProvider[],
): GovernanceAgent[] {
  const providerMap = new Map(providerItems.map((provider) => [provider.id, provider]))

  return agentItems
    .map((agent) => {
      const provider = providerMap.get(agent.provider_id)

      return {
        id: agent.id,
        name: agent.name,
        providerName: provider?.name ?? 'Unknown provider',
        status: normalizeAgentStatus(agent.runtime?.status ?? 'idle'),
        runtimePhase: normalizeRuntimePhase(agent.runtime?.runtime_phase ?? 'none'),
        workspacePath: agent.workspace_path ?? '',
        lastHeartbeat: agent.runtime?.last_heartbeat_at ?? null,
      }
    })
    .sort((left, right) => left.name.localeCompare(right.name))
}

export function parseDefaultProviderSelection(
  rawProviderId: string,
  availableProviders: ProviderOption[],
): ParseResult<string | null> {
  if (!rawProviderId) {
    return { ok: true, value: null }
  }

  if (availableProviders.some((provider) => provider.id === rawProviderId)) {
    return { ok: true, value: rawProviderId }
  }

  return { ok: false, error: 'Selected provider is no longer available.' }
}

function normalizeAgentStatus(status: string): GovernanceAgent['status'] {
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

function normalizeRuntimePhase(runtimePhase: string): GovernanceAgent['runtimePhase'] {
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
