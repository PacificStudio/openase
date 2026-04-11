/**
 * Per-line diff markers for the in-editor gutter, computed against the latest
 * saved content. All line numbers are 1-based, referencing the draft document.
 */
export type WorkspaceFileLineDiffMarkers = {
  /** Lines that exist in the draft but not in the saved version. */
  added: number[]
  /** Lines that replace a saved line at (roughly) the same position. */
  modified: number[]
  /** Lines where deletion happened *immediately above* the marked line. */
  deletionAbove: number[]
  /** True when content was deleted past the end of the draft document. */
  deletionAtEnd: boolean
}

const EMPTY_LINE_DIFF: WorkspaceFileLineDiffMarkers = Object.freeze({
  added: [],
  modified: [],
  deletionAbove: [],
  deletionAtEnd: false,
}) as WorkspaceFileLineDiffMarkers

type DiffOp =
  | { kind: 'eq'; oldIdx: number; newIdx: number }
  | { kind: 'del'; oldIdx: number }
  | { kind: 'add'; newIdx: number }

function lcsDiff(oldLines: string[], newLines: string[]): DiffOp[] {
  const m = oldLines.length
  const n = newLines.length
  const dp: number[][] = []
  for (let i = 0; i <= m; i++) {
    dp.push(new Array<number>(n + 1).fill(0))
  }
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (oldLines[i - 1] === newLines[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1])
      }
    }
  }
  const ops: DiffOp[] = []
  let i = m
  let j = n
  while (i > 0 && j > 0) {
    if (oldLines[i - 1] === newLines[j - 1]) {
      ops.push({ kind: 'eq', oldIdx: i - 1, newIdx: j - 1 })
      i--
      j--
    } else if (dp[i - 1][j] >= dp[i][j - 1]) {
      ops.push({ kind: 'del', oldIdx: i - 1 })
      i--
    } else {
      ops.push({ kind: 'add', newIdx: j - 1 })
      j--
    }
  }
  while (i > 0) {
    ops.push({ kind: 'del', oldIdx: i - 1 })
    i--
  }
  while (j > 0) {
    ops.push({ kind: 'add', newIdx: j - 1 })
    j--
  }
  ops.reverse()
  return ops
}

type Hunk = {
  dels: Array<Extract<DiffOp, { kind: 'del' }>>
  adds: Array<Extract<DiffOp, { kind: 'add' }>>
  end: number
}

function gatherHunk(ops: DiffOp[], start: number): Hunk {
  let end = start
  while (end < ops.length && ops[end].kind !== 'eq') {
    end++
  }
  const dels: Hunk['dels'] = []
  const adds: Hunk['adds'] = []
  for (let i = start; i < end; i++) {
    const op = ops[i]
    if (op.kind === 'del') dels.push(op)
    else if (op.kind === 'add') adds.push(op)
  }
  return { dels, adds, end }
}

function leftoverDeletionAnchor(
  ops: DiffOp[],
  hunk: Hunk,
  newLineCount: number,
): { aboveLine: number | null; atEnd: boolean } {
  if (hunk.adds.length > 0) {
    const lastAddNewIdx = hunk.adds[hunk.adds.length - 1].newIdx
    const candidate = lastAddNewIdx + 2
    if (candidate <= newLineCount) return { aboveLine: candidate, atEnd: false }
    return { aboveLine: null, atEnd: true }
  }
  if (hunk.end < ops.length) {
    const nextEq = ops[hunk.end] as Extract<DiffOp, { kind: 'eq' }>
    return { aboveLine: nextEq.newIdx + 1, atEnd: false }
  }
  return { aboveLine: null, atEnd: true }
}

export function computeDraftLineDiff(
  savedContent: string,
  draftContent: string,
): WorkspaceFileLineDiffMarkers {
  if (savedContent === draftContent) {
    return EMPTY_LINE_DIFF
  }
  const oldLines = savedContent === '' ? [] : savedContent.split('\n')
  const newLines = draftContent === '' ? [] : draftContent.split('\n')
  const ops = lcsDiff(oldLines, newLines)

  const added = new Set<number>()
  const modified = new Set<number>()
  const deletionAbove = new Set<number>()
  let deletionAtEnd = false

  let k = 0
  while (k < ops.length) {
    if (ops[k].kind === 'eq') {
      k++
      continue
    }
    const hunk = gatherHunk(ops, k)
    const pairCount = Math.min(hunk.dels.length, hunk.adds.length)
    for (let p = 0; p < pairCount; p++) {
      modified.add(hunk.adds[p].newIdx + 1)
    }
    for (let p = pairCount; p < hunk.adds.length; p++) {
      added.add(hunk.adds[p].newIdx + 1)
    }
    if (hunk.dels.length > hunk.adds.length) {
      const anchor = leftoverDeletionAnchor(ops, hunk, newLines.length)
      if (anchor.aboveLine !== null) deletionAbove.add(anchor.aboveLine)
      else if (anchor.atEnd) deletionAtEnd = true
    }
    k = hunk.end
  }

  return {
    added: [...added].sort((a, b) => a - b),
    modified: [...modified].sort((a, b) => a - b),
    deletionAbove: [...deletionAbove].sort((a, b) => a - b),
    deletionAtEnd,
  }
}
