import type { ChatDiffPayload } from '$lib/api/chat'

export type DiffLine = {
  kind: 'context' | 'add' | 'remove'
  content: string
  beforeLineNumber?: number
  afterLineNumber?: number
}

export type DiffPreview = {
  addedCount: number
  removedCount: number
  hasChanges: boolean
  lines: DiffLine[]
}

export function buildDiffPreview(before: string, after: string): DiffPreview {
  const beforeLines = normalizeDocument(before).split('\n')
  const afterLines = normalizeDocument(after).split('\n')
  const operations = buildDiffOperations(beforeLines, afterLines)

  const lines: DiffLine[] = []
  let addedCount = 0
  let removedCount = 0
  let beforeLineNumber = 1
  let afterLineNumber = 1

  for (const operation of operations) {
    if (operation.kind === 'context') {
      lines.push({
        kind: 'context',
        content: operation.content,
        beforeLineNumber,
        afterLineNumber,
      })
      beforeLineNumber += 1
      afterLineNumber += 1
      continue
    }

    if (operation.kind === 'remove') {
      lines.push({
        kind: 'remove',
        content: operation.content,
        beforeLineNumber,
      })
      removedCount += 1
      beforeLineNumber += 1
      continue
    }

    lines.push({
      kind: 'add',
      content: operation.content,
      afterLineNumber,
    })
    addedCount += 1
    afterLineNumber += 1
  }

  return {
    addedCount,
    removedCount,
    hasChanges: addedCount > 0 || removedCount > 0,
    lines,
  }
}

export function fingerprintSuggestion(content: string) {
  return `${content.length}:${content.slice(0, 120)}`
}

export function applyStructuredDiffToText(before: string, diff: ChatDiffPayload): string | null {
  const sourceLines = splitNormalizedLines(before)
  const outputLines: string[] = []
  let cursor = 1

  for (const hunk of diff.hunks) {
    if (hunk.oldStart < cursor) {
      return null
    }

    outputLines.push(...sourceLines.slice(cursor - 1, hunk.oldStart - 1))

    let oldConsumed = 0
    let newProduced = 0
    let sourceIndex = hunk.oldStart - 1

    for (const line of hunk.lines) {
      if (line.op === 'add') {
        outputLines.push(line.text)
        newProduced += 1
        continue
      }

      if (sourceLines[sourceIndex] !== line.text) {
        return null
      }

      if (line.op === 'context') {
        outputLines.push(line.text)
        newProduced += 1
      }

      sourceIndex += 1
      oldConsumed += 1
    }

    if (oldConsumed !== hunk.oldLines || newProduced !== hunk.newLines) {
      return null
    }

    cursor = hunk.oldStart + hunk.oldLines
  }

  outputLines.push(...sourceLines.slice(cursor - 1))
  return outputLines.join('\n')
}

function normalizeDocument(text: string) {
  return text.replaceAll('\r\n', '\n').trim()
}

function splitNormalizedLines(text: string) {
  const normalized = normalizeDocument(text)
  if (!normalized) {
    return [] as string[]
  }

  return normalized.split('\n')
}

function buildDiffOperations(before: string[], after: string[]) {
  const rows = before.length
  const columns = after.length
  const matrix = Array.from({ length: rows + 1 }, () => Array<number>(columns + 1).fill(0))

  for (let row = rows - 1; row >= 0; row -= 1) {
    for (let column = columns - 1; column >= 0; column -= 1) {
      if (before[row] === after[column]) {
        matrix[row][column] = matrix[row + 1][column + 1] + 1
        continue
      }

      matrix[row][column] = Math.max(matrix[row + 1][column], matrix[row][column + 1])
    }
  }

  const operations: Array<{ kind: DiffLine['kind']; content: string }> = []
  let beforeIndex = 0
  let afterIndex = 0

  while (beforeIndex < rows && afterIndex < columns) {
    if (before[beforeIndex] === after[afterIndex]) {
      operations.push({ kind: 'context', content: before[beforeIndex] })
      beforeIndex += 1
      afterIndex += 1
      continue
    }

    if (matrix[beforeIndex + 1][afterIndex] >= matrix[beforeIndex][afterIndex + 1]) {
      operations.push({ kind: 'remove', content: before[beforeIndex] })
      beforeIndex += 1
      continue
    }

    operations.push({ kind: 'add', content: after[afterIndex] })
    afterIndex += 1
  }

  while (beforeIndex < rows) {
    operations.push({ kind: 'remove', content: before[beforeIndex] })
    beforeIndex += 1
  }

  while (afterIndex < columns) {
    operations.push({ kind: 'add', content: after[afterIndex] })
    afterIndex += 1
  }

  return operations
}
