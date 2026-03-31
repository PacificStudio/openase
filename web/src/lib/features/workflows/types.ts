export type WorkflowType =
  | 'coding'
  | 'test'
  | 'doc'
  | 'security'
  | 'deploy'
  | 'refine-harness'
  | 'custom'

export type WorkflowVersionSummary = {
  id: string
  version: number
  createdBy: string
  createdAt: string
}

export type WorkflowSummary = {
  id: string
  name: string
  type: WorkflowType
  agentId: string | null
  harnessPath: string
  pickupStatusIds: string[]
  pickupStatusLabel: string
  finishStatusIds: string[]
  finishStatusLabel: string
  maxConcurrent: number
  maxRetry: number
  timeoutMinutes: number
  stallTimeoutMinutes: number
  isActive: boolean
  lastModified: string
  recentSuccessRate: number
  version: number
  history: WorkflowVersionSummary[]
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
  machineName: string
  adapterType: string
  available: boolean
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
