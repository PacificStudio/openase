import type { WorkflowSummary } from './types'

export type WorkflowLifecycleDraft = {
  name: string
  pickupStatusId: string
  finishStatusId: string
  maxConcurrent: string
  maxRetryAttempts: string
  timeoutMinutes: string
  stallTimeoutMinutes: string
  isActive: boolean
}

export type WorkflowLifecyclePayload = {
  finish_status_id: string | null
  is_active: boolean
  max_concurrent: number
  max_retry_attempts: number
  name: string
  pickup_status_id: string
  stall_timeout_minutes: number
  timeout_minutes: number
}

type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export function createWorkflowLifecycleDraft(workflow: WorkflowSummary): WorkflowLifecycleDraft {
  return {
    name: workflow.name,
    pickupStatusId: workflow.pickupStatusId,
    finishStatusId: workflow.finishStatusId ?? '',
    maxConcurrent: String(workflow.maxConcurrent),
    maxRetryAttempts: String(workflow.maxRetry),
    timeoutMinutes: String(workflow.timeoutMinutes),
    stallTimeoutMinutes: String(Math.max(workflow.stallTimeoutMinutes, 1)),
    isActive: workflow.isActive,
  }
}

export function parseWorkflowLifecycleDraft(
  draft: WorkflowLifecycleDraft,
): ParseResult<WorkflowLifecyclePayload> {
  const name = draft.name.trim()
  if (!name) {
    return { ok: false, error: 'Name must not be empty.' }
  }
  if (!draft.pickupStatusId) {
    return { ok: false, error: 'Pickup status is required.' }
  }

  const maxConcurrent = parseIntegerField(draft.maxConcurrent, 'Max concurrent', 1)
  if (!maxConcurrent.ok) return maxConcurrent

  const maxRetryAttempts = parseIntegerField(draft.maxRetryAttempts, 'Max retry', 0)
  if (!maxRetryAttempts.ok) return maxRetryAttempts

  const timeoutMinutes = parseIntegerField(draft.timeoutMinutes, 'Timeout', 1)
  if (!timeoutMinutes.ok) return timeoutMinutes

  const stallTimeoutMinutes = parseIntegerField(draft.stallTimeoutMinutes, 'Stall timeout', 1)
  if (!stallTimeoutMinutes.ok) return stallTimeoutMinutes

  return {
    ok: true,
    value: {
      finish_status_id: draft.finishStatusId || null,
      is_active: draft.isActive,
      max_concurrent: maxConcurrent.value,
      max_retry_attempts: maxRetryAttempts.value,
      name,
      pickup_status_id: draft.pickupStatusId,
      stall_timeout_minutes: stallTimeoutMinutes.value,
      timeout_minutes: timeoutMinutes.value,
    },
  }
}

function parseIntegerField(value: string, label: string, minimum: number): ParseResult<number> {
  const normalized = value.trim()
  if (!normalized) {
    return { ok: false, error: `${label} is required.` }
  }

  if (!/^\d+$/.test(normalized)) {
    return { ok: false, error: `${label} must be a whole number.` }
  }

  const parsed = Number.parseInt(normalized, 10)
  if (parsed < minimum) {
    return {
      ok: false,
      error:
        minimum === 0
          ? `${label} must be zero or greater.`
          : `${label} must be at least ${minimum}.`,
    }
  }

  return { ok: true, value: parsed }
}
