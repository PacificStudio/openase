import type { AgentProvider } from '$lib/api/contracts'
import type {
  ProviderCLIRateLimit,
  ProviderReasoningCapability,
  ProviderReasoningEffort,
} from './types'

export function normalizeProviderCLIRateLimit(
  value: AgentProvider['cli_rate_limit'] | null | undefined,
): ProviderCLIRateLimit | null {
  if (!value) {
    return null
  }

  return {
    provider: value.provider ?? '',
    raw: { ...(value.raw ?? {}) },
    claudeCode: value.claude_code
      ? {
          status: value.claude_code.status ?? '',
          rateLimitType: value.claude_code.rate_limit_type ?? null,
          resetsAt: value.claude_code.resets_at ?? null,
          utilization: value.claude_code.utilization ?? null,
          surpassedThreshold: value.claude_code.surpassed_threshold ?? null,
          overageStatus: value.claude_code.overage_status ?? null,
          overageDisabledReason: value.claude_code.overage_disabled_reason ?? null,
          isUsingOverage: value.claude_code.is_using_overage ?? null,
        }
      : null,
    codex: value.codex
      ? {
          limitId: value.codex.limit_id ?? null,
          limitName: value.codex.limit_name ?? null,
          planType: value.codex.plan_type ?? null,
          primary: value.codex.primary
            ? {
                usedPercent: value.codex.primary.used_percent ?? null,
                windowMinutes: value.codex.primary.window_minutes ?? null,
                resetsAt: value.codex.primary.resets_at ?? null,
              }
            : null,
          secondary: value.codex.secondary
            ? {
                usedPercent: value.codex.secondary.used_percent ?? null,
                windowMinutes: value.codex.secondary.window_minutes ?? null,
                resetsAt: value.codex.secondary.resets_at ?? null,
              }
            : null,
        }
      : null,
    gemini: value.gemini
      ? {
          authType: value.gemini.auth_type ?? null,
          remaining: value.gemini.remaining ?? null,
          limit: value.gemini.limit ?? null,
          resetTime: value.gemini.reset_time ?? null,
          buckets: (value.gemini.buckets ?? []).map((bucket) => ({
            modelId: bucket.model_id ?? null,
            tokenType: bucket.token_type ?? null,
            remainingAmount: bucket.remaining_amount ?? null,
            remainingFraction: bucket.remaining_fraction ?? null,
            resetTime: bucket.reset_time ?? null,
          })),
        }
      : null,
  }
}

export function normalizeProviderReasoningCapability(
  value: AgentProvider['capabilities']['reasoning'] | null | undefined,
): ProviderReasoningCapability | null {
  if (!value) {
    return null
  }

  return {
    state: normalizeProviderCapabilityState(value.state),
    reason: value.reason ?? null,
    supportedEfforts: (value.supported_efforts ?? [])
      .map((effort) => normalizeProviderReasoningEffort(effort))
      .filter((effort): effort is ProviderReasoningEffort => effort !== null),
    defaultEffort: normalizeProviderReasoningEffort(value.default_effort ?? null),
    selectedEffort: normalizeProviderReasoningEffort(value.selected_effort ?? null),
    effectiveEffort: normalizeProviderReasoningEffort(value.effective_effort ?? null),
    supportsProviderPreset: value.supports_provider_preset ?? false,
    supportsModelOverride: value.supports_model_override ?? false,
  }
}

export function normalizeProviderReasoningEffort(
  value: string | null | undefined,
): ProviderReasoningEffort | null {
  switch (value) {
    case 'minimal':
    case 'low':
    case 'medium':
    case 'high':
    case 'xhigh':
    case 'max':
      return value
    default:
      return null
  }
}

function normalizeProviderCapabilityState(value: string | null | undefined) {
  switch (value) {
    case 'available':
    case 'unavailable':
    case 'unsupported':
      return value
    default:
      return 'unsupported'
  }
}
