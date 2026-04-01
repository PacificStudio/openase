import type { EphemeralChatTranscriptEntry } from '$lib/features/chat'
import { applyStructuredDiffToText } from '$lib/features/chat/structured-diff'

export {
  buildDiffPreview,
  fingerprintSuggestion,
  type DiffPreview,
} from '$lib/features/chat/structured-diff'

export type SkillSuggestion = {
  path: string
  content: string
  summary: string
}

const fencedBlockPattern = /```(?:[a-z0-9_+-]+)?\s*([\s\S]*?)```/gi

export function findLatestSkillSuggestion(
  entries: EphemeralChatTranscriptEntry[],
  currentFilePath: string,
  currentContent = '',
): SkillSuggestion | null {
  const normalizedCurrentPath = normalizePath(currentFilePath)
  if (!normalizedCurrentPath) {
    return null
  }

  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index]
    if (entry.role !== 'assistant') {
      continue
    }

    if (entry.kind === 'diff') {
      if (normalizePath(entry.diff.file) !== normalizedCurrentPath) {
        continue
      }
      const content = applyStructuredDiffToText(currentContent, entry.diff)
      if (content == null) {
        continue
      }
      return {
        path: normalizedCurrentPath,
        content,
        summary: `Suggested update for ${normalizedCurrentPath}.`,
      }
    }

    if (entry.kind !== 'text') {
      continue
    }

    const suggestionContent = extractTextSuggestion(entry.content)
    if (!suggestionContent) {
      continue
    }
    return {
      path: normalizedCurrentPath,
      content: suggestionContent,
      summary: normalizeSuggestionSummary(stripCodeBlocks(entry.content)),
    }
  }

  return null
}

function extractTextSuggestion(text: string) {
  const blocks = [...text.matchAll(fencedBlockPattern)]
  for (let index = blocks.length - 1; index >= 0; index -= 1) {
    const content = normalizeDocument(blocks[index]?.[1] ?? '')
    if (content) {
      return content
    }
  }

  return null
}

function normalizeDocument(text: string) {
  return text.replaceAll('\r\n', '\n').trim()
}

function normalizePath(value: string) {
  return value.trim().replaceAll('\\', '/')
}

function stripCodeBlocks(text: string) {
  return text.replaceAll(fencedBlockPattern, '').trim()
}

function normalizeSuggestionSummary(summary: string) {
  const trimmed = summary.trim()
  return trimmed || 'Suggested file update.'
}
