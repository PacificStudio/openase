import type {
  HarnessContent,
  WorkflowClassification,
  WorkflowFamily,
  WorkflowStatusOption,
  WorkflowSummary,
} from './types'

export type SkillState = {
  id: string
  name: string
  description: string
  path: string
  bound: boolean
}

const knownWorkflowFamilies: WorkflowFamily[] = [
  'planning',
  'dispatcher',
  'coding',
  'review',
  'test',
  'docs',
  'deploy',
  'security',
  'harness',
  'environment',
  'research',
  'reporting',
  'unknown',
]

export const workflowFamilyColors: Record<WorkflowFamily, string> = {
  planning: 'bg-sky-500/15 text-sky-400 border-sky-500/20',
  dispatcher: 'bg-neutral-500/15 text-neutral-300 border-neutral-500/20',
  coding: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  review: 'bg-fuchsia-500/15 text-fuchsia-400 border-fuchsia-500/20',
  test: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20',
  docs: 'bg-violet-500/15 text-violet-400 border-violet-500/20',
  deploy: 'bg-amber-500/15 text-amber-400 border-amber-500/20',
  security: 'bg-rose-500/15 text-rose-400 border-rose-500/20',
  harness: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/20',
  environment: 'bg-teal-500/15 text-teal-400 border-teal-500/20',
  research: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/20',
  reporting: 'bg-orange-500/15 text-orange-400 border-orange-500/20',
  unknown: 'bg-neutral-500/15 text-neutral-400 border-neutral-500/20',
}

export const workflowFamilyIcons: Record<WorkflowFamily, string> = {
  planning: '🧭',
  dispatcher: '🗂️',
  coding: '💻',
  review: '🧾',
  test: '🧪',
  docs: '📝',
  deploy: '🚀',
  security: '🔒',
  harness: '🔧',
  environment: '🛠️',
  research: '🔬',
  reporting: '📄',
  unknown: '⚙️',
}

export const workflowFamilyDescriptions: Record<WorkflowFamily, string> = {
  planning: 'Planning and scope definition',
  dispatcher: 'Ticket routing and lane assignment',
  coding: 'Implementation and delivery',
  review: 'Review and approval',
  test: 'QA and verification',
  docs: 'Documentation and guides',
  deploy: 'Release and rollout',
  security: 'Security review and hardening',
  harness: 'Harness and prompt optimization',
  environment: 'Environment setup and repair',
  research: 'Research and experiments',
  reporting: 'Reports and writeups',
  unknown: 'Unclassified workflow family',
}

export function normalizeWorkflowType(type: string): WorkflowSummary['type'] {
  return type.trim()
}

export function normalizeWorkflowFamily(family: string): WorkflowFamily {
  return knownWorkflowFamilies.includes(family as WorkflowFamily)
    ? (family as WorkflowFamily)
    : 'unknown'
}

export function normalizeWorkflowClassification(
  classification:
    | Partial<WorkflowClassification>
    | {
        family?: string
        confidence?: number
        reasons?: unknown[]
      }
    | null
    | undefined,
  fallbackFamily: string,
): WorkflowClassification {
  const reasons = (Array.isArray(classification?.reasons) ? classification.reasons : []).reduce<
    string[]
  >((items, value) => {
    if (typeof value === 'string') {
      items.push(value)
    }
    return items
  }, [])
  return {
    family: normalizeWorkflowFamily(classification?.family ?? fallbackFamily),
    confidence:
      typeof classification?.confidence === 'number' && Number.isFinite(classification.confidence)
        ? classification.confidence
        : 0,
    reasons,
  }
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

export function resolveHarnessTemplateStatusSelection(
  content: string,
  statuses: WorkflowStatusOption[],
) {
  try {
    const bindings = parseHarnessTemplateStatusBindings(content)
    const pickupStatusIds = resolveTemplateStatusIds(bindings.pickupStatusNames, statuses)
    const finishStatusIds = resolveTemplateStatusIds(bindings.finishStatusNames, statuses)
    const missingStatusNames = [...pickupStatusIds.missingNames, ...finishStatusIds.missingNames]

    if (missingStatusNames.length > 0) {
      return {
        pickupStatusIds: [] as string[],
        finishStatusIds: [] as string[],
        error: `Template status bindings are not configured in this project: ${[...new Set(missingStatusNames)].join(', ')}.`,
      }
    }

    return {
      pickupStatusIds: pickupStatusIds.ids,
      finishStatusIds: finishStatusIds.ids,
      error: '',
    }
  } catch (caughtError) {
    return {
      pickupStatusIds: [] as string[],
      finishStatusIds: [] as string[],
      error:
        caughtError instanceof Error
          ? caughtError.message
          : 'Failed to parse workflow template status bindings.',
    }
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

function resolveTemplateStatusIds(names: string[], statuses: WorkflowStatusOption[]) {
  const ids: string[] = []
  const missingNames: string[] = []

  for (const name of names) {
    const status = statuses.find(
      (item) => item.name.trim().toLowerCase() === name.trim().toLowerCase(),
    )
    if (!status) {
      missingNames.push(name)
      continue
    }
    if (!ids.includes(status.id)) {
      ids.push(status.id)
    }
  }

  return { ids, missingNames }
}

export function defaultHarnessTemplate() {
  return `---\nworkflow:\n  type: "Workflow"\n---\n\nYou are an OpenASE workflow.\n`
}
