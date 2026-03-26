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
  agentId: string | null
  harnessPath: string
  requiredMachineLabels: string[]
  pickupStatusId: string
  pickupStatus: string
  finishStatusId: string | null
  finishStatus: string
  maxConcurrent: number
  maxRetry: number
  timeoutMinutes: number
  stallTimeoutMinutes: number
  isActive: boolean
  lastModified: string
  recentSuccessRate: number
  version: number
}

export type WorkflowStatusOption = {
  id: string
  name: string
}

export type WorkflowAgentOption = {
  id: string
  label: string
  agentName: string
  providerName: string
  modelName: string
  workspacePath: string
}

export type HarnessContent = {
  frontmatter: string
  body: string
  rawContent: string
}

export type HarnessVariableMetadata = {
  path: string
  type: string
  description: string
  example?: string
}

export type HarnessVariableGroup = {
  name: string
  variables: HarnessVariableMetadata[]
}
