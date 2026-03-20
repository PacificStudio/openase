import {
  createWorkflow,
  getWorkflowHarness,
  listBuiltinRoles,
  listSkills,
  listStatuses,
  listWorkflows,
} from '$lib/api/openase'
import {
  defaultHarnessTemplate,
  extractBody,
  extractFrontmatter,
  normalizeWorkflowType,
} from './model'
import type { SkillState } from './model'
import type { HarnessContent, WorkflowSummary } from './types'

export async function loadWorkflowIndex(projectId: string, selectedId: string) {
  const [workflowPayload, skillPayload, builtinRolePayload, statusPayload] = await Promise.all([
    listWorkflows(projectId),
    listSkills(projectId),
    listBuiltinRoles(),
    listStatuses(projectId),
  ])

  const workflows = workflowPayload.workflows.map((workflow) => ({
    id: workflow.id,
    name: workflow.name,
    type: normalizeWorkflowType(workflow.type),
    pickupStatus: workflow.pickup_status_id,
    finishStatus: workflow.finish_status_id ?? 'unchanged',
    maxConcurrent: workflow.max_concurrent,
    maxRetry: workflow.max_retry_attempts,
    timeoutMinutes: workflow.timeout_minutes,
    isActive: workflow.is_active,
    lastModified: new Date().toISOString(),
    recentSuccessRate: 0,
    version: workflow.version,
  }))

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
  }
}

export async function loadWorkflowHarness(projectId: string, workflowId: string) {
  const [harnessPayload, skillPayload] = await Promise.all([
    getWorkflowHarness(workflowId),
    listSkills(projectId),
  ])

  const content = harnessPayload.harness.content
  const harness: HarnessContent = {
    frontmatter: extractFrontmatter(content),
    body: extractBody(content),
    rawContent: content,
  }

  return {
    harness,
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

  const workflow: WorkflowSummary = {
    id: payload.workflow.id,
    name: payload.workflow.name,
    type: normalizeWorkflowType(payload.workflow.type),
    pickupStatus: payload.workflow.pickup_status_id,
    finishStatus: payload.workflow.finish_status_id ?? statuses.at(-1)?.id ?? statuses[0].id,
    maxConcurrent: payload.workflow.max_concurrent,
    maxRetry: payload.workflow.max_retry_attempts,
    timeoutMinutes: payload.workflow.timeout_minutes,
    isActive: payload.workflow.is_active,
    lastModified: new Date().toISOString(),
    recentSuccessRate: 0,
    version: payload.workflow.version,
  }

  return {
    workflow,
    selectedId: payload.workflow.id,
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
