export type Organization = {
  id: string
  name: string
  slug: string
  default_agent_provider_id?: string | null
}
export type ProjectStatus = 'planning' | 'active' | 'paused' | 'archived'

export type Project = {
  id: string
  organization_id: string
  name: string
  slug: string
  description: string
  status: ProjectStatus
  default_workflow_id?: string | null
  default_agent_provider_id?: string | null
  max_concurrent_agents: number
}

export type AgentStatus = 'idle' | 'claimed' | 'running' | 'failed' | 'terminated'

export type Agent = {
  id: string
  provider_id: string
  project_id: string
  name: string
  status: AgentStatus
  current_ticket_id?: string | null
  session_id: string
  workspace_path: string
  capabilities: string[]
  total_tokens_used: number
  total_tickets_completed: number
  last_heartbeat_at?: string | null
}

export type ActivityEvent = {
  id: string
  project_id: string
  ticket_id?: string | null
  agent_id?: string | null
  event_type: string
  message: string
  metadata: Record<string, unknown>
  created_at: string
}

export type TicketReference = {
  id: string
  identifier: string
  title: string
  status_id: string
  status_name: string
}

export type TicketDependency = {
  id: string
  type: string
  target: TicketReference
}

export type TicketPriority = 'urgent' | 'high' | 'medium' | 'low'

export type Ticket = {
  id: string
  project_id: string
  identifier: string
  title: string
  description: string
  status_id: string
  status_name: string
  priority: TicketPriority
  type: string
  workflow_id?: string | null
  created_by: string
  parent?: TicketReference | null
  children: TicketReference[]
  dependencies: TicketDependency[]
  external_ref: string
  budget_usd: number
  cost_amount: number
  attempt_count: number
  consecutive_errors: number
  next_retry_at?: string | null
  retry_paused: boolean
  pause_reason: string
  created_at: string
}

export type TicketStatus = {
  id: string
  project_id: string
  name: string
  color: string
  icon?: string
  position: number
  is_default: boolean
  description: string
}

export type WorkflowType =
  | 'coding'
  | 'test'
  | 'doc'
  | 'security'
  | 'deploy'
  | 'refine-harness'
  | 'custom'

export type Workflow = {
  id: string
  project_id: string
  name: string
  type: WorkflowType
  harness_path: string
  harness_content?: string | null
  hooks: Record<string, unknown>
  max_concurrent: number
  max_retry_attempts: number
  timeout_minutes: number
  stall_timeout_minutes: number
  version: number
  is_active: boolean
  pickup_status_id: string
  finish_status_id?: string | null
}

export type HarnessDocument = {
  workflow_id: string
  path: string
  content: string
  version: number
}

export type HarnessValidationIssue = {
  level: 'error' | 'warning' | string
  message: string
  line?: number
  column?: number
}

export type HarnessValidationResponse = {
  valid: boolean
  issues: HarnessValidationIssue[]
}

export type SkillBinding = {
  id: string
  name: string
  harness_path: string
}

export type Skill = {
  name: string
  description: string
  path: string
  is_builtin: boolean
  bound_workflows: SkillBinding[]
}

export type BuiltinRole = {
  slug: string
  name: string
  workflow_type: WorkflowType
  summary: string
  harness_path: string
  content: string
}

export type HRAdvisorSummary = {
  open_tickets: number
  coding_tickets: number
  failing_tickets: number
  blocked_tickets: number
  active_agents: number
  workflow_count: number
  recent_activity_count: number
  active_workflow_types: string[]
}

export type HRAdvisorStaffing = {
  developers: number
  qa: number
  docs: number
  security: number
  product: number
  research: number
}

export type HRAdvisorRecommendation = {
  role_slug: string
  role_name: string
  workflow_type: WorkflowType
  summary: string
  harness_path: string
  priority: 'high' | 'medium' | 'low'
  reason: string
  evidence: string[]
  suggested_headcount: number
  suggested_workflow_name: string
  activation_ready: boolean
  active_workflow_name?: string | null
}

export type HRAdvisorPayload = {
  project_id: string
  summary: HRAdvisorSummary
  staffing: HRAdvisorStaffing
  recommendations: HRAdvisorRecommendation[]
}

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

export type OrganizationPayload = { organizations: Organization[] }
export type ProjectPayload = { projects: Project[] }
export type AgentPayload = { agents: Agent[] }
export type ActivityPayload = { events: ActivityEvent[] }
export type StatusPayload = { statuses: TicketStatus[] }
export type TicketPayload = { tickets: Ticket[] }
export type WorkflowListPayload = { workflows: Workflow[] }
export type WorkflowDetailPayload = { workflow: Workflow }
export type HarnessPayload = { harness: HarnessDocument }
export type SkillListPayload = { skills: Skill[] }
export type BuiltinRolePayload = { roles: BuiltinRole[] }
export type HRAdvisorResponse = HRAdvisorPayload

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
