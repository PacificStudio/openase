import { formatRelativeTime } from '$lib/utils'
import type { AgentProvider } from '$lib/api/contracts'

export type ProviderRateLimitSnapshot = {
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

type ProviderRateLimitView = {
  adapterType: string
  modelName?: string
  cliRateLimit?: ProviderRateLimitSnapshot | null
  cliRateLimitUpdatedAt?: string | null
}

export type RateLimitWindow = {
  label: string
  usedPercent: number
  windowMinutes?: number | null
  resetsAt?: string | null
}

export type ProviderRateLimitSummary = {
  headline: string
  detail: string
  updatedLabel: string
  windows: RateLimitWindow[]
  planType?: string | null
}

function formatObservedAt(value: string | null | undefined): string {
  if (!value) {
    return 'Update time unavailable'
  }

  const observedAt = new Date(value)
  if (Number.isNaN(observedAt.getTime())) {
    return 'Update time unavailable'
  }

  return `Updated ${formatRelativeTime(value)} · ${observedAt.toLocaleString()}`
}

function formatCodexUsedPercent(value: number | null | undefined): string | null {
  if (value == null || Number.isNaN(value)) {
    return null
  }

  const normalizedValue = value <= 1 ? value * 100 : value
  return `${normalizedValue.toFixed(1)}% used`
}

export function summarizeProviderRateLimit(
  provider: ProviderRateLimitView,
): ProviderRateLimitSummary | null {
  if (!provider.cliRateLimit) {
    return null
  }

  const updatedLabel = formatObservedAt(provider.cliRateLimitUpdatedAt)

  if (provider.adapterType === 'claude-code-cli' && provider.cliRateLimit.claudeCode) {
    const snapshot = provider.cliRateLimit.claudeCode
    const headline = [snapshot.status || 'unknown', snapshot.rateLimitType || 'window unknown']
      .filter(Boolean)
      .join(' · ')
    const detailParts = []
    if (snapshot.resetsAt) {
      detailParts.push(`Resets ${new Date(snapshot.resetsAt).toLocaleString()}`)
    }
    if (snapshot.surpassedThreshold != null) {
      detailParts.push(`${(snapshot.surpassedThreshold * 100).toFixed(0)}% warning threshold`)
    }
    if (snapshot.isUsingOverage) {
      detailParts.push('Using overage')
    } else if (snapshot.overageStatus) {
      detailParts.push(`Overage ${snapshot.overageStatus}`)
    }

    const windows: RateLimitWindow[] = []
    if (snapshot.utilization != null) {
      windows.push({
        label: snapshot.rateLimitType || 'Window',
        usedPercent: snapshot.utilization * 100,
        resetsAt: snapshot.resetsAt,
      })
    }

    return {
      headline,
      detail: detailParts.join(' · ') || 'Claude rate limit snapshot captured.',
      updatedLabel,
      windows,
    }
  }

  if (provider.adapterType === 'codex-app-server' && provider.cliRateLimit.codex) {
    const snapshot = provider.cliRateLimit.codex
    const primary = snapshot.primary
    const headlineParts = []
    const primaryUsedPercent = formatCodexUsedPercent(primary?.usedPercent)
    if (primaryUsedPercent) {
      headlineParts.push(primaryUsedPercent)
    }
    if (primary?.windowMinutes != null) {
      headlineParts.push(`${primary.windowMinutes}m window`)
    }
    if (snapshot.planType) {
      headlineParts.push(snapshot.planType)
    }

    const detailParts = []
    if (primary?.resetsAt) {
      detailParts.push(`Primary resets ${new Date(primary.resetsAt).toLocaleString()}`)
    }
    if (snapshot.secondary?.resetsAt) {
      detailParts.push(`Secondary resets ${new Date(snapshot.secondary.resetsAt).toLocaleString()}`)
    }

    const windows: RateLimitWindow[] = []
    if (primary?.usedPercent != null) {
      const pct = primary.usedPercent <= 1 ? primary.usedPercent * 100 : primary.usedPercent
      windows.push({
        label: 'Primary',
        usedPercent: pct,
        windowMinutes: primary.windowMinutes,
        resetsAt: primary.resetsAt,
      })
    }
    if (snapshot.secondary?.usedPercent != null) {
      const pct =
        snapshot.secondary.usedPercent <= 1
          ? snapshot.secondary.usedPercent * 100
          : snapshot.secondary.usedPercent
      windows.push({
        label: 'Secondary',
        usedPercent: pct,
        windowMinutes: snapshot.secondary.windowMinutes,
        resetsAt: snapshot.secondary.resetsAt,
      })
    }

    return {
      headline: headlineParts.join(' · ') || snapshot.limitId || 'codex rate limit',
      detail: detailParts.join(' · ') || 'Codex rate limit snapshot captured.',
      updatedLabel,
      windows,
      planType: snapshot.planType,
    }
  }

  if (provider.adapterType === 'gemini-cli' && provider.cliRateLimit.gemini) {
    const snapshot = provider.cliRateLimit.gemini
    const matchingBucket =
      snapshot.buckets?.find((bucket) => bucket.modelId === provider.modelName) ??
      snapshot.buckets?.[0] ??
      null

    let headline = snapshot.authType || 'gemini rate limit'
    let geminiUsedPercent: number | null = null
    if (snapshot.remaining != null && snapshot.limit != null) {
      headline = `${snapshot.remaining}/${snapshot.limit} remaining`
      if (snapshot.limit > 0) {
        geminiUsedPercent = ((snapshot.limit - snapshot.remaining) / snapshot.limit) * 100
      }
    } else if (matchingBucket?.remainingFraction != null) {
      headline = `${(matchingBucket.remainingFraction * 100).toFixed(1)}% remaining`
      geminiUsedPercent = (1 - matchingBucket.remainingFraction) * 100
    }

    const detailParts = []
    if (snapshot.resetTime) {
      detailParts.push(`Resets ${new Date(snapshot.resetTime).toLocaleString()}`)
    } else if (matchingBucket?.resetTime) {
      detailParts.push(`Resets ${new Date(matchingBucket.resetTime).toLocaleString()}`)
    }
    if (matchingBucket?.modelId) {
      detailParts.push(matchingBucket.modelId)
    }

    const windows: RateLimitWindow[] = []
    if (geminiUsedPercent != null) {
      windows.push({
        label: matchingBucket?.modelId || 'Quota',
        usedPercent: geminiUsedPercent,
        resetsAt: snapshot.resetTime || matchingBucket?.resetTime,
      })
    }

    return {
      headline,
      detail: detailParts.join(' · ') || 'Gemini quota snapshot captured.',
      updatedLabel,
      windows,
    }
  }

  return {
    headline: provider.cliRateLimit.provider || 'rate limit',
    detail: 'Provider-specific rate limit snapshot captured.',
    updatedLabel,
    windows: [],
  }
}

export function summarizeAgentProviderRateLimit(
  provider: Pick<
    AgentProvider,
    'adapter_type' | 'model_name' | 'cli_rate_limit' | 'cli_rate_limit_updated_at'
  >,
): ProviderRateLimitSummary | null {
  if (!provider.cli_rate_limit) {
    return null
  }

  return summarizeProviderRateLimit({
    adapterType: provider.adapter_type,
    modelName: provider.model_name,
    cliRateLimitUpdatedAt: provider.cli_rate_limit_updated_at ?? null,
    cliRateLimit: {
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
    },
  })
}
