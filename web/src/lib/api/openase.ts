import { api } from './client'
import type { SessionGovernanceResponse } from './auth'
import type {
  ActivityPayload,
  AdminAuthModeTransitionResponse,
  AdminAuthResponse,
  AgentPayload,
  AgentRunPayload,
  AgentProviderResponse,
  AgentOutputPayload,
  AgentStepPayload,
  AgentResponse,
  AgentProvider,
  AgentProviderListPayload,
  AgentProviderModelCatalogPayload,
  ArchivedTicketPayload,
  BuiltinRolePayload,
  BuiltinRoleDetailResponse,
  CreateScopedSecretBindingResponse,
  DeleteScopedSecretBindingResponse,
  DeleteGitHubOutboundCredentialResponse,
  OrgGitHubCredentialResponse,
  GitHubRepositoryCreateResponse,
  GitHubRepositoryListResponse,
  GitHubRepositoryNamespacesResponse,
  HarnessPayload,
  WorkflowHistoryPayload,
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
  OIDCDraftTestResponse,
  OIDCEnableResponse,
  ProjectRepoPayload,
  ProjectArchiveResponse,
  ProjectUpdateCommentCreateResponse,
  ProjectUpdateCommentDeleteResponse,
  ProjectUpdateCommentResponse,
  ProjectUpdateCreateResponse,
  ProjectUpdatePayload,
  ProjectUpdateThreadDeleteResponse,
  ProjectUpdateThreadResponse,
  ProjectRepoResponse,
  ProjectCreateResponse,
  ProjectPayload,
  ProjectResponse,
  ScheduledJobDeleteResponse,
  ScheduledJobListPayload,
  ScheduledJobResponse,
  ScheduledJobTriggerResponse,
  ScheduledJobUpdateResponse,
  ScopedSecretBindingPayload,
  ScopedSecretPayload,
  SecuritySettingsResponse,
  ScopedSecretResponse,
  ScopedSecretsResponse,
  RetestGitHubOutboundCredentialResponse,
  SaveGitHubOutboundCredentialResponse,
  SkillListPayload,
  SkillCreateResponse,
  SkillDeleteResponse,
  SkillDetailResponse,
  SkillFilesPayload,
  SkillHistoryPayload,
  SkillRefreshResponse,
  SkillBindingUpdateResponse,
  SkillToggleResponse,
  SkillUpdateResponse,
  StatusDeleteResponse,
  StatusPayload,
  StatusResetPayload,
  StatusResponse,
  SystemDashboardResponse,
  TicketDetailPayload,
  TicketRunDetailPayload,
  TicketRunListPayload,
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
  TicketResponse,
  TicketWorkspaceResetResponse,
  HRAdvisorActivationResponse,
  HRAdvisorResponse,
  Organization,
  OrganizationSummaryResponse,
  OrganizationTokenUsageResponse,
  ProjectTokenUsageResponse,
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
  advertised_endpoint?: string
  agent_cli_path?: string
  description?: string
  execution_mode?: string
  env_vars?: string[]
  host?: string
  labels?: string[]
  name?: string
  port?: number
  reachability_mode?: string
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

export function getWorkspaceSummary(opts?: { signal?: AbortSignal }) {
  return api.get<WorkspaceSummaryResponse>('/api/v1/workspace/summary', opts)
}

export function getOrganizationSummary(orgId: string, opts?: { signal?: AbortSignal }) {
  return api.get<OrganizationSummaryResponse>(`/api/v1/orgs/${orgId}/summary`, opts)
}

export function getOrganizationTokenUsage(
  orgId: string,
  query: {
    from: string
    to: string
  },
  opts?: { signal?: AbortSignal },
) {
  const params = new URLSearchParams({
    from: query.from,
    to: query.to,
  })
  return api.get<OrganizationTokenUsageResponse>(
    `/api/v1/orgs/${orgId}/token-usage?${params.toString()}`,
    opts,
  )
}

export function getProjectTokenUsage(
  projectId: string,
  query: {
    from: string
    to: string
  },
  opts?: { signal?: AbortSignal },
) {
  const params = new URLSearchParams({
    from: query.from,
    to: query.to,
  })
  return api.get<ProjectTokenUsageResponse>(
    `/api/v1/projects/${projectId}/token-usage?${params.toString()}`,
    opts,
  )
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
    permission_profile?: string
    cli_command?: string
    cli_args?: string[]
    auth_config?: Record<string, unknown>
    secret_bindings?: Array<{ env_var_key: string; binding_key: string }>
    model_name: string
    model_temperature?: number
    model_max_tokens?: number
    max_parallel_runs?: number
    cost_per_input_token?: number
    cost_per_output_token?: number
    pricing_config?: Record<string, unknown>
  },
) {
  return api.post<AgentProviderResponse>(`/api/v1/orgs/${orgId}/providers`, { body })
}

export function getProject(projectId: string) {
  return api.get<ProjectResponse>(`/api/v1/projects/${projectId}`)
}

export function getAdminAuth() {
  return api.get<AdminAuthResponse>('/api/v1/admin/auth')
}

export function saveAdminOIDCDraft(body: {
  issuer_url: string
  client_id: string
  client_secret?: string
  redirect_mode: string
  fixed_redirect_url: string
  scopes: string[]
  allowed_email_domains: string[]
  bootstrap_admin_emails: string[]
}) {
  return api.put<AdminAuthResponse>('/api/v1/admin/auth/oidc-draft', { body })
}

export function testAdminOIDCDraft(body: {
  issuer_url: string
  client_id: string
  client_secret?: string
  redirect_mode: string
  fixed_redirect_url: string
  scopes: string[]
  allowed_email_domains: string[]
  bootstrap_admin_emails: string[]
}) {
  return api.post<OIDCDraftTestResponse>('/api/v1/admin/auth/oidc-draft/test', { body })
}

export function enableAdminOIDC(body: {
  issuer_url: string
  client_id: string
  client_secret?: string
  redirect_mode: string
  fixed_redirect_url: string
  scopes: string[]
  allowed_email_domains: string[]
  bootstrap_admin_emails: string[]
}) {
  return api.post<AdminAuthModeTransitionResponse>('/api/v1/admin/auth/oidc-enable', { body })
}

export function disableAdminAuth() {
  return api.post<AdminAuthModeTransitionResponse>('/api/v1/admin/auth/disable')
}

export function getSecuritySettings(projectId: string) {
  return api.get<SecuritySettingsResponse>(`/api/v1/projects/${projectId}/security-settings`)
}

export function listScopedSecrets(projectId: string) {
  return api.get<ScopedSecretPayload>(`/api/v1/projects/${projectId}/security-settings/secrets`)
}

export function listScopedSecretBindings(projectId: string) {
  return api.get<ScopedSecretBindingPayload>(
    `/api/v1/projects/${projectId}/security-settings/secret-bindings`,
  )
}

export function createScopedSecretBinding(
  projectId: string,
  body: {
    secret_id: string
    scope: 'workflow' | 'ticket'
    scope_resource_id: string
    binding_key: string
  },
) {
  return api.post<CreateScopedSecretBindingResponse>(
    `/api/v1/projects/${projectId}/security-settings/secret-bindings`,
    { body },
  )
}

export function listProjectScopedSecrets(projectId: string) {
  return api.get<ScopedSecretsResponse>(`/api/v1/projects/${projectId}/security-settings/secrets`)
}

export function createProjectScopedSecret(
  projectId: string,
  body: {
    scope: 'organization' | 'project'
    name: string
    kind?: string
    description?: string
    value: string
  },
) {
  return api.post<ScopedSecretResponse>(`/api/v1/projects/${projectId}/security-settings/secrets`, {
    body,
  })
}

export function rotateProjectScopedSecret(
  projectId: string,
  secretId: string,
  body: { value: string },
) {
  return api.post<ScopedSecretResponse>(
    `/api/v1/projects/${projectId}/security-settings/secrets/${secretId}/rotate`,
    { body },
  )
}

export function deleteScopedSecretBinding(projectId: string, bindingId: string) {
  return api.delete<DeleteScopedSecretBindingResponse>(
    `/api/v1/projects/${projectId}/security-settings/secret-bindings/${bindingId}`,
  )
}

export function disableProjectScopedSecret(projectId: string, secretId: string) {
  return api.post<ScopedSecretResponse>(
    `/api/v1/projects/${projectId}/security-settings/secrets/${secretId}/disable`,
  )
}

export function deleteProjectScopedSecret(projectId: string, secretId: string) {
  return api.delete<void>(`/api/v1/projects/${projectId}/security-settings/secrets/${secretId}`)
}

export function listOrganizationScopedSecrets(orgId: string) {
  return api.get<ScopedSecretsResponse>(`/api/v1/orgs/${orgId}/security-settings/secrets`)
}

export function createOrganizationScopedSecret(
  orgId: string,
  body: {
    name: string
    kind?: string
    description?: string
    value: string
  },
) {
  return api.post<ScopedSecretResponse>(`/api/v1/orgs/${orgId}/security-settings/secrets`, { body })
}

export function rotateOrganizationScopedSecret(
  orgId: string,
  secretId: string,
  body: { value: string },
) {
  return api.post<ScopedSecretResponse>(
    `/api/v1/orgs/${orgId}/security-settings/secrets/${secretId}/rotate`,
    { body },
  )
}

export function disableOrganizationScopedSecret(orgId: string, secretId: string) {
  return api.post<ScopedSecretResponse>(
    `/api/v1/orgs/${orgId}/security-settings/secrets/${secretId}/disable`,
  )
}

export function deleteOrganizationScopedSecret(orgId: string, secretId: string) {
  return api.delete<void>(`/api/v1/orgs/${orgId}/security-settings/secrets/${secretId}`)
}

export function saveOIDCDraft(
  projectId: string,
  body: {
    issuer_url: string
    client_id: string
    client_secret?: string
    redirect_mode: string
    fixed_redirect_url: string
    scopes: string[]
    allowed_email_domains: string[]
    bootstrap_admin_emails: string[]
  },
) {
  return api.put<SecuritySettingsResponse>(
    `/api/v1/projects/${projectId}/security-settings/oidc-draft`,
    {
      body,
    },
  )
}

export function testOIDCDraft(
  projectId: string,
  body: {
    issuer_url: string
    client_id: string
    client_secret?: string
    redirect_mode: string
    fixed_redirect_url: string
    scopes: string[]
    allowed_email_domains: string[]
    bootstrap_admin_emails: string[]
  },
) {
  return api.post<OIDCDraftTestResponse>(
    `/api/v1/projects/${projectId}/security-settings/oidc-draft/test`,
    { body },
  )
}

export function enableOIDC(
  projectId: string,
  body: {
    issuer_url: string
    client_id: string
    client_secret?: string
    redirect_mode: string
    fixed_redirect_url: string
    scopes: string[]
    allowed_email_domains: string[]
    bootstrap_admin_emails: string[]
  },
) {
  return api.post<OIDCEnableResponse>(
    `/api/v1/projects/${projectId}/security-settings/oidc-enable`,
    {
      body,
    },
  )
}

export function getSessionGovernance() {
  return api.get<SessionGovernanceResponse>('/api/v1/auth/sessions')
}

export function revokeAuthSession(id: string) {
  return api.delete<void>(`/api/v1/auth/sessions/${id}`)
}

export function revokeAllOtherAuthSessions() {
  return api.post<{ revoked_count: number }>('/api/v1/auth/sessions/revoke-all')
}

export function adminRevokeUserAuthSessions(userId: string) {
  return api.post<{ revoked_count: number; user_id: string }>(
    `/api/v1/auth/users/${userId}/sessions/revoke`,
  )
}

export type ScopeGroupsResponse = {
  security: {
    agent_tokens: {
      supported_scope_groups: Array<{ category: string; scopes: string[] }>
    }
  }
}

export async function getScopeGroups(
  projectId: string,
): Promise<Array<{ category: string; scopes: string[] }>> {
  const response = await api.get<ScopeGroupsResponse>(
    `/api/v1/projects/${projectId}/security-settings`,
  )
  return response.security?.agent_tokens?.supported_scope_groups ?? []
}

// Project-scoped credential — only manages the project override
export function saveGitHubOutboundCredential(projectId: string, body: { token: string }) {
  return api.put<SaveGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential`,
    { body },
  )
}

export function importGitHubOutboundCredentialFromGHCLI(projectId: string) {
  return api.post<ImportGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential/import-gh-cli`,
    {},
  )
}

export function retestGitHubOutboundCredential(projectId: string) {
  return api.post<RetestGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential/retest`,
    {},
  )
}

export function deleteGitHubOutboundCredential(projectId: string) {
  return api.delete<DeleteGitHubOutboundCredentialResponse>(
    `/api/v1/projects/${projectId}/security-settings/github-outbound-credential`,
  )
}

// Org-scoped credential — manages the org default that all projects fall back to
export function getOrgGitHubCredential(orgId: string) {
  return api.get<OrgGitHubCredentialResponse>(`/api/v1/orgs/${orgId}/security/github-credential`)
}

export function saveOrgGitHubCredential(orgId: string, body: { token: string }) {
  return api.put<OrgGitHubCredentialResponse>(`/api/v1/orgs/${orgId}/security/github-credential`, {
    body,
  })
}

export function importOrgGitHubCredentialFromGHCLI(orgId: string) {
  return api.post<OrgGitHubCredentialResponse>(
    `/api/v1/orgs/${orgId}/security/github-credential/import-gh-cli`,
    {},
  )
}

export function retestOrgGitHubCredential(orgId: string) {
  return api.post<OrgGitHubCredentialResponse>(
    `/api/v1/orgs/${orgId}/security/github-credential/retest`,
    {},
  )
}

export function deleteOrgGitHubCredential(orgId: string) {
  return api.delete<OrgGitHubCredentialResponse>(`/api/v1/orgs/${orgId}/security/github-credential`)
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
    agent_run_summary_prompt?: string | null
    default_agent_provider_id?: string | null
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

export function listProjectUpdates(
  projectId: string,
  params?: {
    limit?: number
    before?: string
  },
) {
  return api.get<ProjectUpdatePayload>(`/api/v1/projects/${projectId}/updates`, { params })
}

export function createProjectUpdateThread(
  projectId: string,
  body: {
    status: 'on_track' | 'at_risk' | 'off_track'
    body: string
    title?: string
    created_by?: string
  },
) {
  return api.post<ProjectUpdateCreateResponse>(`/api/v1/projects/${projectId}/updates`, { body })
}

export function updateProjectUpdateThread(
  projectId: string,
  threadId: string,
  body: {
    status: 'on_track' | 'at_risk' | 'off_track'
    body: string
    title?: string
    edited_by?: string
    edit_reason?: string
  },
) {
  return api.patch<ProjectUpdateThreadResponse>(
    `/api/v1/projects/${projectId}/updates/${threadId}`,
    {
      body,
    },
  )
}

export function deleteProjectUpdateThread(projectId: string, threadId: string) {
  return api.delete<ProjectUpdateThreadDeleteResponse>(
    `/api/v1/projects/${projectId}/updates/${threadId}`,
  )
}

export function createProjectUpdateComment(
  projectId: string,
  threadId: string,
  body: {
    body: string
    created_by?: string
  },
) {
  return api.post<ProjectUpdateCommentCreateResponse>(
    `/api/v1/projects/${projectId}/updates/${threadId}/comments`,
    { body },
  )
}

export function updateProjectUpdateComment(
  projectId: string,
  threadId: string,
  commentId: string,
  body: {
    body: string
    edited_by?: string
    edit_reason?: string
  },
) {
  return api.patch<ProjectUpdateCommentResponse>(
    `/api/v1/projects/${projectId}/updates/${threadId}/comments/${commentId}`,
    { body },
  )
}

export function deleteProjectUpdateComment(projectId: string, threadId: string, commentId: string) {
  return api.delete<ProjectUpdateCommentDeleteResponse>(
    `/api/v1/projects/${projectId}/updates/${threadId}/comments/${commentId}`,
  )
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

export function interruptAgent(agentId: string) {
  return api.post<AgentResponse>(`/api/v1/agents/${agentId}/interrupt`)
}

export function resumeAgent(agentId: string) {
  return api.post<AgentResponse>(`/api/v1/agents/${agentId}/resume`)
}

export function retireAgent(agentId: string) {
  return api.post<AgentResponse>(`/api/v1/agents/${agentId}/retire`)
}

export function deleteAgent(agentId: string) {
  return api.delete<AgentResponse>(`/api/v1/agents/${agentId}`)
}

export function updateAgent(agentId: string, body: { name?: string; provider_id?: string }) {
  return api.patch<AgentResponse>(`/api/v1/agents/${agentId}`, { body })
}

export function listStatuses(projectId: string) {
  return api.get<StatusPayload>(`/api/v1/projects/${projectId}/statuses`)
}

export function createStatus(
  projectId: string,
  body: {
    name: string
    stage?: string
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
    stage?: string
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

export function listArchivedTickets(
  projectId: string,
  params?: {
    page?: number
    per_page?: number
  },
) {
  return api.get<ArchivedTicketPayload>(`/api/v1/projects/${projectId}/tickets/archived`, {
    params,
  })
}

export function createTicket(
  projectId: string,
  body: {
    archived?: boolean | null
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
    archived?: boolean | null
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
  return api.patch<TicketResponse>(`/api/v1/tickets/${ticketId}`, { body })
}

export function resumeTicketRetry(ticketId: string) {
  return api.post<TicketResponse>(`/api/v1/tickets/${ticketId}/retry/resume`)
}

export function resetTicketWorkspace(ticketId: string) {
  return api.post<TicketWorkspaceResetResponse>(`/api/v1/tickets/${ticketId}/workspace/reset`)
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

export function listTicketRuns(projectId: string, ticketId: string) {
  return api.get<TicketRunListPayload>(`/api/v1/projects/${projectId}/tickets/${ticketId}/runs`)
}

export function getTicketRun(
  projectId: string,
  ticketId: string,
  runId: string,
  query: {
    limit?: number
    before?: string
    after?: string
  } = {},
) {
  return api.get<TicketRunDetailPayload>(
    `/api/v1/projects/${projectId}/tickets/${ticketId}/runs/${runId}`,
    {
      params: {
        limit: query.limit,
        before: query.before,
        after: query.after,
      },
    },
  )
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

export function listGitHubNamespaces(projectId: string) {
  return api.get<GitHubRepositoryNamespacesResponse>(
    `/api/v1/projects/${projectId}/github/namespaces`,
  )
}

export function listGitHubRepositories(
  projectId: string,
  opts?: {
    query?: string
    cursor?: string
  },
) {
  return api.get<GitHubRepositoryListResponse>(`/api/v1/projects/${projectId}/github/repos`, {
    params: {
      query: opts?.query?.trim() || undefined,
      cursor: opts?.cursor?.trim() || undefined,
    },
  })
}

export function createGitHubRepository(
  projectId: string,
  body: {
    owner: string
    name: string
    description?: string
    visibility: 'private' | 'public'
    auto_init?: boolean
  },
) {
  return api.post<GitHubRepositoryCreateResponse>(`/api/v1/projects/${projectId}/github/repos`, {
    body,
  })
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
    platform_access_allowed?: string[]
    pickup_status_ids: string[]
    role_description?: string
    role_name?: string
    role_slug?: string
    skill_names?: string[]
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

export function getWorkflowImpact(workflowId: string) {
  return api.get<{ impact: import('$lib/features/workflows/types').WorkflowImpact }>(
    `/api/v1/workflows/${workflowId}/impact`,
  )
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
    platform_access_allowed?: string[]
    pickup_status_ids?: string[]
    role_description?: string | null
    role_name?: string | null
    role_slug?: string | null
    stall_timeout_minutes?: number | null
    timeout_minutes?: number | null
    type?: string | null
  },
) {
  return api.patch<WorkflowUpdateResponse>(`/api/v1/workflows/${workflowId}`, { body })
}

export function retireWorkflow(workflowId: string, body: { edited_by?: string | null } = {}) {
  return api.post<WorkflowUpdateResponse>(`/api/v1/workflows/${workflowId}/retire`, { body })
}

export function replaceWorkflowReferences(
  workflowId: string,
  body: {
    replacement_workflow_id: string
    edited_by?: string | null
  },
) {
  return api.post<{
    result: import('$lib/features/workflows/types').WorkflowReplaceReferencesResult
  }>(`/api/v1/workflows/${workflowId}/replace-references`, { body })
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

export function listWorkflowHarnessHistory(workflowId: string) {
  return api.get<WorkflowHistoryPayload>(`/api/v1/workflows/${workflowId}/harness/history`)
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

export function listSkillHistory(skillId: string) {
  return api.get<SkillHistoryPayload>(`/api/v1/skills/${skillId}/history`)
}

export function getSkillFiles(skillId: string) {
  return api.get<SkillFilesPayload>(`/api/v1/skills/${skillId}/files`)
}

export function updateSkill(
  skillId: string,
  body: {
    content?: string
    description?: string
    files?: Array<{
      path: string
      content_base64: string
      media_type?: string
      is_executable?: boolean
    }>
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

export function getBuiltinRole(roleSlug: string) {
  return api.get<BuiltinRoleDetailResponse>(`/api/v1/roles/builtin/${roleSlug}`)
}

export function updateProvider(
  providerId: string,
  body: {
    machine_id?: string
    name?: string
    adapter_type?: string
    permission_profile?: string
    cli_command?: string
    cli_args?: string[]
    auth_config?: Record<string, unknown>
    secret_bindings?: Array<{ env_var_key: string; binding_key: string }>
    model_name?: string
    model_temperature?: number
    model_max_tokens?: number
    max_parallel_runs?: number
    cost_per_input_token?: number
    cost_per_output_token?: number
    pricing_config?: Record<string, unknown>
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
