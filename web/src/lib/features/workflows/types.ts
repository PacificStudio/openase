import type { TicketStatusStage } from '$lib/features/statuses/public'
import type { WorkflowHooksPayload } from './workflow-hooks'

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
  roleSlug?: string
  roleName?: string
  roleDescription?: string
  platformAccessAllowed?: string[]
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
  hooks?: WorkflowHooksPayload
  rawHooks?: Record<string, unknown>
}

export type WorkflowStatusOption = {
  id: string
  name: string
  stage: TicketStatusStage
}

export type WorkflowTemplateDraft = {
  name: string
  content: string
  workflowType: WorkflowSummary['type']
  roleSlug?: string
  roleName?: string
  roleDescription?: string
  platformAccessAllowed?: string[]
  skillNames?: string[]
  pickupStatusNames?: string[]
  finishStatusNames?: string[]
  harnessPath?: string | null
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
