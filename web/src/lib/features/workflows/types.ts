import type { TicketStatusStage } from '$lib/features/statuses/public'
import type { WorkflowHooksPayload } from './workflow-hooks'

export type WorkflowType = string

export type WorkflowFamily =
  | 'planning'
  | 'dispatcher'
  | 'coding'
  | 'review'
  | 'test'
  | 'docs'
  | 'deploy'
  | 'security'
  | 'harness'
  | 'environment'
  | 'research'
  | 'reporting'
  | 'unknown'

export type WorkflowClassification = {
  family: WorkflowFamily
  confidence: number
  reasons: string[]
}

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
  workflowFamily: WorkflowFamily
  classification: WorkflowClassification
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
  workflowFamily?: WorkflowFamily
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

export type ScopeGroup = {
  category: string
  scopes: string[]
}

export type WorkflowImpactSummary = {
  ticket_count: number
  scheduled_job_count: number
  active_agent_run_count: number
  historical_agent_run_count: number
  replaceable_reference_count: number
  blocking_reference_count: number
}

export type WorkflowTicketReference = {
  id: string
  identifier: string
  title: string
  status_id: string
  status_name: string
  current_run_id?: string | null
}

export type WorkflowScheduledJobReference = {
  id: string
  name: string
  is_enabled: boolean
}

export type WorkflowAgentRunReference = {
  id: string
  ticket_id: string
  ticket_identifier: string
  ticket_title: string
  status: string
  created_at: string
}

export type WorkflowReplaceableReferences = {
  tickets: WorkflowTicketReference[]
  scheduled_jobs: WorkflowScheduledJobReference[]
}

export type WorkflowBlockingReferences = {
  active_agent_runs: WorkflowAgentRunReference[]
  historical_agent_runs: WorkflowAgentRunReference[]
}

export type WorkflowImpact = {
  workflow_id: string
  can_retire: boolean
  can_replace_references: boolean
  can_purge: boolean
  summary: WorkflowImpactSummary
  replaceable_references: WorkflowReplaceableReferences
  blocking_references: WorkflowBlockingReferences
}

export type WorkflowReplaceReferencesResult = {
  workflow_id: string
  replacement_workflow_id: string
  ticket_count: number
  scheduled_job_count: number
  tickets: WorkflowTicketReference[]
  scheduled_jobs: WorkflowScheduledJobReference[]
}
