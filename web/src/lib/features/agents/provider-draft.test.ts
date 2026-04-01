import { describe, expect, it } from 'vitest'

import { createEmptyProviderDraft, parseProviderDraft } from './provider-draft'

describe('provider draft', () => {
  it('defaults max parallel runs to blank and treats blank as unlimited', () => {
    expect(createEmptyProviderDraft().maxParallelRuns).toBe('')

    const parsed = parseProviderDraft({
      ...createEmptyProviderDraft(),
      machineId: 'machine-1',
      name: 'Codex Local',
      adapterType: 'codex-app-server',
      modelName: 'gpt-5.4',
      modelTemperature: '0',
      modelMaxTokens: '16384',
      maxParallelRuns: '',
      costPerInputToken: '0',
      costPerOutputToken: '0',
    })

    expect(parsed).toEqual({
      ok: true,
      value: expect.objectContaining({
        max_parallel_runs: 0,
      }),
    })
  })

  it('rejects zero and negative max parallel runs', () => {
    for (const maxParallelRuns of ['0', '-1']) {
      const parsed = parseProviderDraft({
        ...createEmptyProviderDraft(),
        machineId: 'machine-1',
        name: 'Codex Local',
        adapterType: 'codex-app-server',
        modelName: 'gpt-5.4',
        modelTemperature: '0',
        modelMaxTokens: '16384',
        maxParallelRuns,
        costPerInputToken: '0',
        costPerOutputToken: '0',
      })

      expect(parsed).toEqual({
        ok: false,
        error: 'Max parallel runs must be a positive integer.',
      })
    }
  })
})
