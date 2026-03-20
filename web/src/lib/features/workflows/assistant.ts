import type { ChatMessagePayload } from '$lib/api/chat'

export type AssistantRole = 'user' | 'assistant' | 'system'

export type AssistantTranscriptEntry = {
  id: string
  role: AssistantRole
  content: string
}

export type HarnessSuggestion = {
  content: string
  summary: string
}

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

const fencedBlockPattern = /```(?:markdown|md|yaml|yml|text)?\s*([\s\S]*?)```/gi

export function findLatestHarnessSuggestion(
  entries: AssistantTranscriptEntry[],
): HarnessSuggestion | null {
  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index]
    if (entry.role !== 'assistant') {
      continue
    }

    const suggestion = extractHarnessSuggestion(entry.content)
    if (suggestion) {
      return suggestion
    }
  }

  return null
}

export function extractHarnessSuggestion(text: string): HarnessSuggestion | null {
  const blocks = [...text.matchAll(fencedBlockPattern)]
  for (let index = blocks.length - 1; index >= 0; index -= 1) {
    const content = parseHarnessDocumentCandidate(blocks[index]?.[1] ?? '')
    if (!content) {
      continue
    }
    return {
      content,
      summary: normalizeSuggestionSummary(stripCodeBlocks(text)),
    }
  }

  const content = parseHarnessDocumentCandidate(text)
  if (!content) {
    return null
  }

  return {
    content,
    summary: normalizeSuggestionSummary(text),
  }
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

export function mapChatPayloadToTranscriptEntry(
  payload: ChatMessagePayload,
): Omit<AssistantTranscriptEntry, 'id'> {
  if (isTextPayload(payload)) {
    return {
      role: 'assistant',
      content: payload.content,
    }
  }

  if (isActionProposalPayload(payload)) {
    return {
      role: 'system',
      content: `Action proposal: ${payload.summary ?? 'Awaiting confirmation.'}`,
    }
  }

  return {
    role: 'system',
    content: describeSystemMessage(payload.type),
  }
}

export function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function parseHarnessDocumentCandidate(raw: string): string | null {
  const normalized = normalizeDocument(raw)
  if (!normalized.startsWith('---\n')) {
    return null
  }

  const closingIndex = normalized.indexOf('\n---\n', 4)
  if (closingIndex === -1) {
    return null
  }

  return normalized
}

function normalizeDocument(text: string) {
  return text.replaceAll('\r\n', '\n').trim()
}

function stripCodeBlocks(text: string) {
  return text.replaceAll(fencedBlockPattern, '').trim()
}

function normalizeSuggestionSummary(summary: string) {
  const trimmed = summary.trim()
  return trimmed || 'Suggested harness update.'
}

function describeSystemMessage(type: string) {
  switch (type) {
    case 'task_started':
      return 'Assistant started a background task.'
    case 'task_progress':
      return 'Assistant reported task progress.'
    case 'task_notification':
      return 'Assistant emitted a task notification.'
    default:
      return `Assistant emitted ${type}.`
  }
}

function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

function isActionProposalPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'action_proposal' }> {
  return payload.type === 'action_proposal'
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
