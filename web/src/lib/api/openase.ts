import { api } from './client'
import type {
  ActivityPayload,
  AgentPayload,
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
  ProjectPayload,
  ProjectResponse,
  SkillListPayload,
  StatusPayload,
  TicketDetailPayload,
  TicketPayload,
  Organization,
  WorkflowDetailPayload,
  WorkflowListPayload,
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

export function listOrganizations() {
  return api.get<{ organizations?: Organization[] }>('/api/v1/orgs')
}

export function listProjects(orgId: string) {
  return api.get<ProjectPayload>(`/api/v1/orgs/${orgId}/projects`)
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

export function getProject(projectId: string) {
  return api.get<ProjectResponse>(`/api/v1/projects/${projectId}`)
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

export function listActivity(
  projectId: string,
  params?: {
    agent_id?: string
    ticket_id?: string
    limit?: string
  },
) {
  return api.get<ActivityPayload>(`/api/v1/projects/${projectId}/activity`, { params })
}

export function listAgents(projectId: string) {
  return api.get<AgentPayload>(`/api/v1/projects/${projectId}/agents`)
}

export function listStatuses(projectId: string) {
  return api.get<StatusPayload>(`/api/v1/projects/${projectId}/statuses`)
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

export function getTicketDetail(projectId: string, ticketId: string) {
  return api.get<TicketDetailPayload>(`/api/v1/projects/${projectId}/tickets/${ticketId}/detail`)
}

export function listWorkflows(projectId: string) {
  return api.get<WorkflowListPayload>(`/api/v1/projects/${projectId}/workflows`)
}

export function createWorkflow(
  projectId: string,
  body: {
    finish_status_id?: string | null
    harness_content?: string
    harness_path?: string | null
    hooks?: Record<string, unknown>
    is_active?: boolean | null
    max_concurrent?: number | null
    max_retry_attempts?: number | null
    name?: string
    pickup_status_id?: string
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
