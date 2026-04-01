import type { SkillFile } from '$lib/api/contracts'
import type { EphemeralChatTranscriptEntry } from '$lib/features/chat'
import { applyStructuredDiffToText } from '$lib/features/chat'

export { buildDiffPreview, fingerprintSuggestion, type DiffPreview } from '$lib/features/chat'

export type SkillSuggestedFile = {
  path: string
  content: string
}

export type SkillSuggestion = {
  files: SkillSuggestedFile[]
  summary: string
}

const fencedBlockPattern = /```(?:[a-z0-9_+-]+)?\s*([\s\S]*?)```/gi

export function findLatestSkillSuggestion(
  entries: EphemeralChatTranscriptEntry[],
  input: {
    selectedFilePath: string
    files: SkillFile[]
  },
): SkillSuggestion | null {
  const normalizedSelectedPath = normalizePath(input.selectedFilePath)
  if (!normalizedSelectedPath) {
    return null
  }

  const filesByPath = new Map(
    input.files.map((file) => [normalizePath(file.path), file] satisfies [string, SkillFile]),
  )

  for (let index = entries.length - 1; index >= 0; index -= 1) {
    const entry = entries[index]
    if (entry.role !== 'assistant') {
      continue
    }

    if (entry.kind === 'bundle_diff') {
      const suggestedFiles = entry.bundleDiff.files
        .map((item) => buildSuggestedFile(filesByPath, normalizePath(item.file), item.hunks))
        .filter((item): item is SkillSuggestedFile => item !== null)
      if (suggestedFiles.length === 0) {
        continue
      }
      return {
        files: sortSuggestedFiles(suggestedFiles, normalizedSelectedPath),
        summary: `Suggested multi-file update for ${suggestedFiles.length} files.`,
      }
    }

    if (entry.kind === 'diff') {
      const suggestedFile = buildSuggestedFile(
        filesByPath,
        normalizePath(entry.diff.file),
        entry.diff.hunks,
      )
      if (!suggestedFile) {
        continue
      }
      return {
        files: [suggestedFile],
        summary: `Suggested update for ${suggestedFile.path}.`,
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
      files: [{ path: normalizedSelectedPath, content: suggestionContent }],
      summary: normalizeSuggestionSummary(stripCodeBlocks(entry.content)),
    }
  }

  return null
}

function buildSuggestedFile(
  filesByPath: Map<string, SkillFile>,
  normalizedPath: string,
  hunks: Array<{
    oldStart: number
    oldLines: number
    newStart: number
    newLines: number
    lines: Array<{ op: 'context' | 'add' | 'remove'; text: string }>
  }>,
) {
  if (!normalizedPath) {
    return null
  }

  const existing = filesByPath.get(normalizedPath)
  if (existing && existing.encoding !== 'utf8') {
    return null
  }

  const content = applyStructuredDiffToText(existing?.content ?? '', {
    type: 'diff',
    file: normalizedPath,
    hunks,
  })
  if (content == null) {
    return null
  }

  return {
    path: normalizedPath,
    content,
  } satisfies SkillSuggestedFile
}

function sortSuggestedFiles(files: SkillSuggestedFile[], selectedFilePath: string) {
  return [...files].sort((left, right) => {
    if (left.path === selectedFilePath) return -1
    if (right.path === selectedFilePath) return 1
    return left.path.localeCompare(right.path)
  })
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
