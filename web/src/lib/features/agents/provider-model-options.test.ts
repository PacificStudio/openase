import { describe, expect, it } from 'vitest'

import type { AgentProviderModelCatalogEntry } from '$lib/api/contracts'
import { recommendedProviderModelId, splitProviderModelSelection } from './provider-model-options'

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
      },
      {
        id: 'gpt-5.4-mini',
        label: 'gpt-5.4-mini',
        description: 'Smaller frontier agentic coding model.',
        recommended: false,
        preview: false,
        pricing_config: null,
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
      },
      {
        id: 'gemini-3-flash-preview',
        label: 'gemini-3-flash-preview',
        description: 'Preview Gemini 3 Flash model.',
        recommended: false,
        preview: true,
        pricing_config: null,
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
})
