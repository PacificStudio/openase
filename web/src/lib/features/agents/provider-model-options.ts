import type { AgentProviderModelCatalogEntry, AgentProviderModelOption } from '$lib/api/contracts'

export const customProviderModelOptionValue = '__custom_model_id__'

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
