import type { ChatDiffPayload } from '$lib/api/chat'
import {
  applyStructuredDiffToText,
  buildDiffPreview,
  fingerprintSuggestion,
} from './structured-diff'

export const WORKSPACE_SELECTION_TEXT_LIMIT = 4000
export const WORKSPACE_SELECTION_CONTEXT_LIMIT = 1200
export const WORKSPACE_WORKING_SET_FILE_LIMIT = 4
export const WORKSPACE_WORKING_SET_TOTAL_CHAR_LIMIT = 6000

export type WorkspaceEditorSelection = {
  from: number
  to: number
  startLine: number
  startColumn: number
  endLine: number
  endColumn: number
  text: string
  contextBefore: string
  contextAfter: string
  truncated: boolean
}

export type WorkspaceWorkingSetEntry = {
  filePath: string
  contentExcerpt: string
  dirty: boolean
  truncated: boolean
}

export type WorkspacePatchProposal = {
  diff: ChatDiffPayload
  proposedContent: string
  preview: ReturnType<typeof buildDiffPreview>
  fingerprint: string
}

export type WorkspaceSelectionInput = {
  from: number
  to: number
}

export function clampSelection(
  content: string,
  selection: WorkspaceSelectionInput | null | undefined,
): WorkspaceSelectionInput | null {
  if (!selection) {
    return null
  }
  const length = content.length
  const from = Math.max(0, Math.min(selection.from, length))
  const to = Math.max(from, Math.min(selection.to, length))
  return { from, to }
}

export function buildWorkspaceSelection(
  content: string,
  selection: WorkspaceSelectionInput | null | undefined,
): WorkspaceEditorSelection | null {
  const clamped = clampSelection(content, selection)
  if (!clamped || clamped.from === clamped.to) {
    return null
  }

  const fullText = content.slice(clamped.from, clamped.to)
  const text = fullText.slice(0, WORKSPACE_SELECTION_TEXT_LIMIT)
  const contextBeforeFull = content.slice(
    Math.max(0, clamped.from - WORKSPACE_SELECTION_CONTEXT_LIMIT),
    clamped.from,
  )
  const contextAfterFull = content.slice(
    clamped.to,
    Math.min(content.length, clamped.to + WORKSPACE_SELECTION_CONTEXT_LIMIT),
  )

  return {
    from: clamped.from,
    to: clamped.to,
    startLine: lineNumberAtOffset(content, clamped.from),
    startColumn: columnNumberAtOffset(content, clamped.from),
    endLine: lineNumberAtOffset(content, clamped.to),
    endColumn: columnNumberAtOffset(content, clamped.to),
    text,
    contextBefore: contextBeforeFull,
    contextAfter: contextAfterFull,
    truncated:
      text.length !== fullText.length ||
      contextBeforeFull.length === WORKSPACE_SELECTION_CONTEXT_LIMIT ||
      contextAfterFull.length === WORKSPACE_SELECTION_CONTEXT_LIMIT,
  }
}

export function createWorkspacePatchProposal(
  before: string,
  diff: ChatDiffPayload,
): WorkspacePatchProposal | null {
  const proposedContent = applyStructuredDiffToText(before, diff)
  if (proposedContent == null) {
    return null
  }
  return {
    diff,
    proposedContent,
    preview: buildDiffPreview(before, proposedContent),
    fingerprint: fingerprintSuggestion(proposedContent),
  }
}

export function formatWorkspaceDocument(filePath: string, content: string): string | null {
  if (!looksLikeJSONFile(filePath)) {
    return null
  }
  const normalized = normalizeLineEndings(content)
  const trailingNewline = normalized.endsWith('\n')
  const parsed = JSON.parse(normalized)
  const formatted = `${JSON.stringify(parsed, null, 2)}${trailingNewline ? '\n' : ''}`
  return restoreLineEndings(content, formatted)
}

export function formatWorkspaceSelection(
  filePath: string,
  content: string,
  selection: WorkspaceSelectionInput | null | undefined,
): { content: string; selection: WorkspaceSelectionInput } | null {
  if (!looksLikeJSONFile(filePath)) {
    return null
  }
  const clamped = clampSelection(content, selection)
  if (!clamped || clamped.from === clamped.to) {
    return null
  }
  const selected = content.slice(clamped.from, clamped.to)
  const normalized = normalizeLineEndings(selected)
  const trailingNewline = normalized.endsWith('\n')
  const parsed = JSON.parse(normalized)
  const formatted = `${JSON.stringify(parsed, null, 2)}${trailingNewline ? '\n' : ''}`
  const restored = restoreLineEndings(selected, formatted)
  return {
    content: `${content.slice(0, clamped.from)}${restored}${content.slice(clamped.to)}`,
    selection: {
      from: clamped.from,
      to: clamped.from + restored.length,
    },
  }
}

export function buildWorkspaceWorkingSet(
  items: Array<{ filePath: string; content: string; dirty: boolean }>,
): WorkspaceWorkingSetEntry[] {
  const result: WorkspaceWorkingSetEntry[] = []
  let remaining = WORKSPACE_WORKING_SET_TOTAL_CHAR_LIMIT

  for (const item of items.slice(0, WORKSPACE_WORKING_SET_FILE_LIMIT)) {
    if (remaining <= 0) {
      break
    }
    const excerpt = item.content.slice(0, remaining)
    result.push({
      filePath: item.filePath,
      contentExcerpt: excerpt,
      dirty: item.dirty,
      truncated: excerpt.length !== item.content.length,
    })
    remaining -= excerpt.length
  }

  return result
}

function looksLikeJSONFile(filePath: string) {
  return /(^|\/)package\.json$/i.test(filePath) || /\.json$/i.test(filePath)
}

function normalizeLineEndings(value: string) {
  return value.replaceAll('\r\n', '\n')
}

function restoreLineEndings(source: string, value: string) {
  return source.includes('\r\n') ? value.replaceAll('\n', '\r\n') : value
}

function lineNumberAtOffset(content: string, offset: number) {
  let line = 1
  for (let index = 0; index < offset; index += 1) {
    if (content[index] === '\n') {
      line += 1
    }
  }
  return line
}

function columnNumberAtOffset(content: string, offset: number) {
  let column = 1
  for (let index = 0; index < offset; index += 1) {
    if (content[index] === '\n') {
      column = 1
    } else {
      column += 1
    }
  }
  return column
}
