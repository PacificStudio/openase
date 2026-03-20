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

export type AgentPayload = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}/agents', 'get'>>
export type Agent = ItemOf<AgentPayload['agents']>

export type ActivityPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/activity', 'get'>
>
export type ActivityEvent = ItemOf<ActivityPayload['events']>

export type StatusPayload = DeepRequired<
  ResponseFor<'/api/v1/projects/{projectId}/statuses', 'get'>
>
export type TicketStatus = ItemOf<StatusPayload['statuses']>

export type TicketPayload = DeepRequired<ResponseFor<'/api/v1/projects/{projectId}/tickets', 'get'>>
export type TicketResponse = DeepRequired<ResponseFor<'/api/v1/tickets/{ticketId}', 'patch'>>
export type Ticket = ItemOf<TicketPayload['tickets']>
export type TicketPriority = Ticket['priority']
export type TicketReference = ItemOf<Ticket['children']>
export type TicketDependency = ItemOf<Ticket['dependencies']>

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
