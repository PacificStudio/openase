import {
  deleteWorkflow,
  getWorkflowImpact,
  replaceWorkflowReferences,
  retireWorkflow,
  updateWorkflow,
} from '$lib/api/openase'
import type {
  WorkflowImpact,
  WorkflowReplaceReferencesResult,
  WorkflowStatusOption,
  WorkflowSummary,
} from './types'
import { mapWorkflowSummary } from './data'
import type { WorkflowLifecyclePayload } from './workflow-lifecycle'
import { mergeWorkflowHooksPayload } from './workflow-hooks'

export async function saveWorkflowLifecycle(
  workflowId: string,
  payload: WorkflowLifecyclePayload,
  statuses: WorkflowStatusOption[],
  currentWorkflow?: WorkflowSummary,
): Promise<WorkflowSummary> {
  const response = await updateWorkflow(workflowId, {
    ...payload,
    hooks: mergeWorkflowHooksPayload(payload.hooks ?? undefined, currentWorkflow?.rawHooks),
  })
  const statusNamesById = new Map(statuses.map((status) => [status.id, status.name]))
  const nextWorkflow = mapWorkflowSummary(response.workflow, statusNamesById)
  return {
    ...nextWorkflow,
    history: currentWorkflow?.history ?? nextWorkflow.history,
    lastModified: currentWorkflow?.lastModified ?? nextWorkflow.lastModified,
  }
}

export async function destroyWorkflow(workflowId: string) {
  await deleteWorkflow(workflowId)
}

export async function retireWorkflowLifecycle(
  workflowId: string,
  statuses: WorkflowStatusOption[],
  currentWorkflow?: WorkflowSummary,
): Promise<WorkflowSummary> {
  const response = await retireWorkflow(workflowId)
  const statusNamesById = new Map(statuses.map((status) => [status.id, status.name]))
  const nextWorkflow = mapWorkflowSummary(response.workflow, statusNamesById)
  return {
    ...nextWorkflow,
    history: currentWorkflow?.history ?? nextWorkflow.history,
    lastModified: currentWorkflow?.lastModified ?? nextWorkflow.lastModified,
  }
}

export async function loadWorkflowImpact(workflowId: string): Promise<WorkflowImpact> {
  const response = await getWorkflowImpact(workflowId)
  return response.impact
}

export async function replaceWorkflowLifecycleReferences(
  workflowId: string,
  replacementWorkflowId: string,
): Promise<WorkflowReplaceReferencesResult> {
  const response = await replaceWorkflowReferences(workflowId, {
    replacement_workflow_id: replacementWorkflowId,
  })
  return response.result
}

export function removeWorkflowFromList(workflows: WorkflowSummary[], workflowId: string) {
  const remaining = workflows.filter((workflow) => workflow.id !== workflowId)

  return {
    remaining,
    nextSelectedId: remaining[0]?.id ?? '',
  }
}
