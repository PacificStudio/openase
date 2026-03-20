export type WorkflowType =
  | 'coding'
  | 'test'
  | 'doc'
  | 'security'
  | 'deploy'
  | 'refine-harness'
  | 'custom'

export type WorkflowSummary = {
  id: string
  name: string
  type: WorkflowType
  pickupStatus: string
  finishStatus: string
  maxConcurrent: number
  maxRetry: number
  timeoutMinutes: number
  isActive: boolean
  lastModified: string
  recentSuccessRate: number
  version: number
}

export type HarnessContent = {
  frontmatter: string
  body: string
  rawContent: string
}
