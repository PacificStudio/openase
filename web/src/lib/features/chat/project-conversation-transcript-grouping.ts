import type {
  ProjectConversationCommandOutputEntry,
  ProjectConversationToolCallEntry,
  ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'

/**
 * A standalone entry rendered on its own (text, interrupt, action_proposal, diff).
 */
export type StandaloneItem = {
  type: 'standalone'
  entry: ProjectConversationTranscriptEntry
}

/**
 * A group of consecutive system entries (tool_call, command_output, task_status)
 * rendered as a single collapsible operation block.
 */
export type OperationGroup = {
  type: 'operation_group'
  id: string
  entries: ProjectConversationTranscriptEntry[]
  summary: string
  detail: string
}

export type TranscriptDisplayItem = StandaloneItem | OperationGroup

const SYSTEM_ENTRY_KINDS = new Set(['tool_call', 'command_output', 'task_status'])

function isSystemEntry(entry: ProjectConversationTranscriptEntry): boolean {
  return SYSTEM_ENTRY_KINDS.has(entry.kind)
}

/**
 * Summarize a tool call to a human-readable one-liner.
 */
export function summarizeToolCall(entry: ProjectConversationToolCallEntry): string {
  const args =
    entry.arguments && typeof entry.arguments === 'object' && !Array.isArray(entry.arguments)
      ? (entry.arguments as Record<string, unknown>)
      : null

  const readString = (...keys: string[]): string => {
    for (const key of keys) {
      const value = args?.[key]
      if (typeof value === 'string' && value.trim()) return value.trim()
    }
    return ''
  }

  switch (entry.tool) {
    case 'functions.exec_command': {
      const cmd = readString('cmd', 'command')
      return cmd ? `Ran \`${truncateInline(cmd, 60)}\`` : 'Ran command'
    }
    case 'functions.apply_patch': {
      const target = readString('path', 'file', 'target')
      return target ? `Applied patch to \`${shortenPath(target)}\`` : 'Applied patch'
    }
    case 'functions.write_stdin': {
      return 'Sent stdin input'
    }
    default: {
      // Try to detect read-like / search-like / list-like calls from the tool name
      const toolName = entry.tool.replace(/^functions\./, '')
      const target = readString('path', 'file', 'target', 'name')
      if (target) return `${capitalize(toolName)} \`${shortenPath(target)}\``
      return capitalize(toolName)
    }
  }
}

function summarizeCommandOutput(entry: ProjectConversationCommandOutputEntry): string {
  const command = entry.command?.trim()
  if (!command) {
    return 'Command output'
  }
  return `Ran \`${truncateInline(command, 60)}\``
}

/**
 * Determine if a tool call is an "exploring" operation (read, search, list).
 */
export function isExploringToolCall(entry: ProjectConversationToolCallEntry): boolean {
  const tool = entry.tool.toLowerCase()
  return (
    tool.includes('read') ||
    tool.includes('search') ||
    tool.includes('grep') ||
    tool.includes('list') ||
    tool.includes('glob') ||
    tool.includes('find')
  )
}

/**
 * Build a summary line for an operation group.
 */
function buildGroupSummary(entries: ProjectConversationTranscriptEntry[]): {
  summary: string
  detail: string
} {
  const toolCalls = entries.filter(
    (e): e is ProjectConversationToolCallEntry => e.kind === 'tool_call',
  )
  const outputs = entries.filter(
    (e): e is ProjectConversationCommandOutputEntry => e.kind === 'command_output',
  )

  if (toolCalls.length === 0) {
    if (outputs.length > 0) {
      return {
        summary: summarizeCommandOutput(outputs[0]),
        detail: `${outputs.length} output block(s)`,
      }
    }
    return { summary: 'System activity', detail: `${entries.length} event(s)` }
  }

  // Check if all tool calls are exploring operations
  const allExploring = toolCalls.every(isExploringToolCall)
  if (allExploring && toolCalls.length > 1) {
    return {
      summary: `Explored ${toolCalls.length} files`,
      detail: toolCalls.map((tc) => summarizeToolCall(tc)).join(', '),
    }
  }

  if (toolCalls.length === 1) {
    return {
      summary: summarizeToolCall(toolCalls[0]),
      detail: outputs.length > 0 ? `${outputs.length} output block(s)` : '',
    }
  }

  // Mixed operations
  const execCalls = toolCalls.filter((tc) => tc.tool === 'functions.exec_command')
  const patchCalls = toolCalls.filter((tc) => tc.tool === 'functions.apply_patch')
  const exploreCalls = toolCalls.filter((tc) => isExploringToolCall(tc))

  const parts: string[] = []
  if (execCalls.length > 0) parts.push(`${execCalls.length} command(s)`)
  if (patchCalls.length > 0) parts.push(`${patchCalls.length} patch(es)`)
  if (exploreCalls.length > 0) parts.push(`${exploreCalls.length} read(s)`)

  return {
    summary: `${toolCalls.length} operations`,
    detail: parts.join(', '),
  }
}

/**
 * Groups transcript entries into display items.
 * Consecutive system entries are merged into operation groups.
 * Everything else is a standalone item.
 */
export function groupTranscriptEntries(
  entries: ProjectConversationTranscriptEntry[],
): TranscriptDisplayItem[] {
  const items: TranscriptDisplayItem[] = []
  let currentGroup: ProjectConversationTranscriptEntry[] = []

  function flushGroup() {
    if (currentGroup.length === 0) return

    const { summary, detail } = buildGroupSummary(currentGroup)
    items.push({
      type: 'operation_group',
      id: `group-${currentGroup[0].id}`,
      entries: currentGroup,
      summary,
      detail,
    })
    currentGroup = []
  }

  for (const entry of entries) {
    if (isSystemEntry(entry)) {
      currentGroup.push(entry)
    } else {
      flushGroup()
      items.push({ type: 'standalone', entry })
    }
  }

  flushGroup()
  return items
}

// --- Helpers ---

function truncateInline(text: string, maxLength: number): string {
  const singleLine = text.replace(/\n/g, ' ').trim()
  if (singleLine.length <= maxLength) return singleLine
  return `${singleLine.slice(0, maxLength - 3)}...`
}

function shortenPath(path: string): string {
  const parts = path.split('/')
  if (parts.length <= 3) return path
  return `.../${parts.slice(-2).join('/')}`
}

function capitalize(text: string): string {
  if (!text) return ''
  return text.charAt(0).toUpperCase() + text.slice(1).replace(/_/g, ' ')
}

/**
 * Truncate command output text using head + tail with middle ellipsis.
 * Returns { head, omitted, tail } or null if no truncation needed.
 */
export function truncateOutput(
  text: string,
  headLines = 5,
  tailLines = 5,
): { head: string; omitted: number; tail: string } | null {
  const lines = text.split('\n')
  const total = lines.length
  if (total <= headLines + tailLines) return null

  const omitted = total - headLines - tailLines
  return {
    head: lines.slice(0, headLines).join('\n'),
    omitted,
    tail: lines.slice(total - tailLines).join('\n'),
  }
}
