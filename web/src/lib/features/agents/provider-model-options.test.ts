import { describe, expect, it } from 'vitest'

import type { AgentProviderModelCatalogEntry } from '$lib/api/contracts'
import {
  providerReasoningCapabilitySummary,
  providerModelReasoningCapability,
  recommendedProviderModelId,
  splitProviderModelSelection,
} from './provider-model-options'

const providerModelCatalogFixture: AgentProviderModelCatalogEntry[] = [
  {
    adapter_type: 'codex-app-server',
    options: [
      {
        id: 'gpt-5.4',
        label: 'gpt-5.4',
        description: 'Latest frontier agentic coding model.',
        recommended: true,
        preview: false,
        pricing_config: null,
        reasoning: {
          state: 'available',
          reason: null,
          supported_efforts: ['low', 'medium', 'high', 'xhigh'],
          default_effort: 'medium',
          supports_provider_preset: true,
          supports_model_override: false,
        },
      },
      {
        id: 'gpt-5.4-mini',
        label: 'gpt-5.4-mini',
        description: 'Smaller frontier agentic coding model.',
        recommended: false,
        preview: false,
        pricing_config: null,
        reasoning: {
          state: 'available',
          reason: null,
          supported_efforts: ['low', 'medium', 'high', 'xhigh'],
          default_effort: 'medium',
          supports_provider_preset: true,
          supports_model_override: false,
        },
      },
    ],
  },
  {
    adapter_type: 'claude-code-cli',
    options: [
      {
        id: 'claude-opus-4-6',
        label: 'Default',
        description: 'Opus 4.6 with 1M context.',
        recommended: true,
        preview: false,
        pricing_config: null,
        reasoning: {
          state: 'available',
          reason: null,
          supported_efforts: ['low', 'medium', 'high', 'max'],
          default_effort: null,
          supports_provider_preset: true,
          supports_model_override: false,
        },
      },
      {
        id: 'claude-sonnet-4-6',
        label: 'Sonnet',
        description: 'Sonnet 4.6.',
        recommended: false,
        preview: false,
        pricing_config: null,
        reasoning: {
          state: 'available',
          reason: null,
          supported_efforts: ['low', 'medium', 'high'],
          default_effort: null,
          supports_provider_preset: true,
          supports_model_override: false,
        },
      },
      {
        id: 'claude-haiku-4-5',
        label: 'Haiku',
        description: 'Haiku 4.5.',
        recommended: false,
        preview: false,
        pricing_config: null,
        reasoning: {
          state: 'unsupported',
          reason: 'reasoning_unsupported',
          supported_efforts: [],
          default_effort: null,
          supports_provider_preset: false,
          supports_model_override: false,
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
        pricing_config: null,
        reasoning: {
          state: 'unsupported',
          reason: 'reasoning_unsupported',
          supported_efforts: [],
          default_effort: null,
          supports_provider_preset: false,
          supports_model_override: false,
        },
      },
      {
        id: 'gemini-3-flash-preview',
        label: 'gemini-3-flash-preview',
        description: 'Preview Gemini 3 Flash model.',
        recommended: false,
        preview: true,
        pricing_config: null,
        reasoning: {
          state: 'unsupported',
          reason: 'reasoning_unsupported',
          supported_efforts: [],
          default_effort: null,
          supports_provider_preset: false,
          supports_model_override: false,
        },
      },
    ],
  },
]

describe('provider model options', () => {
  it('returns the recommended model id for an adapter', () => {
    expect(recommendedProviderModelId(providerModelCatalogFixture, 'codex-app-server')).toBe(
      'gpt-5.4',
    )
  })

  it('keeps a known model as the base selection', () => {
    expect(
      splitProviderModelSelection(
        providerModelCatalogFixture,
        'gemini-cli',
        'gemini-3-flash-preview',
        true,
      ),
    ).toEqual({
      baseModelId: 'gemini-3-flash-preview',
      customModelId: '',
    })
  })

  it('preserves an unknown model as a custom override for the same adapter', () => {
    expect(
      splitProviderModelSelection(
        providerModelCatalogFixture,
        'gemini-cli',
        'gemini-2.5-pro-experimental',
        true,
      ),
    ).toEqual({
      baseModelId: 'auto-gemini-2.5',
      customModelId: 'gemini-2.5-pro-experimental',
    })
  })

  it('falls back to the recommended model when the adapter changes', () => {
    expect(
      splitProviderModelSelection(providerModelCatalogFixture, 'gemini-cli', 'gpt-5.4', false),
    ).toEqual({
      baseModelId: 'auto-gemini-2.5',
      customModelId: '',
    })
  })

  it('derives reasoning capability for known and unknown models', () => {
    expect(
      providerModelReasoningCapability(providerModelCatalogFixture, 'codex-app-server', 'gpt-5.4'),
    ).toEqual(
      expect.objectContaining({
        state: 'available',
        defaultEffort: 'medium',
        supportedEfforts: ['low', 'medium', 'high', 'xhigh'],
      }),
    )

    expect(
      providerModelReasoningCapability(
        providerModelCatalogFixture,
        'codex-app-server',
        'custom-model',
      ),
    ).toEqual(
      expect.objectContaining({
        state: 'unsupported',
        reason: 'unknown_model',
      }),
    )
  })

  it('keeps Claude plan-dependent defaults unset and models unsupported effort explicitly', () => {
    expect(
      providerModelReasoningCapability(
        providerModelCatalogFixture,
        'claude-code-cli',
        'claude-sonnet-4-6',
      ),
    ).toEqual(
      expect.objectContaining({
        state: 'available',
        defaultEffort: null,
        supportedEfforts: ['low', 'medium', 'high'],
      }),
    )

    const unsupported = providerModelReasoningCapability(
      providerModelCatalogFixture,
      'claude-code-cli',
      'claude-haiku-4-5',
    )
    expect(unsupported).toEqual(
      expect.objectContaining({
        state: 'unsupported',
        reason: 'reasoning_unsupported',
      }),
    )
    expect(providerReasoningCapabilitySummary(unsupported)).toBe(
      'Reasoning presets are unavailable for this model or adapter.',
    )
  })
})
