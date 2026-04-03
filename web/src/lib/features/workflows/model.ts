import type { HarnessContent, WorkflowStatusOption, WorkflowSummary } from './types'

export type SkillState = {
  id: string
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

export function toHarnessContent(content: string): HarnessContent {
  return {
    frontmatter: '',
    body: content,
    rawContent: content,
  }
}

export function resolveTemplateStatusSelection(
  pickupStatusNames: readonly string[] | null | undefined,
  finishStatusNames: readonly string[] | null | undefined,
  statuses: WorkflowStatusOption[],
) {
  const pickupStatusIds = resolveTemplateStatusIds(pickupStatusNames, statuses)
  const finishStatusIds = resolveTemplateStatusIds(finishStatusNames, statuses)
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

function resolveTemplateStatusIds(
  names: readonly string[] | null | undefined,
  statuses: WorkflowStatusOption[],
) {
  const ids: string[] = []
  const missingNames: string[] = []

  for (const name of names ?? []) {
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
  return `# Workflow\n\nDescribe the role, constraints, and expected outputs for this workflow.\n`
}
