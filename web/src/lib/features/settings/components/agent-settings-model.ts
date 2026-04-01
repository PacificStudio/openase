import type { Agent, AgentProvider } from '$lib/api/contracts'
import { normalizeProviderAvailabilityState } from '$lib/features/providers'

export type ProviderOption = {
  id: string
  name: string
  machineName: string
  adapterType: string
  modelName: string
  availabilityState: string
  available: boolean
  availabilityCheckedAt?: string | null
  availabilityReason?: string | null
  cliRateLimit?: {
    provider: string
    raw: Record<string, unknown>
    claudeCode?: {
      status: string
      rateLimitType?: string | null
      resetsAt?: string | null
      overageStatus?: string | null
      overageDisabledReason?: string | null
      isUsingOverage?: boolean | null
    } | null
  } | null
  cliRateLimitUpdatedAt?: string | null
  agentCount: number
}

export type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export function buildProviderOptions(
  providerItems: AgentProvider[],
  agentItems: Agent[],
): ProviderOption[] {
  return providerItems.map((provider) => ({
    id: provider.id,
    name: provider.name,
    machineName: provider.machine_name,
    adapterType: provider.adapter_type,
    availabilityState: normalizeProviderAvailabilityState(provider.availability_state),
    modelName: provider.model_name,
    available: provider.available,
    availabilityCheckedAt: provider.availability_checked_at ?? null,
    availabilityReason: provider.availability_reason ?? null,
    cliRateLimit: provider.cli_rate_limit
      ? {
          provider: provider.cli_rate_limit.provider,
          raw: { ...(provider.cli_rate_limit.raw ?? {}) },
          claudeCode: provider.cli_rate_limit.claude_code
            ? {
                status: provider.cli_rate_limit.claude_code.status,
                rateLimitType: provider.cli_rate_limit.claude_code.rate_limit_type ?? null,
                resetsAt: provider.cli_rate_limit.claude_code.resets_at ?? null,
                overageStatus: provider.cli_rate_limit.claude_code.overage_status ?? null,
                overageDisabledReason:
                  provider.cli_rate_limit.claude_code.overage_disabled_reason ?? null,
                isUsingOverage: provider.cli_rate_limit.claude_code.is_using_overage ?? null,
              }
            : null,
        }
      : null,
    cliRateLimitUpdatedAt: provider.cli_rate_limit_updated_at ?? null,
    agentCount: agentItems.filter((agent) => agent.provider_id === provider.id).length,
  }))
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
