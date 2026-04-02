import type { ChatDiffHunk, ChatDiffPayload } from '$lib/api/chat'

export function normalizeDiffPayload(
  payload: Record<string, unknown> | ChatDiffPayload,
  entryId: string,
): ChatDiffPayload {
  const object = asRecord(payload) ?? {}

  return {
    type: 'diff',
    entryId,
    file: readString(object, 'file') || '',
    hunks: readDiffHunks(object.hunks),
  }
}

export function buildProviderStateDetail(raw: Record<string, unknown> | null) {
  const status = readString(raw, 'status')
  const detail = readString(raw, 'detail')
  const flags = readStringList(raw, 'active_flags')

  const parts = [status, detail, flags.length > 0 ? flags.join(', ') : undefined].filter(Boolean)
  return parts.length > 0 ? parts.join(' · ') : undefined
}

export function buildTaskDetail(raw: Record<string, unknown> | null) {
  return (
    readString(raw, 'message') ||
    readString(raw, 'text') ||
    describeStream(raw) ||
    describeStatus(raw)
  )
}

export function buildReasoningDetail(raw: Record<string, unknown> | null) {
  const delta = readString(raw, 'delta')
  if (delta) {
    return delta
  }

  const kind = readString(raw, 'kind')
  if (!kind) {
    return undefined
  }
  return `Kind: ${kind.replace(/_/g, ' ')}`
}

export function parseUnifiedDiffPayloads(diffText: string): ChatDiffPayload[] {
  const lines = diffText.replaceAll('\r\n', '\n').split('\n')
  const files: ChatDiffPayload[] = []
  let current: ChatDiffPayload | null = null
  let currentHunk: ChatDiffHunk | null = null

  const pushCurrentHunk = () => {
    if (current && currentHunk && currentHunk.lines.length > 0) {
      current.hunks.push(currentHunk)
    }
    currentHunk = null
  }

  const pushCurrentFile = () => {
    pushCurrentHunk()
    if (current && current.file && current.hunks.length > 0) {
      files.push(current)
    }
    current = null
  }

  for (const line of lines) {
    if (line.startsWith('diff --git ')) {
      pushCurrentFile()
      current = {
        type: 'diff',
        file: parseDiffFilePath(line),
        hunks: [],
      }
      continue
    }

    if (!current) {
      continue
    }

    if (line.startsWith('@@ ')) {
      pushCurrentHunk()
      const header = parseHunkHeader(line)
      if (!header) {
        continue
      }
      currentHunk = {
        oldStart: header.oldStart,
        oldLines: header.oldLines,
        newStart: header.newStart,
        newLines: header.newLines,
        lines: [],
      }
      continue
    }

    if (!currentHunk || line === '\\ No newline at end of file') {
      continue
    }

    const prefix = line[0]
    if (prefix === ' ' || prefix === '+' || prefix === '-') {
      currentHunk.lines.push({
        op: prefix === ' ' ? 'context' : prefix === '+' ? 'add' : 'remove',
        text: line.slice(1),
      })
    }
  }

  pushCurrentFile()
  return files
}

export function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null
  }
  return value as Record<string, unknown>
}

export function readString(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key]
  return typeof value === 'string' ? value : undefined
}

export function readBoolean(record: Record<string, unknown> | null, key: string) {
  return record?.[key] === true
}

function readDiffHunks(value: unknown): ChatDiffHunk[] {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => {
      const hunk = asRecord(item)
      if (!hunk) {
        return null
      }

      const lines = Array.isArray(hunk.lines)
        ? hunk.lines
            .map((line) => {
              const record = asRecord(line)
              const op = readString(record, 'op')
              const text = readString(record, 'text')
              if (!record || !op || text == null) {
                return null
              }
              if (op !== 'context' && op !== 'add' && op !== 'remove') {
                return null
              }

              return { op, text } as ChatDiffHunk['lines'][number]
            })
            .filter((line): line is ChatDiffHunk['lines'][number] => line != null)
        : []

      return {
        oldStart: readNumber(hunk, 'oldStart', 'old_start') ?? 0,
        oldLines: readNumber(hunk, 'oldLines', 'old_lines') ?? 0,
        newStart: readNumber(hunk, 'newStart', 'new_start') ?? 0,
        newLines: readNumber(hunk, 'newLines', 'new_lines') ?? 0,
        lines,
      } satisfies ChatDiffHunk
    })
    .filter((hunk): hunk is ChatDiffHunk => hunk != null)
}

function describeStream(raw: Record<string, unknown> | null) {
  const stream = readString(raw, 'stream')
  const phase = readString(raw, 'phase')
  if (!stream && !phase) {
    return undefined
  }
  return [stream, phase].filter(Boolean).join(' / ')
}

function describeStatus(raw: Record<string, unknown> | null) {
  const status = readString(raw, 'status')
  return status ? `Status: ${status}` : undefined
}

function parseDiffFilePath(header: string) {
  const match = /^diff --git a\/(.+?) b\/(.+)$/.exec(header)
  if (!match) {
    return ''
  }
  return match[2].trim()
}

function parseHunkHeader(line: string) {
  const match = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(line)
  if (!match) {
    return null
  }
  return {
    oldStart: Number.parseInt(match[1], 10),
    oldLines: Number.parseInt(match[2] || '1', 10),
    newStart: Number.parseInt(match[3], 10),
    newLines: Number.parseInt(match[4] || '1', 10),
  }
}

function readNumber(record: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
  }
  return undefined
}

function readStringList(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
}
