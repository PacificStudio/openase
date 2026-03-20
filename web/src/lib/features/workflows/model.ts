import type { WorkflowSummary } from './types'

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
  const match = content.match(/^---\n([\s\S]*?)\n---/)
  return match?.[1] ?? ''
}

export function extractBody(content: string) {
  const match = content.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  return match?.[1] ?? content
}

export function defaultHarnessTemplate() {
  return `---\ntype: coding\n---\n\nYou are an OpenASE workflow.\n`
}
