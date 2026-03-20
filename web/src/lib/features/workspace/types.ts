import type * as Contract from '$lib/api/contracts'

export type Organization = Contract.Organization
export type ProjectStatus = 'planning' | 'active' | 'paused' | 'archived'
export type Project = Contract.Project
export type AgentStatus = Contract.Agent['status']
export type Agent = Contract.Agent
export type ActivityEvent = Contract.ActivityEvent
export type TicketReference = Contract.TicketReference
export type TicketDependency = Contract.TicketDependency
export type Ticket = Contract.Ticket
export type TicketPriority = Contract.TicketPriority
export type TicketStatus = Contract.TicketStatus

export type WorkflowType =
  | 'coding'
  | 'test'
  | 'doc'
  | 'security'
  | 'deploy'
  | 'refine-harness'
  | 'custom'

export type Workflow = Contract.Workflow
export type HarnessDocument = Contract.HarnessDocument
export type HarnessValidationIssue = Contract.HarnessValidationIssue
export type HarnessValidationResponse = Contract.HarnessValidationResponse
export type SkillBinding = Contract.SkillBinding
export type Skill = Contract.Skill
export type BuiltinRole = Contract.BuiltinRole
export type HRAdvisorSummary = Contract.HRAdvisorSummary
export type HRAdvisorStaffing = Contract.HRAdvisorStaffing
export type HRAdvisorRecommendation = Contract.HRAdvisorRecommendation
export type HRAdvisorPayload = Contract.HRAdvisorResponse
export type HRAdvisorResponse = Contract.HRAdvisorResponse
export type OrganizationPayload = Contract.OrganizationPayload
export type OrganizationResponse = Contract.OrganizationResponse
export type AgentProvider = Contract.AgentProvider
export type AgentProviderListPayload = Contract.AgentProviderListPayload
export type ProjectPayload = Contract.ProjectPayload
export type ProjectResponse = Contract.ProjectResponse
export type AgentPayload = Contract.AgentPayload
export type ActivityPayload = Contract.ActivityPayload
export type StatusPayload = Contract.StatusPayload
export type TicketPayload = Contract.TicketPayload
export type TicketResponse = Contract.TicketResponse
export type WorkflowListPayload = Contract.WorkflowListPayload
export type WorkflowDetailPayload = Contract.WorkflowDetailPayload
export type WorkflowResponse = Contract.WorkflowResponse
export type HarnessPayload = Contract.HarnessPayload
export type WorkflowSkillBindingResponse = Contract.WorkflowSkillBindingResponse
export type SkillListPayload = Contract.SkillListPayload
export type BuiltinRolePayload = Contract.BuiltinRolePayload

export type OnboardingMilestoneKey =
  | 'organization'
  | 'project'
  | 'workflow-lane'
  | 'ticket'
  | 'agent'
  | 'automation-signal'

export type OnboardingMilestone = {
  key: OnboardingMilestoneKey
  title: string
  description: string
  action: string
  completed: boolean
  isCurrent: boolean
}

export type OnboardingSnapshot = {
  organizationCount: number
  projectCount: number
  selectedOrgName: string
  selectedProjectName: string
  statusCount: number
  workflowCount: number
  ticketCount: number
  agentCount: number
  runningAgentCount: number
  activityCount: number
  hasAutomationSignal: boolean
}

export type OnboardingSummary = {
  complete: boolean
  completedCount: number
  totalCount: number
  progressPercent: number
  title: string
  description: string
  actionLabel: string
  stats: Array<{ label: string; value: string }>
  milestones: OnboardingMilestone[]
}

export type StreamEnvelope = {
  topic: string
  type: string
  payload?: unknown
  published_at: string
}

export type OrganizationForm = {
  name: string
  slug: string
}

export type ProjectForm = {
  name: string
  slug: string
  description: string
  status: ProjectStatus
  maxConcurrentAgents: number
}

export type WorkflowForm = {
  name: string
  type: WorkflowType
  pickupStatusId: string
  finishStatusId: string
  maxConcurrent: number
  maxRetryAttempts: number
  timeoutMinutes: number
  stallTimeoutMinutes: number
  isActive: boolean
}
