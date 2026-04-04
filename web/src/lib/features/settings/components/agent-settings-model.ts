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
          provider: provider.cli_rate_limit.provider ?? '',
          raw: { ...(provider.cli_rate_limit.raw ?? {}) },
          claudeCode: provider.cli_rate_limit.claude_code
            ? {
                status: provider.cli_rate_limit.claude_code.status ?? '',
                rateLimitType: provider.cli_rate_limit.claude_code.rate_limit_type ?? null,
                resetsAt: provider.cli_rate_limit.claude_code.resets_at ?? null,
                utilization: provider.cli_rate_limit.claude_code.utilization ?? null,
                surpassedThreshold: provider.cli_rate_limit.claude_code.surpassed_threshold ?? null,
                overageStatus: provider.cli_rate_limit.claude_code.overage_status ?? null,
                overageDisabledReason:
                  provider.cli_rate_limit.claude_code.overage_disabled_reason ?? null,
                isUsingOverage: provider.cli_rate_limit.claude_code.is_using_overage ?? null,
              }
            : null,
          codex: provider.cli_rate_limit.codex
            ? {
                limitId: provider.cli_rate_limit.codex.limit_id ?? null,
                limitName: provider.cli_rate_limit.codex.limit_name ?? null,
                planType: provider.cli_rate_limit.codex.plan_type ?? null,
                primary: provider.cli_rate_limit.codex.primary
                  ? {
                      usedPercent: provider.cli_rate_limit.codex.primary.used_percent ?? null,
                      windowMinutes: provider.cli_rate_limit.codex.primary.window_minutes ?? null,
                      resetsAt: provider.cli_rate_limit.codex.primary.resets_at ?? null,
                    }
                  : null,
                secondary: provider.cli_rate_limit.codex.secondary
                  ? {
                      usedPercent: provider.cli_rate_limit.codex.secondary.used_percent ?? null,
                      windowMinutes: provider.cli_rate_limit.codex.secondary.window_minutes ?? null,
                      resetsAt: provider.cli_rate_limit.codex.secondary.resets_at ?? null,
                    }
                  : null,
              }
            : null,
          gemini: provider.cli_rate_limit.gemini
            ? {
                authType: provider.cli_rate_limit.gemini.auth_type ?? null,
                remaining: provider.cli_rate_limit.gemini.remaining ?? null,
                limit: provider.cli_rate_limit.gemini.limit ?? null,
                resetTime: provider.cli_rate_limit.gemini.reset_time ?? null,
                buckets: (provider.cli_rate_limit.gemini.buckets ?? []).map((bucket) => ({
                  modelId: bucket.model_id ?? null,
                  tokenType: bucket.token_type ?? null,
                  remainingAmount: bucket.remaining_amount ?? null,
                  remainingFraction: bucket.remaining_fraction ?? null,
                  resetTime: bucket.reset_time ?? null,
                })),
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
