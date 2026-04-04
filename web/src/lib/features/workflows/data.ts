import {
  createWorkflow,
  getWorkflowHarness,
  listWorkflowHarnessHistory,
  listAgents,
  listHarnessVariables,
  listBuiltinRoles,
  listProviders,
  listSkills,
  listStatuses,
  listWorkflows,
} from '$lib/api/openase'
import type { TicketStatusStage } from '$lib/features/statuses/public'
import { defaultHarnessTemplate, toHarnessContent } from './model'
import type { SkillState } from './model'
import type {
  HarnessVariableGroup,
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
  WorkflowVersionSummary,
} from './types'
import { mergeWorkflowHooksPayload, type WorkflowHooksPayload } from './workflow-hooks'
import { buildWorkflowSummary } from './workflow-summary'

export function mapWorkflowSummary(
  workflow: Awaited<ReturnType<typeof listWorkflows>>['workflows'][number],
  statusNamesById: Map<string, string>,
): WorkflowSummary {
  return buildWorkflowSummary(workflow, {
    resolveStatusName: (statusId) => statusNamesById.get(statusId) ?? statusId,
  })
}

function mapWorkflowVersionHistory(
  history: Awaited<ReturnType<typeof listWorkflowHarnessHistory>>['history'],
): WorkflowVersionSummary[] {
  return history.map((item) => ({
    id: item.id,
    version: item.version,
    createdBy: item.created_by,
    createdAt: item.created_at,
  }))
}

export function mapWorkflowAgentOptions(
  agents: Awaited<ReturnType<typeof listAgents>>['agents'],
  providers: Awaited<ReturnType<typeof listProviders>>['providers'],
): WorkflowAgentOption[] {
  const providersById = new Map(providers.map((provider) => [provider.id, provider]))

  return agents
    .filter((agent) => agent.runtime_control_state !== 'retired')
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
        adapterType: provider?.adapter_type ?? 'custom',
        available: provider?.available ?? false,
      }
    })
    .sort((left, right) => left.label.localeCompare(right.label))
}

export function mapStatusOptions(
  statuses: Awaited<ReturnType<typeof listStatuses>>['statuses'],
): WorkflowStatusOption[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({
      id: status.id,
      name: status.name,
      stage: (status.stage as TicketStatusStage) ?? 'unstarted',
    }))
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
      builtinRolePayload.roles.find((role) => role.slug === 'fullstack-developer')?.content ??
      defaultHarnessTemplate(),
    providers: catalog.providers,
    statuses: catalog.statuses,
    skillStates: mapSkillStates(skillPayload.skills, currentWorkflowId),
    variableGroups: variablePayload.groups as HarnessVariableGroup[],
  }
}

export async function loadWorkflowHarness(projectId: string, workflowId: string) {
  const [harnessPayload, historyPayload, skillPayload] = await Promise.all([
    getWorkflowHarness(workflowId),
    listWorkflowHarnessHistory(workflowId),
    listSkills(projectId),
  ])

  return {
    harness: toHarnessContent(harnessPayload.harness.content),
    history: mapWorkflowVersionHistory(historyPayload.history),
    skillStates: mapSkillStates(skillPayload.skills, workflowId),
  }
}

export async function loadWorkflowPageData(projectId: string, orgId: string, selectedId: string) {
  const index = await loadWorkflowIndex(projectId, orgId, selectedId)
  const selectedWorkflowId = selectedId || index.workflows[0]?.id || ''
  const harnessPayload = selectedWorkflowId
    ? await loadWorkflowHarness(projectId, selectedWorkflowId)
    : null

  return {
    agentOptions: index.agentOptions,
    workflows: index.workflows.map((workflow) =>
      workflow.id === selectedWorkflowId
        ? { ...workflow, history: harnessPayload?.history ?? workflow.history }
        : workflow,
    ),
    builtinRoleContent: index.builtinRoleContent,
    providers: index.providers,
    statuses: index.statuses,
    skillStates: harnessPayload?.skillStates ?? index.skillStates,
    variableGroups: index.variableGroups,
    selectedWorkflowId,
    harness: harnessPayload?.harness ?? null,
  }
}

export async function createWorkflowWithBinding(
  projectId: string,
  input: {
    agentId: string
    name: string
    workflowType: string
    roleSlug?: string
    roleName?: string
    roleDescription?: string
    platformAccessAllowed?: string[]
    skillNames?: string[]
    harnessPath?: string | null
    pickupStatusIds: string[]
    finishStatusIds: string[]
    hooks?: WorkflowHooksPayload
  },
  statuses: WorkflowStatusOption[],
  builtinRoleContent: string,
) {
  const response = await createWorkflow(projectId, {
    agent_id: input.agentId,
    name: input.name,
    type: input.workflowType,
    role_slug: input.roleSlug,
    role_name: input.roleName,
    role_description: input.roleDescription,
    platform_access_allowed: input.platformAccessAllowed ?? [],
    skill_names: input.skillNames ?? [],
    harness_path: input.harnessPath ?? null,
    pickup_status_ids: input.pickupStatusIds,
    finish_status_ids: input.finishStatusIds,
    harness_content: builtinRoleContent || defaultHarnessTemplate(),
    is_active: true,
    max_concurrent: 0,
    max_retry_attempts: 1,
    timeout_minutes: 30,
    hooks: mergeWorkflowHooksPayload(input.hooks),
  })

  if (!response.workflow) {
    throw new Error('Failed to create workflow: no workflow data returned from API.')
  }

  const createdWorkflow = response.workflow
  const workflow = buildWorkflowSummary(createdWorkflow, {
    resolveStatusName: (statusId) =>
      statuses.find((status) => status.id === statusId)?.name ?? statusId,
    fallbackMetadata: {
      roleSlug: input.roleSlug,
      roleName: input.roleName,
      roleDescription: input.roleDescription,
      platformAccessAllowed: input.platformAccessAllowed,
    },
  })

  return {
    workflow,
    selectedId: createdWorkflow.id,
  }
}

function mapSkillStates(
  skills: Array<{
    id: string
    name: string
    description: string
    path: string
    bound_workflows: Array<{ id: string }>
  }>,
  workflowId?: string,
): SkillState[] {
  return skills.map((skill) => ({
    id: skill.id,
    name: skill.name,
    description: skill.description,
    path: skill.path,
    bound: Boolean(skill.bound_workflows.some((workflow) => workflow.id === workflowId)),
  }))
}
