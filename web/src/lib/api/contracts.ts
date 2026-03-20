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

export type SystemDashboardResponse = DeepRequired<ResponseFor<'/api/v1/system/dashboard', 'get'>>
export type SystemMemorySnapshot = SystemDashboardResponse['memory']

export type OrganizationPayload = DeepRequired<ResponseFor<'/api/v1/orgs', 'get'>>
export type OrganizationResponse = DeepRequired<ResponseFor<'/api/v1/orgs', 'post'>>
export type Organization = ItemOf<OrganizationPayload['organizations']>

export type AgentProviderListPayload = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}/providers', 'get'>
>
export type AgentProvider = ItemOf<AgentProviderListPayload['providers']>

export type ProjectPayload = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}/projects', 'get'>>
export type ProjectResponse = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}', 'get'>>
export type Project = ItemOf<ProjectPayload['projects']>

export type MachinePayload = DeepRequired<ResponseFor<'/api/v1/orgs/{orgId}/machines', 'get'>>
export type MachineCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/orgs/{orgId}/machines', 'post'>
>
export type MachineResponse = DeepRequired<ResponseFor<'/api/v1/machines/{machineId}', 'get'>>
export type Machine = ItemOf<MachinePayload['machines']>
export type MachineTestResponse = DeepRequired<
  ResponseFor<'/api/v1/machines/{machineId}/test', 'post'>
>
export type MachineProbe = MachineTestResponse['probe']
export type MachineResourcesResponse = DeepRequired<
  ResponseFor<'/api/v1/machines/{machineId}/resources', 'get'>
>

export type AgentPayload = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}/agents', 'get'>>
export type Agent = ItemOf<AgentPayload['agents']>
export type AgentResponse = DeepRequired<ResponseFor<'/api/v1/agents/{agentId}', 'get'>>

export type ActivityPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/activity', 'get'>
>
export type ActivityEvent = ItemOf<ActivityPayload['events']>

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
export type TicketCreateResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets', 'post'>
>
export type TicketResponse = DeepRequired<ResponseFor<'/api/v1/tickets/{ticketId}', 'patch'>>
export type Ticket = ItemOf<TicketPayload['tickets']>
export type TicketPriority = Ticket['priority']
export type TicketReference = ItemOf<Ticket['children']>
export type TicketDependency = ItemOf<Ticket['dependencies']>
export type TicketDependencyResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/dependencies', 'post'>
>
export type TicketDependencyDeleteResponse = DeepRequired<
  ResponseFor<'/api/v1/tickets/{ticketId}/dependencies/{dependencyId}', 'delete'>
>

export type ProjectRepoPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/repos', 'get'>
>
export type ProjectRepoResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/repos', 'post'>
>
export type ProjectRepoRecord = ItemOf<ProjectRepoPayload['repos']>

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
export type Workflow = ItemOf<WorkflowListPayload['workflows']>
export type WorkflowDetailPayload = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}', 'get'>
>

export type HarnessPayload = DeepRequired<
  ResponseFor<'/api/v1/workflows/{workflowId}/harness', 'get'>
>
export type HarnessDocument = HarnessPayload['harness']
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

export type BuiltinRolePayload = DeepRequired<ResponseFor<'/api/v1/roles/builtin', 'get'>>
export type BuiltinRole = ItemOf<BuiltinRolePayload['roles']>

export type HRAdvisorResponse = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/hr-advisor', 'get'>
>
export type HRAdvisorSummary = HRAdvisorResponse['summary']
export type HRAdvisorStaffing = HRAdvisorResponse['staffing']
export type HRAdvisorRecommendation = ItemOf<HRAdvisorResponse['recommendations']>

export type TicketDetailPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/tickets/{ticketId}/detail', 'get'>
>
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
