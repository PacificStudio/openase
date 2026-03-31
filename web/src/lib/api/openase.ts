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
  AgentProviderModelCatalogPayload,
  BuiltinRolePayload,
  DeleteGitHubOutboundCredentialResponse,
  HarnessPayload,
  HarnessVariableDictionaryPayload,
  HarnessValidationResponse,
  ImportGitHubOutboundCredentialResponse,
  MachineCreateResponse,
  MachinePayload,
  MachineHealthRefreshResponse,
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
  RetestGitHubOutboundCredentialResponse,
  SaveGitHubOutboundCredentialResponse,
  SkillListPayload,
  SkillCreateResponse,
  SkillDeleteResponse,
  SkillDetailResponse,
  SkillRefreshResponse,
  SkillHarvestResponse,
  SkillBindingUpdateResponse,
  SkillToggleResponse,
  SkillUpdateResponse,
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
  TicketCommentRevisionListResponse,
  TicketCommentUpdateResponse,
  TicketExternalLinkDeleteResponse,
  TicketExternalLinkResponse,
  TicketPayload,
  HRAdvisorActivationResponse,
  HRAdvisorResponse,
  Organization,
  OrganizationSummaryResponse,
  OrganizationArchiveResponse,
  OrganizationResponse,
  OrganizationUpdateResponse,
  TicketRepoScopePayload,
  TicketRepoScopeResponse,
  WorkflowDetailPayload,
  WorkflowDeleteResponse,
  WorkflowListPayload,
  WorkflowUpdateResponse,
  WorkspaceSummaryResponse,
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
  mirror_root?: string
}

export type IssueConnectorRecord = {
  id: string
  project_id: string
  type: string
  name: string
  status: string
  config: {
    type: string
    base_url: string
    project_ref: string
    poll_interval: string
    sync_direction: string
    filters: {
      labels: string[]
      exclude_labels: string[]
      states: string[]
      authors: string[]
    }
    status_mapping: Record<string, string>
    auto_workflow: string
    auth_token_configured: boolean
    webhook_secret_configured: boolean
  }
  last_sync_at?: string | null
  last_error: string
  stats: {
    total_synced: number
    synced24h: number
    failed_count: number
  }
}

export type IssueConnectorListPayload = {
  connectors: IssueConnectorRecord[]
}

export type IssueConnectorResponse = {
  connector: IssueConnectorRecord
}

export type IssueConnectorDeleteResponse = {
  deleted_connector_id: string
}

export type IssueConnectorTestResponse = {
  result: {
    healthy: boolean
    checked_at: string
    message: string
  }
}

export type IssueConnectorSyncResponse = {
  connector: IssueConnectorRecord
  report: {
    connectors_scanned: number
    connectors_synced: number
    connectors_failed: number
    issues_synced: number
  }
}

export type IssueConnectorStatsResponse = {
  stats: {
    connector_id: string
    status: string
    last_sync_at?: string | null
    last_error: string
    stats: {
      total_synced: number
      synced24h: number
      failed_count: number
    }
  }
}

export function getSystemDashboard() {
  return api.get<SystemDashboardResponse>('/api/v1/system/dashboard')
}

export function listOrganizations() {
  return api.get<{ organizations?: Organization[] }>('/api/v1/orgs')
}

export function getWorkspaceSummary(opts?: { signal?: AbortSignal }) {
  return api.get<WorkspaceSummaryResponse>('/api/v1/workspace/summary', opts)
}

export function getOrganizationSummary(orgId: string, opts?: { signal?: AbortSignal }) {
  return api.get<OrganizationSummaryResponse>(`/api/v1/orgs/${orgId}/summary`, opts)
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

export function refreshMachineHealth(machineId: string) {
  return api.post<MachineHealthRefreshResponse>(`/api/v1/machines/${machineId}/refresh-health`)
}

export function getMachineResources(machineId: string) {
  return api.get<MachineResourcesResponse>(`/api/v1/machines/${machineId}/resources`)
}

export function listProviders(orgId: string) {
  return api.get<AgentProviderListPayload>(`/api/v1/orgs/${orgId}/providers`)
}

export function listProviderModelOptions() {
  return api.get<AgentProviderModelCatalogPayload>('/api/v1/provider-model-options')
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
    max_parallel_runs?: number
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

export function saveGitHubOutboundCredential(
  projectId: string,
  body: {
    scope: 'organization' | 'project'
    token: string
  },
) {
  return api.put<SaveGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential`,
    { body },
  )
}

export function importGitHubOutboundCredentialFromGHCLI(
  projectId: string,
  body: {
    scope: 'organization' | 'project'
  },
) {
  return api.post<ImportGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential/import-gh-cli`,
    { body },
  )
}

export function retestGitHubOutboundCredential(
  projectId: string,
  body: {
    scope: 'organization' | 'project'
  },
) {
  return api.post<RetestGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential/retest`,
    { body },
  )
}

export function deleteGitHubOutboundCredential(
  projectId: string,
  scope: 'organization' | 'project',
) {
  const params = new URLSearchParams({ scope })
  return api.delete<DeleteGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential?${params.toString()}`,
  )
}

export function listIssueConnectors(projectId: string) {
  return api.get<IssueConnectorListPayload>(`/api/v1/projects/${projectId}/connectors`)
}

export function createIssueConnector(
  projectId: string,
  body: {
    type: string
    name: string
    status?: string
    config: {
      type: string
      base_url?: string
      auth_token?: string
      project_ref?: string
      poll_interval?: string
      sync_direction?: string
      filters?: {
        labels?: string[]
        exclude_labels?: string[]
        states?: string[]
        authors?: string[]
      }
      status_mapping?: Record<string, string>
      webhook_secret?: string
      auto_workflow?: string
    }
  },
) {
  return api.post<IssueConnectorResponse>(`/api/v1/projects/${projectId}/connectors`, { body })
}

export function updateIssueConnector(
  connectorId: string,
  body: {
    name?: string
    status?: string
    config?: {
      base_url?: string
      auth_token?: string
      project_ref?: string
      poll_interval?: string
      sync_direction?: string
      filters?: {
        labels?: string[]
        exclude_labels?: string[]
        states?: string[]
        authors?: string[]
      }
      status_mapping?: Record<string, string>
      webhook_secret?: string
      auto_workflow?: string
    }
  },
) {
  return api.patch<IssueConnectorResponse>(`/api/v1/connectors/${connectorId}`, { body })
}

export function deleteIssueConnector(connectorId: string) {
  return api.delete<IssueConnectorDeleteResponse>(`/api/v1/connectors/${connectorId}`)
}

export function testIssueConnector(connectorId: string) {
  return api.post<IssueConnectorTestResponse>(`/api/v1/connectors/${connectorId}/test`)
}

export function syncIssueConnector(connectorId: string) {
  return api.post<IssueConnectorSyncResponse>(`/api/v1/connectors/${connectorId}/sync`)
}

export function getIssueConnectorStats(connectorId: string) {
  return api.get<IssueConnectorStatsResponse>(`/api/v1/connectors/${connectorId}/stats`)
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

// TODO(backend): replace with real PATCH /api/v1/agents/{agentId} once implemented
export function updateAgent(
  agentId: string,
  body: { name?: string; provider_id?: string },
): Promise<AgentResponse> {
  void agentId
  void body
  return Promise.resolve({ agent: {} } as AgentResponse)
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
    max_active_runs?: number | null
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
    max_active_runs?: number | null
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
    repo_scopes?: Array<{
      repo_id: string
      branch_name?: string | null
    }>
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
    edited_by?: string | null
    edit_reason?: string | null
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

export function listTicketCommentRevisions(ticketId: string, commentId: string) {
  return api.get<TicketCommentRevisionListResponse>(
    `/api/v1/tickets/${ticketId}/comments/${commentId}/revisions`,
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
    workspace_dirname?: string | null
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
    workspace_dirname?: string | null
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

export function getWorkflowRepositoryPrerequisite(projectId: string) {
  return api.get<{
    prerequisite: {
      kind: string
      repo_count: number
      action: string
    }
  }>(`/api/v1/projects/${projectId}/workflows/prerequisite`)
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

export function createSkill(
  projectId: string,
  body: {
    name: string
    content: string
    description?: string
    created_by?: string
    is_enabled?: boolean
  },
) {
  return api.post<SkillCreateResponse>(`/api/v1/projects/${projectId}/skills`, { body })
}

export function getSkill(skillId: string) {
  return api.get<SkillDetailResponse>(`/api/v1/skills/${skillId}`)
}

export function updateSkill(
  skillId: string,
  body: {
    content: string
    description?: string
  },
) {
  return api.put<SkillUpdateResponse>(`/api/v1/skills/${skillId}`, { body })
}

export function deleteSkill(skillId: string) {
  return api.delete<SkillDeleteResponse>(`/api/v1/skills/${skillId}`)
}

export function enableSkill(skillId: string) {
  return api.post<SkillToggleResponse>(`/api/v1/skills/${skillId}/enable`)
}

export function disableSkill(skillId: string) {
  return api.post<SkillToggleResponse>(`/api/v1/skills/${skillId}/disable`)
}

export function bindSkill(skillId: string, workflowIds: string[]) {
  return api.post<SkillBindingUpdateResponse>(`/api/v1/skills/${skillId}/bind`, {
    body: { workflow_ids: workflowIds },
  })
}

export function unbindSkill(skillId: string, workflowIds: string[]) {
  return api.post<SkillBindingUpdateResponse>(`/api/v1/skills/${skillId}/unbind`, {
    body: { workflow_ids: workflowIds },
  })
}

export function refreshSkills(
  projectId: string,
  body: {
    workspace_root: string
    adapter_type: string
    workflow_id?: string
  },
) {
  return api.post<SkillRefreshResponse>(`/api/v1/projects/${projectId}/skills/refresh`, { body })
}

export function harvestSkills(
  projectId: string,
  body: {
    workspace_root: string
    adapter_type: string
    workflow_id?: string
  },
) {
  return api.post<SkillHarvestResponse>(`/api/v1/projects/${projectId}/skills/harvest`, { body })
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
    max_parallel_runs?: number
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
