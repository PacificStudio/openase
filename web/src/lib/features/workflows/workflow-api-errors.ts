import { ApiError } from '$lib/api/client'

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
