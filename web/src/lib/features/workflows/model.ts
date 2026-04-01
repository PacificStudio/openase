import type { HarnessContent, WorkflowSummary } from './types'

export type SkillState = {
  name: string
  description: string
  path: string
  bound: boolean
}

export function normalizeWorkflowType(type: string): WorkflowSummary['type'] {
  if (
    type === 'coding' ||
    type === 'test' ||
    type === 'doc' ||
    type === 'security' ||
    type === 'deploy' ||
    type === 'refine-harness' ||
    type === 'custom'
  ) {
    return type
  }

  return 'custom'
}

export function extractFrontmatter(content: string) {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/)
  return match?.[1] ?? ''
}

export function extractBody(content: string) {
  const match = content.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?([\s\S]*)$/)
  return match?.[1] ?? content
}

export function toHarnessContent(content: string): HarnessContent {
  return {
    frontmatter: extractFrontmatter(content),
    body: extractBody(content),
    rawContent: content,
  }
}

export function parseHarnessTemplateStatusBindings(content: string) {
  const frontmatter = extractFrontmatter(content)
  const statusBlock = extractYamlSection(frontmatter, 'status')
  return {
    pickupStatusNames: parseYamlStringList(statusBlock, 'pickup'),
    finishStatusNames: parseYamlStringList(statusBlock, 'finish'),
  }
}

function extractYamlSection(frontmatter: string, sectionName: string): string[] {
  const lines = frontmatter.replaceAll('\r\n', '\n').split('\n')
  let sectionIndent = -1
  const sectionLines: string[] = []

  for (let index = 0; index < lines.length; index += 1) {
    const match = lines[index].match(/^(\s*)([^:#]+):\s*$/)
    if (!match) continue

    const [, indent, name] = match
    if (name.trim() !== sectionName) continue
    sectionIndent = indent.length

    for (let nextIndex = index + 1; nextIndex < lines.length; nextIndex += 1) {
      const line = lines[nextIndex]
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) {
        sectionLines.push(line)
        continue
      }

      const nextIndent = line.match(/^\s*/)?.[0].length ?? 0
      if (nextIndent <= sectionIndent) break
      sectionLines.push(line)
    }
    break
  }

  return sectionLines
}

function parseYamlStringList(sectionLines: string[], key: string): string[] {
  for (let index = 0; index < sectionLines.length; index += 1) {
    const match = sectionLines[index].match(/^(\s+)([^:#]+):\s*(.*)$/)
    if (!match || match[2].trim() !== key) continue

    const keyIndent = match[1].length
    const rawValue = match[3].trim()
    if (rawValue) {
      return dedupeStrings(parseYamlInlineStringList(rawValue))
    }

    const values: string[] = []
    for (let nextIndex = index + 1; nextIndex < sectionLines.length; nextIndex += 1) {
      const line = sectionLines[nextIndex]
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) continue

      const nextIndent = line.match(/^\s*/)?.[0].length ?? 0
      if (nextIndent <= keyIndent) break

      const itemMatch = line.match(/^\s*-\s*(.+?)\s*$/)
      if (!itemMatch) {
        throw new Error(`Template status.${key} must be a string or list of strings.`)
      }
      values.push(parseYamlScalar(itemMatch[1]))
    }
    return dedupeStrings(values)
  }

  return []
}

function parseYamlInlineStringList(rawValue: string): string[] {
  if (rawValue.startsWith('[')) {
    if (!rawValue.endsWith(']')) {
      throw new Error('Template status list must close with ].')
    }
    const inner = rawValue.slice(1, -1).trim()
    if (!inner) return []
    return inner
      .split(',')
      .map((item) => parseYamlScalar(item))
      .filter(Boolean)
  }

  return [parseYamlScalar(rawValue)]
}

function parseYamlScalar(rawValue: string): string {
  const trimmed = rawValue.trim()
  if (!trimmed) {
    throw new Error('Template status entries must not be empty.')
  }

  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1).trim()
  }

  return trimmed
}

function dedupeStrings(items: string[]): string[] {
  const seen = new Set<string>()
  const deduped: string[] = []
  for (const item of items) {
    const trimmed = item.trim()
    if (!trimmed) continue
    const key = trimmed.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    deduped.push(trimmed)
  }
  return deduped
}

export function defaultHarnessTemplate() {
  return `---\ntype: coding\n---\n\nYou are an OpenASE workflow.\n`
}
