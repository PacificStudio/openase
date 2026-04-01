import type { AgentProviderModelCatalogEntry } from '$lib/api/contracts'
import type { ProviderPricingConfig } from './types'

export const TOKENS_PER_MILLION = 1_000_000

export function createCustomFlatPricingConfig(
  inputPerToken: number,
  outputPerToken: number,
): ProviderPricingConfig {
  return {
    source_kind: 'custom',
    pricing_mode: 'flat',
    rates: {
      input_per_token: inputPerToken,
      output_per_token: outputPerToken,
    },
  }
}

export function parseProviderPricingConfig(raw: string): ProviderPricingConfig | null {
  const text = raw.trim()
  if (!text) {
    return null
  }

  try {
    return normalizeProviderPricingConfig(JSON.parse(text) as ProviderPricingConfig)
  } catch {
    return null
  }
}

export function stringifyProviderPricingConfig(config: ProviderPricingConfig | null): string {
  if (!config) {
    return ''
  }
  return JSON.stringify(config)
}

export function normalizeProviderPricingConfig(
  config: ProviderPricingConfig | null | undefined,
): ProviderPricingConfig {
  return {
    version: config?.version,
    source_kind: config?.source_kind,
    pricing_mode: config?.pricing_mode,
    provider: config?.provider,
    model_id: config?.model_id,
    source_url: config?.source_url,
    source_verified_at: config?.source_verified_at,
    default_cache_write_window: config?.default_cache_write_window,
    notes: [...(config?.notes ?? [])],
    rates: config?.rates ? { ...config.rates } : undefined,
    tiers: (config?.tiers ?? []).map((tier) => ({
      label: tier.label,
      max_prompt_tokens: tier.max_prompt_tokens,
      rates: tier.rates ? { ...tier.rates } : undefined,
    })),
  }
}

export function summaryInputPerToken(config: ProviderPricingConfig | null | undefined) {
  const firstTier = config?.tiers?.[0]
  if (firstTier) {
    return firstTier.rates?.input_per_token ?? 0
  }
  return config?.rates?.input_per_token ?? 0
}

export function summaryOutputPerToken(config: ProviderPricingConfig | null | undefined) {
  const firstTier = config?.tiers?.[0]
  if (firstTier) {
    return firstTier.rates?.output_per_token ?? 0
  }
  return config?.rates?.output_per_token ?? 0
}

export function isCustomPricingConfig(config: ProviderPricingConfig | null | undefined) {
  return config?.source_kind === 'custom'
}

export function isRoutedOfficialPricingConfig(config: ProviderPricingConfig | null | undefined) {
  return config?.source_kind === 'official' && config?.pricing_mode === 'routed'
}

export function providerPricingStatusText(config: ProviderPricingConfig | null | undefined) {
  if (!config?.source_kind) {
    return ''
  }
  if (config.source_kind === 'custom') {
    return 'Using custom override'
  }
  if (config.pricing_mode === 'routed') {
    return `Official routed pricing; runtime billing may vary${verifiedSuffix(config)}`
  }
  const providerLabel = config.provider ? titleCase(config.provider) : 'provider'
  return `Official default from ${providerLabel} pricing${verifiedSuffix(config)}`
}

export function providerPricingDetailRows(
  config: ProviderPricingConfig | null | undefined,
): [string, string][] {
  const rates = config?.tiers?.[0]?.rates ?? config?.rates
  if (!rates) {
    return [] as [string, string][]
  }

  const rows: [string, string][] = [
    ['Input', formatPricingPerMillion(rates.input_per_token ?? 0)],
    ['Cached input', formatPricingPerMillion(rates.cached_input_read_per_token ?? 0)],
    ['Cache write 5m', formatPricingPerMillion(rates.cache_write_5m_per_token ?? 0)],
    ['Cache write 1h', formatPricingPerMillion(rates.cache_write_1h_per_token ?? 0)],
    ['Output', formatPricingPerMillion(rates.output_per_token ?? 0)],
    ['Cache storage / hour', formatPricingPerMillion(rates.cache_storage_per_token_hour ?? 0)],
  ]
  return rows.filter(([, value]) => value !== '')
}

export function findBuiltinPricingConfig(
  modelCatalog: AgentProviderModelCatalogEntry[],
  adapterType: string,
  modelName: string,
) {
  const entry = modelCatalog.find((item) => item.adapter_type === adapterType)
  const option = entry?.options?.find((candidate) => candidate.id === modelName)
  return option?.pricing_config ? normalizeProviderPricingConfig(option.pricing_config) : null
}

export function suggestPricingDraftValues(input: {
  modelCatalog: AgentProviderModelCatalogEntry[]
  adapterType: string
  modelName: string
  currentPricingConfig: ProviderPricingConfig | null
  currentCostPerInputToken: string
  currentCostPerOutputToken: string
}) {
  if (isCustomPricingConfig(input.currentPricingConfig)) {
    return null
  }

  const builtinPricingConfig = findBuiltinPricingConfig(
    input.modelCatalog,
    input.adapterType,
    input.modelName,
  )
  if (builtinPricingConfig) {
    return {
      pricingConfig: builtinPricingConfig,
      costPerInputToken: formatPricingPerMillion(summaryInputPerToken(builtinPricingConfig)),
      costPerOutputToken: formatPricingPerMillion(summaryOutputPerToken(builtinPricingConfig)),
    }
  }

  if (input.currentPricingConfig?.source_kind === 'official') {
    return {
      pricingConfig: createCustomFlatPricingConfig(
        parsePricingPerMillion(input.currentCostPerInputToken),
        parsePricingPerMillion(input.currentCostPerOutputToken),
      ),
      costPerInputToken: input.currentCostPerInputToken,
      costPerOutputToken: input.currentCostPerOutputToken,
    }
  }

  return null
}

export function formatPricingPerMillion(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return ''
  }
  return String(value * TOKENS_PER_MILLION)
}

function parsePricingPerMillion(value: string) {
  const parsed = Number(value.trim())
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0
  }
  return parsed / TOKENS_PER_MILLION
}

function verifiedSuffix(config: ProviderPricingConfig) {
  return config.source_verified_at ? ` (verified ${config.source_verified_at})` : ''
}

function titleCase(value: string) {
  if (!value) {
    return value
  }
  return value.charAt(0).toUpperCase() + value.slice(1)
}
