import type { ChatDiffPayload } from '$lib/api/chat'
import { applyStructuredDiffToText, type EphemeralChatTranscriptEntry } from '$lib/features/chat'
export { buildDiffPreview, fingerprintSuggestion, type DiffPreview } from '$lib/features/chat'

export type HarnessSuggestion = {
  content: string
  summary: string
}

const fencedBlockPattern = /```(?:markdown|md|yaml|yml|text)?\s*([\s\S]*?)```/gi

export function findLatestHarnessSuggestion(
  entries: EphemeralChatTranscriptEntry[],
  currentContent = '',
): HarnessSuggestion | null {
  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index]
    if (entry.role !== 'assistant') {
      continue
    }

    if (entry.kind === 'diff') {
      const suggestion = buildHarnessSuggestionFromDiff(currentContent, entry.diff)
      if (suggestion) {
        return suggestion
      }
      continue
    }

    if (entry.kind !== 'text') {
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
    summary: 'Suggested harness update.',
  }
}

export function applyStructuredDiff(before: string, diff: ChatDiffPayload): string | null {
  if (!isHarnessContentTarget(diff.file)) {
    return null
  }
  return applyStructuredDiffToText(before, diff)
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

function buildHarnessSuggestionFromDiff(
  currentContent: string,
  diff: ChatDiffPayload,
): HarnessSuggestion | null {
  const content = applyStructuredDiff(currentContent, diff)
  if (content == null) {
    return null
  }

  return {
    content,
    summary: `Suggested harness update for ${diff.file}.`,
  }
}

function stripCodeBlocks(text: string) {
  return text.replaceAll(fencedBlockPattern, '').trim()
}

function normalizeSuggestionSummary(summary: string) {
  const trimmed = summary.trim()
  return trimmed || 'Suggested harness update.'
}

function isHarnessContentTarget(file: string) {
  return file.trim().toLowerCase() === 'harness content'
}
