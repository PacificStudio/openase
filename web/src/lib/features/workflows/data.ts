import {
  createWorkflow,
  getWorkflowHarness,
  listHarnessVariables,
  listBuiltinRoles,
  listSkills,
  listStatuses,
  listWorkflows,
} from '$lib/api/openase'
import { defaultHarnessTemplate, normalizeWorkflowType, toHarnessContent } from './model'
import type { SkillState } from './model'
import type { HarnessVariableGroup, WorkflowSummary } from './types'

function mapWorkflowSummary(
  workflow: Awaited<ReturnType<typeof listWorkflows>>['workflows'][number],
  statusNamesById: Map<string, string>,
): WorkflowSummary {
  return {
    id: workflow.id,
    name: workflow.name,
    type: normalizeWorkflowType(workflow.type),
    harnessPath: workflow.harness_path ?? '',
    pickupStatus: statusNamesById.get(workflow.pickup_status_id) ?? workflow.pickup_status_id,
    finishStatus: workflow.finish_status_id
      ? (statusNamesById.get(workflow.finish_status_id) ?? workflow.finish_status_id)
      : 'unchanged',
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

export async function loadWorkflowIndex(projectId: string, selectedId: string) {
  const [workflowPayload, skillPayload, builtinRolePayload, statusPayload, variablePayload] =
    await Promise.all([
      listWorkflows(projectId),
      listSkills(projectId),
      listBuiltinRoles(),
      listStatuses(projectId),
      listHarnessVariables(),
    ])

  const statusNamesById = new Map(statusPayload.statuses.map((status) => [status.id, status.name]))
  const workflows = workflowPayload.workflows.map((workflow) =>
    mapWorkflowSummary(workflow, statusNamesById),
  )
  const currentWorkflowId = selectedId || workflows[0]?.id

  return {
    workflows,
    builtinRoleContent:
      builtinRolePayload.roles.find((role) => role.workflow_type === 'coding')?.content ??
      defaultHarnessTemplate(),
    statuses: statusPayload.statuses
      .slice()
      .sort((left, right) => left.position - right.position)
      .map((status) => ({ id: status.id, name: status.name })),
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

export async function createDefaultWorkflow(
  projectId: string,
  existingCount: number,
  statuses: Array<{ id: string; name: string }>,
  builtinRoleContent: string,
) {
  const index = existingCount + 1
  const payload = await createWorkflow(projectId, {
    name: `Workflow ${index}`,
    type: 'coding',
    pickup_status_id: statuses[0].id,
    finish_status_id: statuses.at(-1)?.id ?? statuses[0].id,
    harness_content: builtinRoleContent || defaultHarnessTemplate(),
    is_active: true,
    max_concurrent: 1,
    max_retry_attempts: 1,
    timeout_minutes: 30,
  })

  if (!payload.workflow) {
    throw new Error('Failed to create workflow: no workflow data returned from API.')
  }

  const createdWorkflow = payload.workflow

  const workflow: WorkflowSummary = {
    id: createdWorkflow.id,
    name: createdWorkflow.name,
    type: normalizeWorkflowType(createdWorkflow.type),
    harnessPath: createdWorkflow.harness_path ?? '',
    pickupStatus:
      statuses.find((status) => status.id === createdWorkflow.pickup_status_id)?.name ??
      createdWorkflow.pickup_status_id,
    finishStatus:
      statuses.find((status) => status.id === createdWorkflow.finish_status_id)?.name ??
      createdWorkflow.finish_status_id ??
      statuses.at(-1)?.name ??
      statuses[0].name,
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
