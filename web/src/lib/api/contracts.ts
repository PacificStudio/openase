import type { paths } from './generated/openapi'

type JsonBody<T> = T extends { content: { 'application/json': infer Body } } ? Body : never

type SuccessJson<T> =
  | JsonBody<T extends { responses: { 200: infer Response } } ? Response : never>
  | JsonBody<T extends { responses: { 201: infer Response } } ? Response : never>
  | JsonBody<T extends { responses: { 202: infer Response } } ? Response : never>
  | JsonBody<T extends { responses: { 204: infer Response } } ? Response : never>

type OperationFor<Path extends keyof paths, Method extends keyof paths[Path]> = NonNullable<
  paths[Path][Method]
>
type ResponseFor<Path extends keyof paths, Method extends keyof paths[Path]> = SuccessJson<
  OperationFor<Path, Method>
>

type Defined<T> = Exclude<T, undefined>
type ItemOf<T> = T extends readonly (infer Item)[] ? Item : T extends (infer Item)[] ? Item : never

type DeepRequired<T> = T extends readonly (infer Item)[]
  ? DeepRequired<Defined<Item>>[]
  : T extends (infer Item)[]
    ? DeepRequired<Defined<Item>>[]
    : T extends object
      ? { [K in keyof T]-?: DeepRequired<Defined<T[K]>> }
      : Defined<T>
type ShallowRequired<T> = T extends object ? { [K in keyof T]-?: Defined<T[K]> } : Defined<T>

export type SystemDashboardResponse = DeepRequired<ResponseFor<'/api/v1/system/dashboard', 'get'>>
export type SystemMemorySnapshot = SystemDashboardResponse['memory']

export type OrganizationPayload = DeepRequired<ResponseFor<'/api/v1/orgs', 'get'>>
export type OrganizationResponse = DeepRequired<ResponseFor<'/api/v1/orgs', 'post'>>
export type OrganizationUpdateResponse = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}', 'patch'>>
export type OrganizationArchiveResponse = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}', 'delete'>
>
export type Organization = ItemOf<OrganizationPayload['organizations']>
export type WorkspaceSummaryResponse = ResponseFor<'/api/v1/workspace/summary', 'get'>
export type WorkspaceDashboardSummary = NonNullable<WorkspaceSummaryResponse['workspace']>
export type WorkspaceOrganizationSummary = ItemOf<
  NonNullable<WorkspaceSummaryResponse['organizations']>
>
export type OrganizationSummaryResponse = ResponseFor<'/api/v1/orgs/{orgId}/summary', 'get'>
export type OrganizationDashboardSummary = NonNullable<OrganizationSummaryResponse['organization']>
export type OrganizationProjectSummary = ItemOf<
  NonNullable<OrganizationSummaryResponse['projects']>
>
export type OrganizationTokenUsageResponse = ResponseFor<'/api/v1/orgs/{orgId}/token-usage', 'get'>
export type OrganizationTokenUsageDay = ItemOf<NonNullable<OrganizationTokenUsageResponse['days']>>
export type OrganizationTokenUsageSummary = NonNullable<OrganizationTokenUsageResponse['summary']>
export type OrganizationTokenUsagePeakDay = NonNullable<OrganizationTokenUsageSummary['peak_day']>
export type ProjectTokenUsageResponse = ResponseFor<
  '/api/v1/projects/{projectId}/token-usage',
  'get'
>
export type ProjectTokenUsageDay = ItemOf<NonNullable<ProjectTokenUsageResponse['days']>>
export type ProjectTokenUsageSummary = NonNullable<ProjectTokenUsageResponse['summary']>
export type ProjectTokenUsagePeakDay = NonNullable<ProjectTokenUsageSummary['peak_day']>
export type TokenUsageResponse = OrganizationTokenUsageResponse | ProjectTokenUsageResponse
export type TokenUsageDay = OrganizationTokenUsageDay | ProjectTokenUsageDay
export type TokenUsageSummary = OrganizationTokenUsageSummary | ProjectTokenUsageSummary
export type TokenUsagePeakDay = OrganizationTokenUsagePeakDay | ProjectTokenUsagePeakDay

export type ScopedSecretRecord = {
  id: string
  organization_id: string
  project_id?: string | null
  scope: 'organization' | 'project' | string
  name: string
  kind: string
  description: string
  disabled: boolean
  disabled_at?: string | null
  created_at: string
  updated_at: string
  usage_count: number
  usage_scopes?: string[]
  encryption: {
    algorithm: string
    key_id: string
    key_source: string
    rotated_at: string
    value_preview: string
  }
}

export type ScopedSecretsResponse = {
  secrets: ScopedSecretRecord[]
}

export type ScopedSecretResponse = {
  secret: ScopedSecretRecord
}

type RawAgentProviderListPayload = ResponseFor<'/api/v1/orgs/{orgId}/providers', 'get'>
type RawAgentProviderResponse = ResponseFor<'/api/v1/orgs/{orgId}/providers', 'post'>
export type AgentProvider = ShallowRequired<
  ItemOf<Defined<RawAgentProviderListPayload['providers']>>
>
export type AgentProviderListPayload = Omit<
  ShallowRequired<RawAgentProviderListPayload>,
  'providers'
> & {
  providers: AgentProvider[]
}
export type AgentProviderResponse = Omit<ShallowRequired<RawAgentProviderResponse>, 'provider'> & {
  provider: AgentProvider
}

type RawAgentProviderModelCatalogPayload = ResponseFor<'/api/v1/provider-model-options', 'get'>
export type AgentProviderModelOption = ShallowRequired<
  ItemOf<
    Defined<
      ItemOf<Defined<RawAgentProviderModelCatalogPayload['adapter_model_options']>>['options']
    >
  >
>
export type AgentProviderModelCatalogEntry = Omit<
  ShallowRequired<ItemOf<Defined<RawAgentProviderModelCatalogPayload['adapter_model_options']>>>,
  'options'
> & {
  options: AgentProviderModelOption[]
}
export type AgentProviderModelCatalogPayload = Omit<
  ShallowRequired<RawAgentProviderModelCatalogPayload>,
  'adapter_model_options'
> & {
  adapter_model_options: AgentProviderModelCatalogEntry[]
}

type RawProjectPayload = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}/projects', 'get'>>
type RawProjectCreateResponse = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}/projects', 'post'>>
type RawProjectResponse = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}', 'get'>>
export type Project = Omit<
  ItemOf<RawProjectPayload['projects']>,
  | 'agent_run_summary_prompt'
  | 'effective_agent_run_summary_prompt'
  | 'agent_run_summary_prompt_source'
> & {
  agent_run_summary_prompt?: string
  effective_agent_run_summary_prompt?: string
  agent_run_summary_prompt_source?: 'builtin' | 'project_override'
}
export type ProjectPayload = Omit<RawProjectPayload, 'projects'> & {
  projects: Project[]
}
export type ProjectCreateResponse = Omit<RawProjectCreateResponse, 'project'> & {
  project: Project
}
export type ProjectResponse = Omit<RawProjectResponse, 'project'> & {
  project: Project
}
export type ProjectArchiveResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}', 'delete'>
>

export type MachinePayload = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}/machines', 'get'>>
export type MachineCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}/machines', 'post'>
>
export type MachineResponse = DeepRequired<ResponseFor<'/api/v1/machines/{machineId}', 'get'>>
export type Machine = ItemOf<MachinePayload['machines']>
export type MachineTestResponse = DeepRequired<
  ResponseFor<'/api/v1/machines/{machineId}/test', 'post'>
>
export type MachineHealthRefreshResponse = {
  machine: Machine
}
export type MachineProbe = MachineTestResponse['probe']
export type MachineResourcesResponse = DeepRequired<
  ResponseFor<'/api/v1/machines/{machineId}/resources', 'get'>
>

export type AgentPayload = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}/agents', 'get'>>
export type Agent = ItemOf<AgentPayload['agents']>
export type AgentRunPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/agent-runs', 'get'>
>
export type AgentRun = ItemOf<AgentRunPayload['agent_runs']>
export type AgentResponse = DeepRequired<ResponseFor<'/api/v1/agents/{agentId}', 'get'>>
export type AgentOutputPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/agents/{agentId}/output', 'get'>
>
export type AgentOutputEntry = ItemOf<AgentOutputPayload['entries']>
export type AgentStepPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/agents/{agentId}/steps', 'get'>
>
export type AgentStepEntry = ItemOf<AgentStepPayload['entries']>

export type ActivityPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/activity', 'get'>
>
export type ActivityEvent = ItemOf<ActivityPayload['events']>
export type ProjectUpdatePayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates', 'get'>
>
export type ProjectUpdateCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates', 'post'>
>
export type ProjectUpdateThreadRecord = ItemOf<ProjectUpdatePayload['threads']>
export type ProjectUpdateCommentRecord = ItemOf<ProjectUpdateThreadRecord['comments']>
export type ProjectUpdateThreadResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}', 'patch'>
>
export type ProjectUpdateThreadDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}', 'delete'>
>
export type ProjectUpdateThreadRevisionListResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}/revisions', 'get'>
>
export type ProjectUpdateThreadRevisionRecord = ItemOf<
  ProjectUpdateThreadRevisionListResponse['revisions']
>
export type ProjectUpdateCommentCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}/comments', 'post'>
>
export type ProjectUpdateCommentResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}', 'patch'>
>
export type ProjectUpdateCommentDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}', 'delete'>
>
export type ProjectUpdateCommentRevisionListResponse = DeepRequired<
  ResponseFor<
    '/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}/revisions',
    'get'
  >
>
export type ProjectUpdateCommentRevisionRecord = ItemOf<
  ProjectUpdateCommentRevisionListResponse['revisions']
>

export type StatusPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/statuses', 'get'>
>
export type StatusResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/statuses', 'post'>
>
export type StatusResetPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/statuses/reset', 'post'>
>
export type StatusDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/statuses/{statusId}', 'delete'>
>
export type TicketStatus = ItemOf<StatusPayload['statuses']>

export type TicketPayload = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}/tickets', 'get'>>
export type ArchivedTicketPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/archived', 'get'>
>
export type TicketCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets', 'post'>
>
export type TicketResponse = DeepRequired<ResponseFor<'/api/v1/tickets/{ticketId}', 'patch'>>
export type TicketWorkspaceResetResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/workspace/reset', 'post'>
>
export type Ticket = ItemOf<TicketPayload['tickets']>
export type ArchivedTicket = ItemOf<ArchivedTicketPayload['tickets']>
export type TicketPriority = Ticket['priority']
export type TicketReference = ItemOf<Ticket['children']>
export type TicketDependency = ItemOf<Ticket['dependencies']>
export type TicketCommentCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/comments', 'post'>
>
export type TicketCommentUpdateResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/comments/{commentId}', 'patch'>
>
export type TicketCommentDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/comments/{commentId}', 'delete'>
>
export type TicketCommentRevisionListResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/comments/{commentId}/revisions', 'get'>
>
export type TicketCommentRevisionRecord = ItemOf<TicketCommentRevisionListResponse['revisions']>
type RawTicketRunListPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/runs', 'get'>
>
type RawTicketRunDetailPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/runs/{runId}', 'get'>
>
export type TicketRunCompletionSummaryRecord = {
  status: 'pending' | 'completed' | 'failed'
  markdown?: string
  json?: Record<string, unknown>
  generated_at?: string
  error?: string
}
export type TicketRunRecord = Omit<
  ItemOf<RawTicketRunListPayload['runs']>,
  'completion_summary'
> & {
  completion_summary?: TicketRunCompletionSummaryRecord
}
export type TicketRunListPayload = Omit<RawTicketRunListPayload, 'runs'> & {
  runs: TicketRunRecord[]
}
export type TicketRunTranscriptItemRecord = {
  kind: 'step' | 'trace'
  cursor: string
  step_entry?: TicketRunStepRecord
  trace_entry?: TicketRunTraceRecord
}
export type TicketRunTranscriptPageRecord = {
  items: TicketRunTranscriptItemRecord[]
  has_older: boolean
  hidden_older_count: number
  has_newer: boolean
  hidden_newer_count: number
  oldest_cursor?: string
  newest_cursor?: string
}
export type TicketRunDetailPayload = Omit<RawTicketRunDetailPayload, 'run'> & {
  run: TicketRunRecord
  transcript_page?: TicketRunTranscriptPageRecord
}
export type TicketRunTraceRecord = ItemOf<TicketRunDetailPayload['trace_entries']>
export type TicketRunStepRecord = ItemOf<TicketRunDetailPayload['step_entries']>
export type TicketDependencyResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/dependencies', 'post'>
>
export type TicketDependencyDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/dependencies/{dependencyId}', 'delete'>
>
export type TicketExternalLinkResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/external-links', 'post'>
>
export type TicketExternalLinkDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/external-links/{externalLinkId}', 'delete'>
>

export type ProjectRepoPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/repos', 'get'>
>
export type ProjectRepoResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/repos', 'post'>
>
export type ProjectRepoRecord = ItemOf<ProjectRepoPayload['repos']>
export type GitHubRepositoryNamespacesResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/github/namespaces', 'get'>
>
export type GitHubRepositoryListResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/github/repos', 'get'>
>
export type GitHubRepositoryCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/github/repos', 'post'>
>
export type GitHubRepositoryNamespaceRecord = ItemOf<
  GitHubRepositoryNamespacesResponse['namespaces']
>
export type GitHubRepositoryRecord = ItemOf<GitHubRepositoryListResponse['repositories']>

export type TicketRepoScopePayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes', 'get'>
>
export type TicketRepoScopeResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes', 'post'>
>
export type TicketRepoScopeRecord = ItemOf<TicketRepoScopePayload['repo_scopes']>

export type WorkflowListPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/workflows', 'get'>
>
export type WorkflowResponse = DeepRequired<ResponseFor<'/api/v1/workflows/{workflowId}', 'get'>>
export type WorkflowUpdateResponse = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}', 'patch'>
>
export type WorkflowDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}', 'delete'>
>
export type Workflow = ItemOf<WorkflowListPayload['workflows']>
export type WorkflowDetailPayload = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}', 'get'>
>
export type ScheduledJobListPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/scheduled-jobs', 'get'>
>
export type ScheduledJobResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/scheduled-jobs', 'post'>
>
export type ScheduledJobUpdateResponse = DeepRequired<
  ResponseFor<'/api/v1/scheduled-jobs/{jobId}', 'patch'>
>
export type ScheduledJobDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/scheduled-jobs/{jobId}', 'delete'>
>
export type ScheduledJobTriggerResponse = DeepRequired<
  ResponseFor<'/api/v1/scheduled-jobs/{jobId}/trigger', 'post'>
>
export type ScheduledJob = ItemOf<ScheduledJobListPayload['scheduled_jobs']>

export type HarnessPayload = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}/harness', 'get'>
>
export type HarnessDocument = HarnessPayload['harness']
export type WorkflowHistoryPayload = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}/harness/history', 'get'>
>
export type HarnessVariableDictionaryPayload = DeepRequired<
  ResponseFor<'/api/v1/harness/variables', 'get'>
>
export type WorkflowSkillBindingResponse = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}/skills/bind', 'post'>
>

export type HarnessValidationResponse = DeepRequired<
  ResponseFor<'/api/v1/harness/validate', 'post'>
>
export type HarnessValidationIssue = {
  level: string
  message: string
  line?: number
  column?: number
}

export type SkillListPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/skills', 'get'>
>
export type Skill = ItemOf<SkillListPayload['skills']>
export type SkillBinding = ItemOf<Skill['bound_workflows']>
export type SkillDetailResponse = DeepRequired<ResponseFor<'/api/v1/skills/{skillId}', 'get'>>
export type SkillDetail = SkillDetailResponse['skill']
export type SkillHistoryPayload = DeepRequired<
  ResponseFor<'/api/v1/skills/{skillId}/history', 'get'>
>
export type SkillCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/skills', 'post'>
>
export type SkillUpdateResponse = DeepRequired<ResponseFor<'/api/v1/skills/{skillId}', 'put'>>
export type SkillDeleteResponse = DeepRequired<ResponseFor<'/api/v1/skills/{skillId}', 'delete'>>
export type SkillToggleResponse = DeepRequired<
  ResponseFor<'/api/v1/skills/{skillId}/enable', 'post'>
>
export type SkillBindingUpdateResponse = DeepRequired<
  ResponseFor<'/api/v1/skills/{skillId}/bind', 'post'>
>
export type SkillRefreshResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/skills/refresh', 'post'>
>

/** Bundle file returned by the skill bundle file APIs. */
export type SkillFile = {
  path: string
  file_kind: 'entrypoint' | 'metadata' | 'script' | 'reference' | 'asset'
  media_type: string
  encoding: 'utf8' | 'binary' | 'base64'
  is_executable: boolean
  size_bytes: number
  sha256: string
  content?: string
  content_base64?: string
}

export type SkillFilesPayload = DeepRequired<
  ResponseFor<'/api/v1/skills/{skillId}/files', 'get'>
> & {
  files: SkillFile[]
}

export type BuiltinRolePayload = DeepRequired<ResponseFor<'/api/v1/roles/builtin', 'get'>>
export type BuiltinRole = ItemOf<BuiltinRolePayload['roles']>
export type BuiltinRoleDetailResponse = DeepRequired<
  ResponseFor<'/api/v1/roles/builtin/{roleSlug}', 'get'>
>
export type BuiltinRoleDetail = BuiltinRoleDetailResponse['role']

export type AppContextPayload = {
  organizations: Organization[]
  projects: Project[]
  providers: AgentProvider[]
  agent_count: number
}

export type HRAdvisorResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/hr-advisor', 'get'>
>
export type HRAdvisorSummary = HRAdvisorResponse['summary']
export type HRAdvisorStaffing = HRAdvisorResponse['staffing']
export type HRAdvisorRecommendation = ItemOf<HRAdvisorResponse['recommendations']>
export type HRAdvisorActivationResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/hr-advisor/activate', 'post'>
>

export type TicketDetailPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/detail', 'get'>
>
export type TicketCommentRecord = ItemOf<TicketDetailPayload['comments']>
export type TicketTimelineItemRecord = ItemOf<TicketDetailPayload['timeline']>
export type TicketRepoScope = ItemOf<TicketDetailPayload['repo_scopes']>
export type ProjectRepo = NonNullable<TicketRepoScope['repo']>

export type NotificationRuleEventTypesPayload = DeepRequired<
  ResponseFor<'/api/v1/notification-event-types', 'get'>
>
export type NotificationRuleEventType = ItemOf<NotificationRuleEventTypesPayload['event_types']>

export type NotificationChannelPayload = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}/channels', 'get'>
>
export type NotificationChannelResponse = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}/channels', 'post'>
>
export type NotificationChannelDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/channels/{channelId}', 'delete'>
>
export type NotificationChannelTestResponse = DeepRequired<
  ResponseFor<'/api/v1/channels/{channelId}/test', 'post'>
>
export type NotificationChannel = ItemOf<NotificationChannelPayload['channels']>

export type NotificationRulePayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/notification-rules', 'get'>
>
export type NotificationRuleResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/notification-rules', 'post'>
>
export type NotificationRuleDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/notification-rules/{ruleId}', 'delete'>
>
export type NotificationRule = ItemOf<NotificationRulePayload['rules']>

type RawSecuritySettingsResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings', 'get'>
>
export type SecurityAuthBootstrapState = {
  status: string
  admin_emails: string[]
  summary: string
}
export type SecurityOIDCDraft = {
  issuer_url: string
  client_id: string
  client_secret_configured: boolean
  redirect_mode: string
  fixed_redirect_url: string
  scopes: string[]
  allowed_email_domains: string[]
  bootstrap_admin_emails: string[]
}
export type SecurityDocumentationLink = {
  title: string
  href: string
  summary: string
}
export type SecurityAuthSessionPolicy = {
  session_ttl: string
  session_idle_ttl: string
}
export type SecurityAuthValidationDiagnostics = {
  status: string
  message: string
  checked_at?: string | null
  issuer_url?: string
  authorization_endpoint?: string
  token_endpoint?: string
  redirect_url?: string
  warnings: string[]
}
export type SecurityAuthSettings = {
  active_mode: string
  configured_mode: string
  issuer_url?: string
  local_principal: string
  mode_summary: string
  recommended_mode: string
  public_exposure_risk: string
  warnings: string[]
  next_steps: string[]
  config_path?: string
  bootstrap_state: SecurityAuthBootstrapState
  session_policy: SecurityAuthSessionPolicy
  last_validation: SecurityAuthValidationDiagnostics
  oidc_draft: SecurityOIDCDraft
  docs: SecurityDocumentationLink[]
}
export type SecuritySettingsResponse = Omit<RawSecuritySettingsResponse, 'security'> & {
  security: RawSecuritySettingsResponse['security'] & {
    auth: SecurityAuthSettings
    secret_hygiene: NonNullable<RawSecuritySettingsResponse['security']>['secret_hygiene'] & {
      machine_env_vars_redacted: boolean
      runtime_secret_responses_redacted: boolean
      legacy_providers_requiring_migration: number
      legacy_provider_inline_secret_bindings: number
      legacy_machines_requiring_migration: number
      legacy_machine_secret_env_vars: number
      rollout_checklist: Array<{
        key: string
        title: string
        status: string
        summary: string
      }>
    }
  }
}
export type ScopedSecretPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings/secrets', 'get'>
>
export type ScopedSecret = ItemOf<ScopedSecretPayload['secrets']>
export type ScopedSecretBindingPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings/secret-bindings', 'get'>
>
export type ScopedSecretBinding = ItemOf<ScopedSecretBindingPayload['bindings']>
export type CreateScopedSecretBindingResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings/secret-bindings', 'post'>
>
export type DeleteScopedSecretBindingResponse = DeepRequired<
  ResponseFor<
    '/api/v1/projects/{projectId}/security-settings/secret-bindings/{bindingId}',
    'delete'
  >
>
export type OIDCDraftTestResponse = {
  status: string
  message: string
  issuer_url: string
  authorization_endpoint: string
  token_endpoint: string
  redirect_url: string
  warnings: string[]
}
export type OIDCEnableResponse = {
  activation: {
    status: string
    message: string
    restart_required: boolean
    next_steps: string[]
  }
  security: SecuritySettingsResponse['security']
}
export type AdminAuthResponse = {
  auth: SecurityAuthSettings
}
export type AdminAuthModeTransitionResponse = {
  transition: {
    status: string
    message: string
    restart_required: boolean
    next_steps: string[]
  }
  auth: SecurityAuthSettings
}
export type SaveGitHubOutboundCredentialResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings/github-outbound-credential', 'put'>
>
export type ImportGitHubOutboundCredentialResponse = DeepRequired<
  ResponseFor<
    '/api/v1/projects/{projectId}/security-settings/github-outbound-credential/import-gh-cli',
    'post'
  >
>
export type RetestGitHubOutboundCredentialResponse = DeepRequired<
  ResponseFor<
    '/api/v1/projects/{projectId}/security-settings/github-outbound-credential/retest',
    'post'
  >
>
export type DeleteGitHubOutboundCredentialResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/security-settings/github-outbound-credential', 'delete'>
>

// Org-level GitHub credential — managed under /orgs/:orgId/security/github-credential
export type GitHubCredentialSlot = {
  configured: boolean
  scope?: string
  source?: string
  token_preview?: string
  probe: {
    state: string
    configured: boolean
    valid: boolean
    login?: string
    permissions: string[]
    repo_access: string
    checked_at?: string
    last_error?: string
  }
}
export type OrgGitHubCredentialResponse = {
  credential: GitHubCredentialSlot
}
