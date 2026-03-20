import type { ProjectStatus, WorkflowType } from './types'

export const agentConsoleLimit = 40

export const projectStatuses: ProjectStatus[] = ['planning', 'active', 'paused', 'archived']

export const workflowTypes: WorkflowType[] = [
  'coding',
  'test',
  'doc',
  'security',
  'deploy',
  'refine-harness',
  'custom',
]

export const inputClass =
  'w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10'

export const textAreaClass =
  'min-h-32 w-full rounded-2xl border border-border/70 bg-background/80 px-4 py-3 text-sm outline-none transition focus:border-foreground/40 focus:ring-2 focus:ring-foreground/10'

export const editorPlaceholder = `---
workflow:
  name: "coding"
  type: "coding"
status:
  pickup: "Todo"
  finish: "Done"
---

# Coding Workflow

You are handling {{ ticket.identifier }}.
`
