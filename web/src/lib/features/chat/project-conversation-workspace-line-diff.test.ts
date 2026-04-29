import { describe, expect, it } from 'vitest'

import {
  computeDraftLineDiff,
  computePatchLineDiff,
  isWorkspaceFileLineDiffEmpty,
} from './project-conversation-workspace-line-diff'

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

  it('marks modified and deleted lines from a saved workspace patch', () => {
    expect(
      computePatchLineDiff({
        status: 'modified',
        diffKind: 'text',
        diff: '@@ -1,3 +1,3 @@\n line one\n-line two\n+line two changed\n line three\n',
        content: 'line one\nline two changed\nline three\n',
      }),
    ).toEqual({
      added: [],
      modified: [2],
      deletionAbove: [],
      deletionAtEnd: false,
    })
  })

  it('marks all lines as added for newly created files', () => {
    expect(
      computePatchLineDiff({
        status: 'added',
        diffKind: 'text',
        diff: '@@ -0,0 +1,2 @@\n+alpha\n+beta\n',
        content: 'alpha\nbeta\n',
      }),
    ).toEqual({
      added: [1, 2, 3],
      modified: [],
      deletionAbove: [],
      deletionAtEnd: false,
    })
  })

  it('detects when line diff markers are empty', () => {
    expect(
      isWorkspaceFileLineDiffEmpty({
        added: [],
        modified: [],
        deletionAbove: [],
        deletionAtEnd: false,
      }),
    ).toBe(true)
    expect(
      isWorkspaceFileLineDiffEmpty({
        added: [],
        modified: [2],
        deletionAbove: [],
        deletionAtEnd: false,
      }),
    ).toBe(false)
  })
})
