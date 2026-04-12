import type { AgentProvider } from '$lib/api/contracts'
import { providersT } from './i18n'
import { formatCodexUsedPercent, formatObservedAt } from './provider-rate-limit-core'
import type {
  ProviderRateLimitSnapshot,
  ProviderRateLimitSummary,
  ProviderRateLimitView,
  RateLimitWindow,
} from './provider-rate-limit-core'

export type { ProviderRateLimitSummary } from './provider-rate-limit-core'

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
      detailParts.push(
        providersT('providers.rateLimit.resetsAt', {
          time: new Date(snapshot.resetsAt).toLocaleString(),
        }),
      )
    }
    if (snapshot.surpassedThreshold != null) {
      detailParts.push(
        providersT('providers.rateLimit.warningThreshold', {
          percent: (snapshot.surpassedThreshold * 100).toFixed(0),
        }),
      )
    }
    if (snapshot.isUsingOverage) {
      detailParts.push(providersT('providers.rateLimit.usingOverage'))
    } else if (snapshot.overageStatus) {
      detailParts.push(
        providersT('providers.rateLimit.overageStatus', { status: snapshot.overageStatus }),
      )
    }

    const windows: RateLimitWindow[] = []
    if (snapshot.utilization != null) {
      windows.push({
        label: snapshot.rateLimitType || providersT('providers.rateLimit.window'),
        usedPercent: snapshot.utilization * 100,
        resetsAt: snapshot.resetsAt,
      })
    }

    return {
      headline,
      detail: detailParts.join(' · ') || providersT('providers.rateLimit.claudeSnapshot'),
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
      detailParts.push(
        providersT('providers.rateLimit.primaryResets', {
          time: new Date(primary.resetsAt).toLocaleString(),
        }),
      )
    }
    if (snapshot.secondary?.resetsAt) {
      detailParts.push(
        providersT('providers.rateLimit.secondaryResets', {
          time: new Date(snapshot.secondary.resetsAt).toLocaleString(),
        }),
      )
    }

    const windows: RateLimitWindow[] = []
    if (primary?.usedPercent != null) {
      const pct = primary.usedPercent <= 1 ? primary.usedPercent * 100 : primary.usedPercent
      windows.push({
        label: providersT('providers.rateLimit.primaryLabel'),
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
        label: providersT('providers.rateLimit.secondaryLabel'),
        usedPercent: pct,
        windowMinutes: snapshot.secondary.windowMinutes,
        resetsAt: snapshot.secondary.resetsAt,
      })
    }

    return {
      headline:
        headlineParts.join(' · ') ||
        snapshot.limitId ||
        providersT('providers.rateLimit.codexHeadline'),
      detail: detailParts.join(' · ') || providersT('providers.rateLimit.codexSnapshot'),
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
      detailParts.push(
        providersT('providers.rateLimit.resetsAt', {
          time: new Date(snapshot.resetTime).toLocaleString(),
        }),
      )
    } else if (matchingBucket?.resetTime) {
      detailParts.push(
        providersT('providers.rateLimit.resetsAt', {
          time: new Date(matchingBucket.resetTime).toLocaleString(),
        }),
      )
    }
    if (matchingBucket?.modelId) {
      detailParts.push(matchingBucket.modelId)
    }

    const windows: RateLimitWindow[] = []
    if (geminiUsedPercent != null) {
      windows.push({
        label: matchingBucket?.modelId || providersT('providers.rateLimit.quotaLabel'),
        usedPercent: geminiUsedPercent,
        resetsAt: snapshot.resetTime || matchingBucket?.resetTime,
      })
    }

    return {
      headline,
      detail: detailParts.join(' · ') || providersT('providers.rateLimit.geminiSnapshot'),
      updatedLabel,
      windows,
    }
  }

  return {
    headline: provider.cliRateLimit.provider || providersT('providers.rateLimit.defaultHeadline'),
    detail: providersT('providers.rateLimit.defaultSnapshot'),
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
    cliRateLimit: mapSnapshot(provider.cli_rate_limit),
  })
}

function mapSnapshot(raw: NonNullable<AgentProvider['cli_rate_limit']>): ProviderRateLimitSnapshot {
  return {
    provider: raw.provider ?? '',
    raw: { ...(raw.raw ?? {}) },
    claudeCode: raw.claude_code
      ? {
          status: raw.claude_code.status ?? '',
          rateLimitType: raw.claude_code.rate_limit_type ?? null,
          resetsAt: raw.claude_code.resets_at ?? null,
          utilization: raw.claude_code.utilization ?? null,
          surpassedThreshold: raw.claude_code.surpassed_threshold ?? null,
          overageStatus: raw.claude_code.overage_status ?? null,
          overageDisabledReason: raw.claude_code.overage_disabled_reason ?? null,
          isUsingOverage: raw.claude_code.is_using_overage ?? null,
        }
      : null,
    codex: raw.codex
      ? {
          limitId: raw.codex.limit_id ?? null,
          limitName: raw.codex.limit_name ?? null,
          planType: raw.codex.plan_type ?? null,
          primary: raw.codex.primary
            ? {
                usedPercent: raw.codex.primary.used_percent ?? null,
                windowMinutes: raw.codex.primary.window_minutes ?? null,
                resetsAt: raw.codex.primary.resets_at ?? null,
              }
            : null,
          secondary: raw.codex.secondary
            ? {
                usedPercent: raw.codex.secondary.used_percent ?? null,
                windowMinutes: raw.codex.secondary.window_minutes ?? null,
                resetsAt: raw.codex.secondary.resets_at ?? null,
              }
            : null,
        }
      : null,
    gemini: raw.gemini
      ? {
          authType: raw.gemini.auth_type ?? null,
          remaining: raw.gemini.remaining ?? null,
          limit: raw.gemini.limit ?? null,
          resetTime: raw.gemini.reset_time ?? null,
          buckets: (raw.gemini.buckets ?? []).map((bucket) => ({
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
