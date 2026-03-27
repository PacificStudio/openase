import {
  createWorkflow,
  getWorkflowHarness,
  listAgents,
  listHarnessVariables,
  listBuiltinRoles,
  listProviders,
  listSkills,
  listStatuses,
  listWorkflows,
} from '$lib/api/openase'
import { defaultHarnessTemplate, normalizeWorkflowType, toHarnessContent } from './model'
import type { SkillState } from './model'
import type {
  HarnessVariableGroup,
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
} from './types'

export function mapWorkflowSummary(
  workflow: Awaited<ReturnType<typeof listWorkflows>>['workflows'][number],
  statusNamesById: Map<string, string>,
): WorkflowSummary {
  const pickupStatusIds = workflow.pickup_status_ids ?? []
  const finishStatusIds = workflow.finish_status_ids ?? []

  return {
    id: workflow.id,
    name: workflow.name,
    type: normalizeWorkflowType(workflow.type),
    agentId: workflow.agent_id ?? null,
    harnessPath: workflow.harness_path ?? '',
    pickupStatusIds,
    pickupStatusLabel: pickupStatusIds
      .map((statusId) => statusNamesById.get(statusId) ?? statusId)
      .join(', '),
    finishStatusIds,
    finishStatusLabel: finishStatusIds
      .map((statusId) => statusNamesById.get(statusId) ?? statusId)
      .join(', '),
    maxConcurrent: workflow.max_concurrent,
    maxRetry: workflow.max_retry_attempts,
    timeoutMinutes: workflow.timeout_minutes,
    stallTimeoutMinutes: workflow.stall_timeout_minutes ?? 0,
    isActive: workflow.is_active,
    lastModified: new Date().toISOString(),
    recentSuccessRate: 0,
    version: workflow.version,
  }
}

function mapWorkflowAgentOptions(
  agents: Awaited<ReturnType<typeof listAgents>>['agents'],
  providers: Awaited<ReturnType<typeof listProviders>>['providers'],
): WorkflowAgentOption[] {
  const providersById = new Map(providers.map((provider) => [provider.id, provider]))

  return agents
    .map((agent) => {
      const provider = providersById.get(agent.provider_id)
      const providerName = provider?.name ?? 'Unknown provider'
      const modelName = provider?.model_name ?? 'Unknown model'
      const machineName = provider?.machine_name ?? 'Unknown machine'

      return {
        id: agent.id,
        label: `${agent.name} · ${providerName} · ${machineName} · ${modelName}`,
        agentName: agent.name,
        providerName,
        modelName,
        machineName,
      }
    })
    .sort((left, right) => left.label.localeCompare(right.label))
}

function mapStatusOptions(
  statuses: Awaited<ReturnType<typeof listStatuses>>['statuses'],
): WorkflowStatusOption[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({ id: status.id, name: status.name }))
}

export async function loadWorkflowCatalog(projectId: string, orgId: string) {
  const [workflowPayload, statusPayload, agentPayload, providerPayload] = await Promise.all([
    listWorkflows(projectId),
    listStatuses(projectId),
    listAgents(projectId),
    listProviders(orgId),
  ])
  const statuses = mapStatusOptions(statusPayload.statuses)
  const statusNamesById = new Map(statuses.map((status) => [status.id, status.name]))
  const agentOptions = mapWorkflowAgentOptions(agentPayload.agents, providerPayload.providers)

  return {
    agentOptions,
    providers: providerPayload.providers,
    statuses,
    workflows: workflowPayload.workflows.map((workflow) =>
      mapWorkflowSummary(workflow, statusNamesById),
    ),
  }
}

export async function loadWorkflowIndex(projectId: string, orgId: string, selectedId: string) {
  const [catalog, skillPayload, builtinRolePayload, variablePayload] = await Promise.all([
    loadWorkflowCatalog(projectId, orgId),
    listSkills(projectId),
    listBuiltinRoles(),
    listHarnessVariables(),
  ])
  const currentWorkflowId = selectedId || catalog.workflows[0]?.id

  return {
    agentOptions: catalog.agentOptions,
    workflows: catalog.workflows,
    builtinRoleContent:
      builtinRolePayload.roles.find((role) => role.workflow_type === 'coding')?.content ??
      defaultHarnessTemplate(),
    providers: catalog.providers,
    statuses: catalog.statuses,
    skillStates: mapSkillStates(skillPayload.skills, currentWorkflowId),
    variableGroups: variablePayload.groups as HarnessVariableGroup[],
  }
}

export async function loadWorkflowHarness(projectId: string, workflowId: string) {
  const [harnessPayload, skillPayload] = await Promise.all([
    getWorkflowHarness(workflowId),
    listSkills(projectId),
  ])

  return {
    harness: toHarnessContent(harnessPayload.harness.content),
    skillStates: mapSkillStates(skillPayload.skills, workflowId),
  }
}

export async function createWorkflowWithBinding(
  projectId: string,
  input: {
    agentId: string
    name: string
    pickupStatusIds: string[]
    finishStatusIds: string[]
  },
  statuses: WorkflowStatusOption[],
  builtinRoleContent: string,
) {
  const response = await createWorkflow(projectId, {
    agent_id: input.agentId,
    name: input.name,
    type: 'coding',
    pickup_status_ids: input.pickupStatusIds,
    finish_status_ids: input.finishStatusIds,
    harness_content: builtinRoleContent || defaultHarnessTemplate(),
    is_active: true,
    max_concurrent: 1,
    max_retry_attempts: 1,
    timeout_minutes: 30,
  })

  if (!response.workflow) {
    throw new Error('Failed to create workflow: no workflow data returned from API.')
  }

  const createdWorkflow = response.workflow

  const workflow: WorkflowSummary = {
    id: createdWorkflow.id,
    name: createdWorkflow.name,
    type: normalizeWorkflowType(createdWorkflow.type),
    agentId: createdWorkflow.agent_id ?? null,
    harnessPath: createdWorkflow.harness_path ?? '',
    pickupStatusIds: createdWorkflow.pickup_status_ids,
    pickupStatusLabel: createdWorkflow.pickup_status_ids
      .map((statusId) => statuses.find((status) => status.id === statusId)?.name ?? statusId)
      .join(', '),
    finishStatusIds: createdWorkflow.finish_status_ids,
    finishStatusLabel: createdWorkflow.finish_status_ids
      .map((statusId) => statuses.find((status) => status.id === statusId)?.name ?? statusId)
      .join(', '),
    maxConcurrent: createdWorkflow.max_concurrent,
    maxRetry: createdWorkflow.max_retry_attempts,
    timeoutMinutes: createdWorkflow.timeout_minutes,
    stallTimeoutMinutes: createdWorkflow.stall_timeout_minutes ?? 0,
    isActive: createdWorkflow.is_active,
    lastModified: new Date().toISOString(),
    recentSuccessRate: 0,
    version: createdWorkflow.version,
  }

  return {
    workflow,
    selectedId: createdWorkflow.id,
  }
}

function mapSkillStates(
  skills: Array<{
    name: string
    description: string
    path: string
    bound_workflows: Array<{ id: string }>
  }>,
  workflowId?: string,
): SkillState[] {
  return skills.map((skill) => ({
    name: skill.name,
    description: skill.description,
    path: skill.path,
    bound: Boolean(skill.bound_workflows.some((workflow) => workflow.id === workflowId)),
  }))
}
