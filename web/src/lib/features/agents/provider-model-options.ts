import type { AgentProviderModelCatalogEntry, AgentProviderModelOption } from '$lib/api/contracts'
import type { ProviderReasoningCapability, ProviderReasoningEffort } from './types'

export const customProviderModelOptionValue = '__custom_model_id__'
export const providerDefaultReasoningValue = '__provider_default_reasoning__'

export function providerModelOptionsForAdapter(
  catalog: AgentProviderModelCatalogEntry[],
  adapterType: string,
): AgentProviderModelOption[] {
  const entry = catalog.find((item) => item.adapter_type === adapterType)
  return entry?.options ? [...entry.options] : []
}

export function recommendedProviderModelId(
  catalog: AgentProviderModelCatalogEntry[],
  adapterType: string,
): string {
  const options = providerModelOptionsForAdapter(catalog, adapterType)
  return options.find((option) => option.recommended)?.id ?? options[0]?.id ?? ''
}

export function splitProviderModelSelection(
  catalog: AgentProviderModelCatalogEntry[],
  adapterType: string,
  modelName: string,
  preserveUnknownModel: boolean,
): { baseModelId: string; customModelId: string } {
  const normalizedModelName = modelName.trim()
  const options = providerModelOptionsForAdapter(catalog, adapterType)

  if (options.some((option) => option.id === normalizedModelName)) {
    return {
      baseModelId: normalizedModelName,
      customModelId: '',
    }
  }

  const baseModelId = recommendedProviderModelId(catalog, adapterType)

  return {
    baseModelId,
    customModelId: preserveUnknownModel ? normalizedModelName : '',
  }
}

export function providerModelReasoningCapability(
  catalog: AgentProviderModelCatalogEntry[],
  adapterType: string,
  modelName: string,
): ProviderReasoningCapability | null {
  const normalizedModelName = modelName.trim()
  const options = providerModelOptionsForAdapter(catalog, adapterType)
  const option = options.find((item) => item.id === normalizedModelName)

  if (option?.reasoning) {
    return normalizeReasoningCapability(option.reasoning)
  }
  if (normalizedModelName && options.length > 0) {
    return {
      state: 'unsupported',
      reason: 'unknown_model',
      supportedEfforts: [],
      defaultEffort: null,
      supportsProviderPreset: false,
      supportsModelOverride: false,
    }
  }
  if (options.length === 0) {
    return {
      state: 'unsupported',
      reason: 'reasoning_unsupported',
      supportedEfforts: [],
      defaultEffort: null,
      supportsProviderPreset: false,
      supportsModelOverride: false,
    }
  }
  return null
}

export function formatReasoningEffortLabel(value: string) {
  switch (value) {
    case 'xhigh':
      return 'Extra high'
    case 'max':
      return 'Max'
    case 'minimal':
      return 'Minimal'
    case 'low':
      return 'Low'
    case 'medium':
      return 'Medium'
    case 'high':
      return 'High'
    default:
      return value
  }
}

export function providerReasoningCapabilitySummary(
  capability: ProviderReasoningCapability | null | undefined,
) {
  if (!capability) {
    return ''
  }

  switch (capability.reason) {
    case 'unknown_model':
      return 'Reasoning presets are only available for built-in models in the catalog.'
    case 'reasoning_unsupported':
      return 'Reasoning presets are unavailable for this model or adapter.'
    default:
      if (capability.state === 'available') {
        const efforts = capability.supportedEfforts ?? []
        if (efforts.length === 0) {
          return 'Reasoning preset is available.'
        }
        return `Supported presets: ${efforts.map((effort) => formatReasoningEffortLabel(effort)).join(', ')}.`
      }
      return 'Reasoning preset is unavailable for this selection.'
  }
}

function normalizeReasoningCapability(
  value: NonNullable<AgentProviderModelOption['reasoning']>,
): ProviderReasoningCapability {
  const defaultEffort = value.default_effort
  const normalizedDefaultEffort = isProviderReasoningEffort(defaultEffort ?? '')
    ? (defaultEffort as ProviderReasoningEffort)
    : null
  return {
    state:
      value.state === 'available' || value.state === 'unavailable' || value.state === 'unsupported'
        ? value.state
        : 'unsupported',
    reason: value.reason ?? null,
    supportedEfforts: (value.supported_efforts ?? []).filter(isProviderReasoningEffort),
    defaultEffort: normalizedDefaultEffort,
    supportsProviderPreset: value.supports_provider_preset ?? false,
    supportsModelOverride: value.supports_model_override ?? false,
  }
}

function isProviderReasoningEffort(value: string): value is ProviderReasoningEffort {
  return (
    value === 'minimal' ||
    value === 'low' ||
    value === 'medium' ||
    value === 'high' ||
    value === 'xhigh' ||
    value === 'max'
  )
}
