import { describe, expect, it } from 'vitest'

import { buildRunSummaryPrompt, defaultRunSummarySectionKeys } from './run-summary-prompt-template'

describe('run summary prompt template', () => {
  it('builds the default selected sections in order', () => {
    const prompt = buildRunSummaryPrompt(defaultRunSummarySectionKeys, '')

    expect(prompt).toContain('## Major Steps')
    expect(prompt).toContain('## Long-Running Operations')
    expect(prompt).toContain('## Repeated Trial-and-Error')
    expect(prompt).toContain('## Security / Safety Risks')
    expect(prompt).toContain('## Outcome')
    expect(prompt).not.toContain('## Overview')
  })

  it('appends additional instructions when provided', () => {
    const prompt = buildRunSummaryPrompt(['outcome'], 'Keep it terse.')

    expect(prompt).toContain('## Outcome')
    expect(prompt).toContain('Additional instructions:\nKeep it terse.')
  })
})
