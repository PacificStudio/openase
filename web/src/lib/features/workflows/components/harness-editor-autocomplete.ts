import type { HarnessVariableGroup, HarnessVariableMetadata } from '../types'

export type Suggestion = {
  id: string
  kind: 'variable' | 'filter'
  groupName: string
  label: string
  insertText: string
  description: string
  example?: string
}

export type CompletionState = {
  mode: 'variable' | 'filter'
  query: string
  tokenStart: number
}

export function flattenSuggestions(groups: HarnessVariableGroup[]): Suggestion[] {
  return groups.flatMap((group) => group.variables.map((item) => mapSuggestion(group.name, item)))
}

export function filterSuggestions(
  items: Suggestion[],
  state: CompletionState | null,
): Suggestion[] {
  if (!state) {
    return []
  }

  const normalizedQuery = state.query.trim().toLowerCase()
  return items
    .filter((item) => item.kind === state.mode)
    .filter((item) => {
      if (!normalizedQuery) {
        return true
      }
      const label = item.label.toLowerCase()
      return label.startsWith(normalizedQuery) || label.includes(normalizedQuery)
    })
    .sort((left, right) => {
      const leftStarts = left.label.toLowerCase().startsWith(normalizedQuery)
      const rightStarts = right.label.toLowerCase().startsWith(normalizedQuery)
      if (leftStarts !== rightStarts) {
        return leftStarts ? -1 : 1
      }
      return left.label.localeCompare(right.label)
    })
    .slice(0, 8)
}

export function findCompletionState(rawContent: string, cursor: number): CompletionState | null {
  const beforeCursor = rawContent.slice(0, cursor)
  const expressionStart = Math.max(beforeCursor.lastIndexOf('{{'), beforeCursor.lastIndexOf('{%'))
  if (expressionStart === -1) {
    return null
  }

  const latestClose = Math.max(beforeCursor.lastIndexOf('}}'), beforeCursor.lastIndexOf('%}'))
  if (latestClose > expressionStart) {
    return null
  }

  const segment = beforeCursor.slice(expressionStart + 2)
  if (segment.includes('\n')) {
    return null
  }

  const pipeIndex = segment.lastIndexOf('|')
  if (pipeIndex >= 0) {
    const afterPipe = segment.slice(pipeIndex + 1)
    const trimmed = afterPipe.replace(/^\s+/, '')
    return {
      mode: 'filter',
      query: trimmed,
      tokenStart: cursor - trimmed.length,
    }
  }

  const match = segment.match(/([A-Za-z_][\w.[\]]*)?$/)
  if (!match) {
    return null
  }

  const query = match[1] ?? ''
  return {
    mode: 'variable',
    query,
    tokenStart: cursor - query.length,
  }
}

function mapSuggestion(groupName: string, item: HarnessVariableMetadata): Suggestion {
  return {
    id: `${groupName}:${item.path}`,
    kind: item.type === 'filter' || groupName === 'filters' ? 'filter' : 'variable',
    groupName,
    label: item.path,
    insertText: item.path,
    description: item.description,
    example: item.example,
  }
}
