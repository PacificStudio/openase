import { describe, expect, it } from 'vitest'

import { applyStructuredDiff, findLatestHarnessSuggestion } from './assistant'

describe('workflow assistant helpers', () => {
  it('applies structured harness diffs into a full suggestion', () => {
    const before = ['---', 'type: coding', '---', '', 'Write code.'].join('\n')
    const suggestion = findLatestHarnessSuggestion(
      [
        {
          id: 'entry-1',
          role: 'assistant',
          kind: 'diff',
          diff: {
            type: 'diff',
            file: 'harness content',
            hunks: [
              {
                oldStart: 5,
                oldLines: 1,
                newStart: 5,
                newLines: 2,
                lines: [
                  { op: 'remove', text: 'Write code.' },
                  { op: 'add', text: 'Write code.' },
                  { op: 'add', text: 'Add tests.' },
                ],
              },
            ],
          },
        },
      ],
      before,
    )

    expect(suggestion).toEqual({
      content: ['---', 'type: coding', '---', '', 'Write code.', 'Add tests.'].join('\n'),
      summary: 'Suggested harness update for harness content.',
    })
  })

  it('rejects structured diffs for non-harness targets', () => {
    const result = applyStructuredDiff('line one', {
      type: 'diff',
      file: 'README.md',
      hunks: [
        {
          oldStart: 1,
          oldLines: 1,
          newStart: 1,
          newLines: 1,
          lines: [{ op: 'context', text: 'line one' }],
        },
      ],
    })

    expect(result).toBeNull()
  })
})
