import { describe, expect, it } from 'vitest'

import type { AgentProviderModelCatalogEntry } from '$lib/api/contracts'
import { createCustomFlatPricingConfig, suggestPricingDraftValues } from './provider-pricing'

const providerModelCatalogFixture: AgentProviderModelCatalogEntry[] = [
  {
    adapter_type: 'codex-app-server',
    options: [
      {
        id: 'gpt-5.5',
        label: 'gpt-5.5',
        description: 'Default flagship model for complex reasoning and coding.',
        recommended: true,
        preview: false,
        reasoning: null,
        pricing_config: {
          source_kind: 'official',
          pricing_mode: 'flat',
          provider: 'openai',
          model_id: 'gpt-5.5',
          source_url: 'https://openai.com/api/pricing/',
          source_verified_at: '2026-04-29',
          rates: {
            input_per_token: 0.000005,
            output_per_token: 0.00003,
            cached_input_read_per_token: 0.0000005,
          },
          notes: [],
          tiers: [],
          version: '2026-04-29',
        },
      },
    ],
  },
  {
    adapter_type: 'gemini-cli',
    options: [
      {
        id: 'auto-gemini-2.5',
        label: 'Auto (Gemini 2.5)',
        description: 'Let Gemini CLI route requests.',
        recommended: true,
        preview: false,
        reasoning: null,
        pricing_config: {
          source_kind: 'official',
          pricing_mode: 'routed',
          provider: 'google',
          model_id: 'auto-gemini-2.5',
          source_url: 'https://ai.google.dev/gemini-api/docs/pricing',
          source_verified_at: '2026-04-01',
          notes: [],
          rates: {},
          tiers: [],
          version: '2026-04-01',
        },
      },
    ],
  },
]

describe('provider pricing draft suggestions', () => {
  it('auto-fills official builtin pricing for a known model', () => {
    expect(
      suggestPricingDraftValues({
        modelCatalog: providerModelCatalogFixture,
        adapterType: 'codex-app-server',
        modelName: 'gpt-5.5',
        currentPricingConfig: null,
        currentCostPerInputToken: '0',
        currentCostPerOutputToken: '0',
      }),
    ).toEqual({
      pricingConfig: expect.objectContaining({
        source_kind: 'official',
        model_id: 'gpt-5.5',
      }),
      costPerInputToken: '5',
      costPerOutputToken: '30',
    })
  })

  it('preserves a custom override instead of auto-filling again', () => {
    expect(
      suggestPricingDraftValues({
        modelCatalog: providerModelCatalogFixture,
        adapterType: 'codex-app-server',
        modelName: 'gpt-5.5',
        currentPricingConfig: createCustomFlatPricingConfig(0.000003, 0.000015),
        currentCostPerInputToken: '3',
        currentCostPerOutputToken: '15',
      }),
    ).toBeNull()
  })

  it('does not fake flat prices for routed models', () => {
    expect(
      suggestPricingDraftValues({
        modelCatalog: providerModelCatalogFixture,
        adapterType: 'gemini-cli',
        modelName: 'auto-gemini-2.5',
        currentPricingConfig: null,
        currentCostPerInputToken: '',
        currentCostPerOutputToken: '',
      }),
    ).toEqual({
      pricingConfig: expect.objectContaining({
        pricing_mode: 'routed',
        model_id: 'auto-gemini-2.5',
      }),
      costPerInputToken: '',
      costPerOutputToken: '',
    })
  })

  it('drops official provenance when a custom model override no longer matches a builtin', () => {
    expect(
      suggestPricingDraftValues({
        modelCatalog: providerModelCatalogFixture,
        adapterType: 'codex-app-server',
        modelName: 'gpt-5.5-experimental',
        currentPricingConfig: providerModelCatalogFixture[0].options[0].pricing_config,
        currentCostPerInputToken: '5',
        currentCostPerOutputToken: '30',
      }),
    ).toEqual({
      pricingConfig: {
        source_kind: 'custom',
        pricing_mode: 'flat',
        rates: {
          input_per_token: 0.000005,
          output_per_token: 0.00003,
        },
      },
      costPerInputToken: '5',
      costPerOutputToken: '30',
    })
  })
})
