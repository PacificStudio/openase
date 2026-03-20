import { deleteWorkflow, updateWorkflow } from '$lib/api/openase'
import type { WorkflowStatusOption, WorkflowSummary } from './types'
import { mapWorkflowSummary } from './data'
import type { WorkflowLifecyclePayload } from './workflow-lifecycle'

export async function saveWorkflowLifecycle(
  workflowId: string,
  payload: WorkflowLifecyclePayload,
  statuses: WorkflowStatusOption[],
): Promise<WorkflowSummary> {
  const response = await updateWorkflow(workflowId, payload)
  const statusNamesById = new Map(statuses.map((status) => [status.id, status.name]))
  return mapWorkflowSummary(response.workflow, statusNamesById)
}

export async function destroyWorkflow(workflowId: string) {
  await deleteWorkflow(workflowId)
}

export function removeWorkflowFromList(workflows: WorkflowSummary[], workflowId: string) {
  const remaining = workflows.filter((workflow) => workflow.id !== workflowId)

  return {
    remaining,
    nextSelectedId: remaining[0]?.id ?? '',
  }
}
