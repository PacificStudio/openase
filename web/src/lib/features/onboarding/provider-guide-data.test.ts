import { describe, expect, it } from 'vitest'

import { guideText } from './provider-guide-data'

describe('provider guide data', () => {
  it('surfaces gpt-5.5 as the Codex recommended model', () => {
    expect(guideText('codex', 'recommendedModel')).toBe('gpt-5.5')
  })
})
