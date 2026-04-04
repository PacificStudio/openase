import { ApiError } from '$lib/api/client'
import type { WorkflowImpact } from './types'

function workflowConflictMessage(code: string, detail: string): string | null {
  switch (code) {
    case 'WORKFLOW_NAME_CONFLICT':
      return 'A workflow with this name already exists in the project.'
    case 'WORKFLOW_HARNESS_PATH_CONFLICT':
      return 'This harness path is already used by another workflow.'
    case 'WORKFLOW_STATUS_BINDING_OVERLAP':
      return 'Pickup and finish statuses must be mutually exclusive.'
    case 'WORKFLOW_REFERENCED_BY_TICKETS':
      return 'This workflow cannot be deleted because tickets still reference it.'
    case 'WORKFLOW_REFERENCED_BY_SCHEDULED_JOBS':
      return 'This workflow cannot be deleted because scheduled jobs still reference it.'
    case 'WORKFLOW_IN_USE':
      return 'This workflow cannot be deleted because it is still referenced.'
    case 'WORKFLOW_REPLACEMENT_REQUIRED':
      return 'Replace ticket and scheduled job references before permanently deleting this workflow.'
    case 'WORKFLOW_ACTIVE_AGENT_RUNS':
      return 'This workflow cannot be deleted while agent runs are still active.'
    case 'WORKFLOW_HISTORICAL_AGENT_RUNS':
      return 'This workflow cannot be deleted because historical agent runs still reference it.'
    default:
      return detail.trim() || null
  }
}

export function describeWorkflowApiError(error: unknown, fallback: string): string {
  if (!(error instanceof ApiError)) return fallback

  const detail = error.detail.trim()
  if (error.code) {
    const message = workflowConflictMessage(error.code, detail)
    if (message) return message
  }
  return detail || fallback
}

export function workflowImpactFromError(error: unknown): WorkflowImpact | null {
  if (!(error instanceof ApiError)) return null
  const details = error.details
  if (!details || typeof details !== 'object') return null
  const candidate = details as Partial<WorkflowImpact>
  if (typeof candidate.workflow_id !== 'string' || !candidate.summary) return null
  return candidate as WorkflowImpact
}

export function describeWorkflowImpact(impact: WorkflowImpact): string {
  const parts: string[] = []
  if (impact.summary.ticket_count > 0) {
    parts.push(`${impact.summary.ticket_count} active tickets`)
  }
  if (impact.summary.scheduled_job_count > 0) {
    parts.push(`${impact.summary.scheduled_job_count} scheduled jobs`)
  }
  if (impact.summary.active_agent_run_count > 0) {
    parts.push(`${impact.summary.active_agent_run_count} active agent runs`)
  }
  if (impact.summary.historical_agent_run_count > 0) {
    parts.push(`${impact.summary.historical_agent_run_count} historical agent runs`)
  }
  if (parts.length === 0) {
    return 'No references block permanent deletion.'
  }
  return parts.join(', ')
}
