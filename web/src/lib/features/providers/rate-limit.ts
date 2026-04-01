import { formatRelativeTime } from '$lib/utils'

export type ProviderRateLimitSnapshot = {
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
}

type ProviderRateLimitView = {
  adapterType: string
  cliRateLimit?: ProviderRateLimitSnapshot | null
  cliRateLimitUpdatedAt?: string | null
}

export type ProviderRateLimitSummary = {
  headline: string
  detail: string
  updatedLabel: string
}

export function summarizeProviderRateLimit(
  provider: ProviderRateLimitView,
): ProviderRateLimitSummary | null {
  if (!provider.cliRateLimit) {
    return null
  }

  const updatedLabel = provider.cliRateLimitUpdatedAt
    ? `Updated ${formatRelativeTime(provider.cliRateLimitUpdatedAt)}`
    : 'Update time unavailable'

  if (provider.adapterType === 'claude-code-cli' && provider.cliRateLimit.claudeCode) {
    const snapshot = provider.cliRateLimit.claudeCode
    const headline = [snapshot.status || 'unknown', snapshot.rateLimitType || 'window unknown']
      .filter(Boolean)
      .join(' · ')
    const detailParts = []
    if (snapshot.resetsAt) {
      detailParts.push(`Resets ${new Date(snapshot.resetsAt).toLocaleString()}`)
    }
    if (snapshot.isUsingOverage) {
      detailParts.push('Using overage')
    } else if (snapshot.overageStatus) {
      detailParts.push(`Overage ${snapshot.overageStatus}`)
    }

    return {
      headline,
      detail: detailParts.join(' · ') || 'Claude rate limit snapshot captured.',
      updatedLabel,
    }
  }

  return {
    headline: provider.cliRateLimit.provider || 'rate limit',
    detail: 'Provider-specific rate limit snapshot captured.',
    updatedLabel,
  }
}
