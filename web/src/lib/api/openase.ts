import { api } from './client'
import type {
  ActivityPayload,
  AgentPayload,
  AgentRunPayload,
  AgentProviderResponse,
  AgentOutputPayload,
  AgentStepPayload,
  AgentResponse,
  AgentProvider,
  AgentProviderListPayload,
  BuiltinRolePayload,
  HarnessPayload,
  HarnessVariableDictionaryPayload,
  HarnessValidationResponse,
  MachineCreateResponse,
  MachinePayload,
  MachineResourcesResponse,
  MachineResponse,
  MachineTestResponse,
  NotificationChannelDeleteResponse,
  NotificationChannelPayload,
  NotificationChannelResponse,
  NotificationChannelTestResponse,
  NotificationRuleDeleteResponse,
  NotificationRuleEventTypesPayload,
  NotificationRulePayload,
  NotificationRuleResponse,
  ProjectRepoPayload,
  ProjectArchiveResponse,
  ProjectRepoResponse,
  ProjectCreateResponse,
  ProjectPayload,
  ProjectResponse,
  ScheduledJobDeleteResponse,
  ScheduledJobListPayload,
  ScheduledJobResponse,
  ScheduledJobTriggerResponse,
  ScheduledJobUpdateResponse,
  SecuritySettingsResponse,
  SkillListPayload,
  StatusDeleteResponse,
  StatusPayload,
  StatusResetPayload,
  StatusResponse,
  SystemDashboardResponse,
  TicketDetailPayload,
  TicketDependencyDeleteResponse,
  TicketDependencyResponse,
  TicketCreateResponse,
  TicketCommentCreateResponse,
  TicketCommentDeleteResponse,
  TicketCommentUpdateResponse,
  TicketExternalLinkDeleteResponse,
  TicketExternalLinkResponse,
  TicketPayload,
  HRAdvisorActivationResponse,
  HRAdvisorResponse,
  Organization,
  OrganizationArchiveResponse,
  OrganizationResponse,
  OrganizationUpdateResponse,
  TicketRepoScopePayload,
  TicketRepoScopeResponse,
  WorkflowDetailPayload,
  WorkflowDeleteResponse,
  WorkflowListPayload,
  WorkflowUpdateResponse,
} from './contracts'

type MachineMutationBody = {
  agent_cli_path?: string
  description?: string
  env_vars?: string[]
  host?: string
  labels?: string[]
  name?: string
  port?: number
  ssh_key_path?: string
  ssh_user?: string
  status?: string
  workspace_root?: string
}

export function getSystemDashboard() {
  return api.get<SystemDashboardResponse>('/api/v1/system/dashboard')
}

export function listOrganizations() {
  return api.get<{ organizations?: Organization[] }>('/api/v1/orgs')
}

export function createOrganization(body: {
  name: string
  slug: string
  default_agent_provider_id?: string | null
}) {
  return api.post<OrganizationResponse>('/api/v1/orgs', { body })
}

export function updateOrganization(
  orgId: string,
  body: {
    name?: string | null
    slug?: string | null
    default_agent_provider_id?: string | null
  },
) {
  return api.patch<OrganizationUpdateResponse>(`/api/v1/orgs/${orgId}`, { body })
}

export function archiveOrganization(orgId: string) {
  return api.delete<OrganizationArchiveResponse>(`/api/v1/orgs/${orgId}`)
}

export function listProjects(orgId: string) {
  return api.get<ProjectPayload>(`/api/v1/orgs/${orgId}/projects`)
}

export function createProject(
  orgId: string,
  body: {
    name: string
    slug: string
    description?: string
    status?: string
    default_workflow_id?: string | null
    default_agent_provider_id?: string | null
    accessible_machine_ids?: string[]
    max_concurrent_agents?: number
  },
) {
  return api.post<ProjectCreateResponse>(`/api/v1/orgs/${orgId}/projects`, { body })
}

export function listMachines(orgId: string) {
  return api.get<MachinePayload>(`/api/v1/orgs/${orgId}/machines`)
}

export function createMachine(orgId: string, body: MachineMutationBody) {
  return api.post<MachineCreateResponse>(`/api/v1/orgs/${orgId}/machines`, { body })
}

export function getMachine(machineId: string) {
  return api.get<MachineResponse>(`/api/v1/machines/${machineId}`)
}

export function updateMachine(machineId: string, body: MachineMutationBody) {
  return api.patch<MachineResponse>(`/api/v1/machines/${machineId}`, { body })
}

export function deleteMachine(machineId: string) {
  return api.delete<MachineResponse>(`/api/v1/machines/${machineId}`)
}

export function testMachineConnection(machineId: string) {
  return api.post<MachineTestResponse>(`/api/v1/machines/${machineId}/test`)
}

export function getMachineResources(machineId: string) {
  return api.get<MachineResourcesResponse>(`/api/v1/machines/${machineId}/resources`)
}

export function listProviders(orgId: string) {
  return api.get<AgentProviderListPayload>(`/api/v1/orgs/${orgId}/providers`)
}

export function createProvider(
  orgId: string,
  body: {
    machine_id: string
    name: string
    adapter_type: string
    cli_command?: string
    cli_args?: string[]
    auth_config?: Record<string, unknown>
    model_name: string
    model_temperature?: number
    model_max_tokens?: number
    cost_per_input_token?: number
    cost_per_output_token?: number
  },
) {
  return api.post<AgentProviderResponse>(`/api/v1/orgs/${orgId}/providers`, { body })
}

export function getProject(projectId: string) {
  return api.get<ProjectResponse>(`/api/v1/projects/${projectId}`)
}

export function getSecuritySettings(projectId: string) {
  return api.get<SecuritySettingsResponse>(`/api/v1/projects/${projectId}/security-settings`)
}

export function getHRAdvisor(projectId: string) {
  return api.get<HRAdvisorResponse>(`/api/v1/projects/${projectId}/hr-advisor`)
}

export function activateHRRecommendation(
  projectId: string,
  body: {
    role_slug: string
    create_bootstrap_ticket?: boolean | null
  },
) {
  return api.post<HRAdvisorActivationResponse>(
    `/api/v1/projects/${projectId}/hr-advisor/activate`,
    {
      body,
    },
  )
}

export function updateProject(
  projectId: string,
  body: {
    default_agent_provider_id?: string | null
    default_workflow_id?: string | null
    description?: string | null
    max_concurrent_agents?: number | null
    name?: string | null
    slug?: string | null
    status?: string | null
  },
) {
  return api.patch<ProjectResponse>(`/api/v1/projects/${projectId}`, { body })
}

export function archiveProject(projectId: string) {
  return api.delete<ProjectArchiveResponse>(`/api/v1/projects/${projectId}`)
}

export function listActivity(
  projectId: string,
  params?: {
    agent_id?: string
    ticket_id?: string
    limit?: number
  },
) {
  return api.get<ActivityPayload>(`/api/v1/projects/${projectId}/activity`, { params })
}

export function listAgents(projectId: string) {
  return api.get<AgentPayload>(`/api/v1/projects/${projectId}/agents`)
}

export function listAgentRuns(projectId: string) {
  return api.get<AgentRunPayload>(`/api/v1/projects/${projectId}/agent-runs`)
}

export function listAgentOutput(
  projectId: string,
  agentId: string,
  params?: {
    ticket_id?: string
    limit?: number
  },
) {
  return api.get<AgentOutputPayload>(`/api/v1/projects/${projectId}/agents/${agentId}/output`, {
    params,
  })
}

export function listAgentSteps(
  projectId: string,
  agentId: string,
  params?: {
    ticket_id?: string
    limit?: number
  },
) {
  return api.get<AgentStepPayload>(`/api/v1/projects/${projectId}/agents/${agentId}/steps`, {
    params,
  })
}

export function createAgent(
  projectId: string,
  body: {
    provider_id: string
    name: string
  },
) {
  return api.post<AgentResponse>(`/api/v1/projects/${projectId}/agents`, { body })
}

export function getAgent(agentId: string) {
  return api.get<AgentResponse>(`/api/v1/agents/${agentId}`)
}

export function pauseAgent(agentId: string) {
  return api.post<AgentResponse>(`/api/v1/agents/${agentId}/pause`)
}

export function resumeAgent(agentId: string) {
  return api.post<AgentResponse>(`/api/v1/agents/${agentId}/resume`)
}

export function deleteAgent(agentId: string) {
  return api.delete<AgentResponse>(`/api/v1/agents/${agentId}`)
}

export function listStatuses(projectId: string) {
  return api.get<StatusPayload>(`/api/v1/projects/${projectId}/statuses`)
}

export function createStatus(
  projectId: string,
  body: {
    name: string
    color: string
    icon?: string
    position?: number
    is_default?: boolean
    description?: string
  },
) {
  return api.post<StatusResponse>(`/api/v1/projects/${projectId}/statuses`, { body })
}

export function resetStatuses(projectId: string) {
  return api.post<StatusResetPayload>(`/api/v1/projects/${projectId}/statuses/reset`)
}

export function updateStatus(
  statusId: string,
  body: {
    name?: string
    color?: string
    icon?: string
    position?: number
    is_default?: boolean
    description?: string
  },
) {
  return api.patch<StatusResponse>(`/api/v1/statuses/${statusId}`, { body })
}

export function deleteStatus(statusId: string) {
  return api.delete<StatusDeleteResponse>(`/api/v1/statuses/${statusId}`)
}

export function listTickets(
  projectId: string,
  params?: {
    status_name?: string
    priority?: string
  },
) {
  return api.get<TicketPayload>(`/api/v1/projects/${projectId}/tickets`, { params })
}

export function createTicket(
  projectId: string,
  body: {
    title: string
    description?: string
    status_id?: string | null
    priority?: string | null
    type?: string | null
    workflow_id?: string | null
    created_by?: string | null
    parent_ticket_id?: string | null
    external_ref?: string | null
    budget_usd?: number | null
  },
) {
  return api.post<TicketCreateResponse>(`/api/v1/projects/${projectId}/tickets`, { body })
}

export function updateTicket(
  ticketId: string,
  body: {
    budget_usd?: number | null
    created_by?: string | null
    description?: string | null
    external_ref?: string | null
    parent_ticket_id?: string | null
    priority?: string | null
    status_id?: string | null
    title?: string | null
    type?: string | null
    workflow_id?: string | null
  },
) {
  return api.patch(`/api/v1/tickets/${ticketId}`, { body })
}

export function createTicketComment(
  ticketId: string,
  body: {
    body: string
    created_by?: string | null
  },
) {
  return api.post<TicketCommentCreateResponse>(`/api/v1/tickets/${ticketId}/comments`, { body })
}

export function updateTicketComment(
  ticketId: string,
  commentId: string,
  body: {
    body: string
  },
) {
  return api.patch<TicketCommentUpdateResponse>(
    `/api/v1/tickets/${ticketId}/comments/${commentId}`,
    { body },
  )
}

export function deleteTicketComment(ticketId: string, commentId: string) {
  return api.delete<TicketCommentDeleteResponse>(
    `/api/v1/tickets/${ticketId}/comments/${commentId}`,
  )
}

export function addTicketDependency(
  ticketId: string,
  body: {
    target_ticket_id: string
    type: string
  },
) {
  return api.post<TicketDependencyResponse>(`/api/v1/tickets/${ticketId}/dependencies`, { body })
}

export function deleteTicketDependency(ticketId: string, dependencyId: string) {
  return api.delete<TicketDependencyDeleteResponse>(
    `/api/v1/tickets/${ticketId}/dependencies/${dependencyId}`,
  )
}

export function addTicketExternalLink(
  ticketId: string,
  body: {
    type: string
    url: string
    external_id: string
    title?: string | null
    status?: string | null
    relation?: string | null
  },
) {
  return api.post<TicketExternalLinkResponse>(`/api/v1/tickets/${ticketId}/external-links`, {
    body,
  })
}

export function deleteTicketExternalLink(ticketId: string, externalLinkId: string) {
  return api.delete<TicketExternalLinkDeleteResponse>(
    `/api/v1/tickets/${ticketId}/external-links/${externalLinkId}`,
  )
}

export function getTicketDetail(projectId: string, ticketId: string) {
  return api.get<TicketDetailPayload>(`/api/v1/projects/${projectId}/tickets/${ticketId}/detail`)
}

export function listProjectRepos(projectId: string) {
  return api.get<ProjectRepoPayload>(`/api/v1/projects/${projectId}/repos`)
}

export function createProjectRepo(
  projectId: string,
  body: {
    name: string
    repository_url: string
    default_branch: string
    clone_path?: string | null
    is_primary?: boolean
    labels?: string[]
  },
) {
  return api.post<ProjectRepoResponse>(`/api/v1/projects/${projectId}/repos`, { body })
}

export function updateProjectRepo(
  projectId: string,
  repoId: string,
  body: {
    name?: string | null
    repository_url?: string | null
    default_branch?: string | null
    clone_path?: string | null
    is_primary?: boolean | null
    labels?: string[] | null
  },
) {
  return api.patch<ProjectRepoResponse>(`/api/v1/projects/${projectId}/repos/${repoId}`, { body })
}

export function deleteProjectRepo(projectId: string, repoId: string) {
  return api.delete<ProjectRepoResponse>(`/api/v1/projects/${projectId}/repos/${repoId}`)
}

export function listTicketRepoScopes(projectId: string, ticketId: string) {
  return api.get<TicketRepoScopePayload>(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/repo-scopes`,
  )
}

export function createTicketRepoScope(
  projectId: string,
  ticketId: string,
  body: {
    repo_id: string
    branch_name?: string | null
    pull_request_url?: string | null
    pr_status?: string
    ci_status?: string
    is_primary_scope?: boolean
  },
) {
  return api.post<TicketRepoScopeResponse>(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/repo-scopes`,
    { body },
  )
}

export function updateTicketRepoScope(
  projectId: string,
  ticketId: string,
  scopeId: string,
  body: {
    branch_name?: string | null
    pull_request_url?: string | null
    pr_status?: string | null
    ci_status?: string | null
    is_primary_scope?: boolean | null
  },
) {
  return api.patch<TicketRepoScopeResponse>(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/repo-scopes/${scopeId}`,
    { body },
  )
}

export function deleteTicketRepoScope(projectId: string, ticketId: string, scopeId: string) {
  return api.delete<TicketRepoScopeResponse>(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/repo-scopes/${scopeId}`,
  )
}

export function listWorkflows(projectId: string) {
  return api.get<WorkflowListPayload>(`/api/v1/projects/${projectId}/workflows`)
}

export function createWorkflow(
  projectId: string,
  body: {
    agent_id: string
    finish_status_ids: string[]
    harness_content?: string
    harness_path?: string | null
    hooks?: Record<string, unknown>
    is_active?: boolean | null
    max_concurrent?: number | null
    max_retry_attempts?: number | null
    name?: string
    pickup_status_ids: string[]
    stall_timeout_minutes?: number | null
    timeout_minutes?: number | null
    type?: string
  },
) {
  return api.post<{ workflow?: WorkflowDetailPayload['workflow'] }>(
    `/api/v1/projects/${projectId}/workflows`,
    { body },
  )
}

export function getWorkflow(workflowId: string) {
  return api.get<WorkflowDetailPayload>(`/api/v1/workflows/${workflowId}`)
}

export function updateWorkflow(
  workflowId: string,
  body: {
    agent_id?: string | null
    finish_status_ids?: string[]
    harness_path?: string | null
    hooks?: Record<string, unknown> | null
    is_active?: boolean | null
    max_concurrent?: number | null
    max_retry_attempts?: number | null
    name?: string | null
    pickup_status_ids?: string[]
    stall_timeout_minutes?: number | null
    timeout_minutes?: number | null
    type?: string | null
  },
) {
  return api.patch<WorkflowUpdateResponse>(`/api/v1/workflows/${workflowId}`, { body })
}

export function deleteWorkflow(workflowId: string) {
  return api.delete<WorkflowDeleteResponse>(`/api/v1/workflows/${workflowId}`)
}

export function listScheduledJobs(projectId: string) {
  return api.get<ScheduledJobListPayload>(`/api/v1/projects/${projectId}/scheduled-jobs`)
}

export function createScheduledJob(
  projectId: string,
  body: {
    cron_expression: string
    is_enabled?: boolean | null
    name: string
    ticket_template?: {
      budget_usd?: number
      created_by?: string
      description?: string
      priority?: string
      status?: string
      title?: string
      type?: string
    }
    workflow_id: string
  },
) {
  return api.post<ScheduledJobResponse>(`/api/v1/projects/${projectId}/scheduled-jobs`, { body })
}

export function updateScheduledJob(
  jobId: string,
  body: {
    cron_expression?: string | null
    is_enabled?: boolean | null
    name?: string | null
    ticket_template?: {
      budget_usd?: number
      created_by?: string
      description?: string
      priority?: string
      status?: string
      title?: string
      type?: string
    } | null
    workflow_id?: string | null
  },
) {
  return api.patch<ScheduledJobUpdateResponse>(`/api/v1/scheduled-jobs/${jobId}`, { body })
}

export function deleteScheduledJob(jobId: string) {
  return api.delete<ScheduledJobDeleteResponse>(`/api/v1/scheduled-jobs/${jobId}`)
}

export function triggerScheduledJob(jobId: string) {
  return api.post<ScheduledJobTriggerResponse>(`/api/v1/scheduled-jobs/${jobId}/trigger`)
}

export function getWorkflowHarness(workflowId: string) {
  return api.get<HarnessPayload>(`/api/v1/workflows/${workflowId}/harness`)
}

export function saveWorkflowHarness(workflowId: string, content: string) {
  return api.put<HarnessPayload>(`/api/v1/workflows/${workflowId}/harness`, {
    body: { content },
  })
}

export function validateHarness(content: string) {
  return api.post<HarnessValidationResponse>('/api/v1/harness/validate', {
    body: { content },
  })
}

export function listHarnessVariables() {
  return api.get<HarnessVariableDictionaryPayload>('/api/v1/harness/variables')
}

export function listSkills(projectId: string) {
  return api.get<SkillListPayload>(`/api/v1/projects/${projectId}/skills`)
}

export function bindWorkflowSkills(workflowId: string, skills: string[]) {
  return api.post<HarnessPayload>(`/api/v1/workflows/${workflowId}/skills/bind`, {
    body: { skills },
  })
}

export function unbindWorkflowSkills(workflowId: string, skills: string[]) {
  return api.post<HarnessPayload>(`/api/v1/workflows/${workflowId}/skills/unbind`, {
    body: { skills },
  })
}

export function listBuiltinRoles() {
  return api.get<BuiltinRolePayload>('/api/v1/roles/builtin')
}

export function updateProvider(
  providerId: string,
  body: {
    machine_id?: string
    name?: string
    adapter_type?: string
    cli_command?: string
    cli_args?: string[]
    auth_config?: Record<string, unknown>
    model_name?: string
    model_temperature?: number
    model_max_tokens?: number
    cost_per_input_token?: number
    cost_per_output_token?: number
  },
) {
  return api.patch<{ provider?: AgentProvider }>(`/api/v1/providers/${providerId}`, { body })
}

export function listNotificationEventTypes() {
  return api.get<NotificationRuleEventTypesPayload>('/api/v1/notification-event-types')
}

export function listNotificationChannels(orgId: string) {
  return api.get<NotificationChannelPayload>(`/api/v1/orgs/${orgId}/channels`)
}

export function createNotificationChannel(
  orgId: string,
  body: {
    name: string
    type: string
    config?: Record<string, unknown>
    is_enabled?: boolean
  },
) {
  return api.post<NotificationChannelResponse>(`/api/v1/orgs/${orgId}/channels`, { body })
}

export function updateNotificationChannel(
  channelId: string,
  body: {
    name?: string
    type?: string
    config?: Record<string, unknown>
    is_enabled?: boolean
  },
) {
  return api.patch<NotificationChannelResponse>(`/api/v1/channels/${channelId}`, { body })
}

export function deleteNotificationChannel(channelId: string) {
  return api.delete<NotificationChannelDeleteResponse>(`/api/v1/channels/${channelId}`)
}

export function testNotificationChannel(channelId: string) {
  return api.post<NotificationChannelTestResponse>(`/api/v1/channels/${channelId}/test`)
}

export function listNotificationRules(projectId: string) {
  return api.get<NotificationRulePayload>(`/api/v1/projects/${projectId}/notification-rules`)
}

export function createNotificationRule(
  projectId: string,
  body: {
    name: string
    event_type: string
    filter?: Record<string, unknown>
    channel_id: string
    template?: string
    is_enabled?: boolean
  },
) {
  return api.post<NotificationRuleResponse>(`/api/v1/projects/${projectId}/notification-rules`, {
    body,
  })
}

export function updateNotificationRule(
  ruleId: string,
  body: {
    name?: string
    event_type?: string
    filter?: Record<string, unknown>
    channel_id?: string
    template?: string
    is_enabled?: boolean
  },
) {
  return api.patch<NotificationRuleResponse>(`/api/v1/notification-rules/${ruleId}`, { body })
}

export function deleteNotificationRule(ruleId: string) {
  return api.delete<NotificationRuleDeleteResponse>(`/api/v1/notification-rules/${ruleId}`)
}
