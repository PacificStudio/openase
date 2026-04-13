import { formatRelativeTime } from '$lib/utils'
import { providersT } from './i18n'

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

export type ProviderRateLimitView = {
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

export function formatObservedAt(value: string | null | undefined): string {
  if (!value) {
    return providersT('providers.rateLimit.updateTimeUnavailable')
  }

  const observedAt = new Date(value)
  if (Number.isNaN(observedAt.getTime())) {
    return providersT('providers.rateLimit.updateTimeUnavailable')
  }

  return providersT('providers.rateLimit.updated', {
    relativeTime: formatRelativeTime(value),
    timestamp: observedAt.toLocaleString(),
  })
}

export function formatCodexUsedPercent(value: number | null | undefined): string | null {
  if (value == null || Number.isNaN(value)) {
    return null
  }

  const normalizedValue = value <= 1 ? value * 100 : value
  return providersT('providers.rateLimit.percentUsed', { percent: normalizedValue.toFixed(1) })
}
