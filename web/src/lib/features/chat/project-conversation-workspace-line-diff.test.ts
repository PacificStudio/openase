import { describe, expect, it } from 'vitest'

import { computeDraftLineDiff } from './project-conversation-workspace-line-diff'

describe('computeDraftLineDiff', () => {
  it('returns empty markers when the draft matches the saved file', () => {
    expect(computeDraftLineDiff('line one\nline two\n', 'line one\nline two\n')).toEqual({
      added: [],
      modified: [],
      deletionAbove: [],
      deletionAtEnd: false,
    })
  })

  it('marks modified and added draft lines separately', () => {
    expect(
      computeDraftLineDiff('alpha\nbeta\ngamma\n', 'alpha\nbeta changed\ngamma\nextra\n'),
    ).toEqual({
      added: [4],
      modified: [2],
      deletionAbove: [],
      deletionAtEnd: false,
    })
  })

  it('anchors deletions above the next surviving draft line', () => {
    expect(computeDraftLineDiff('alpha\nbeta\ngamma\ndelta\n', 'alpha\ndelta\n')).toEqual({
      added: [],
      modified: [],
      deletionAbove: [2],
      deletionAtEnd: false,
    })
  })

  it('marks deletions at end-of-file when removed content has no following line', () => {
    expect(computeDraftLineDiff('alpha\n', '')).toEqual({
      added: [],
      modified: [],
      deletionAbove: [],
      deletionAtEnd: true,
    })
  })
})
