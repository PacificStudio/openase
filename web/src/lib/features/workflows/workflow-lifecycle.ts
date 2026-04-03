import type { WorkflowSummary } from './types'

export type WorkflowLifecycleDraft = {
  agentId: string
  name: string
  typeLabel: string
  pickupStatusIds: string[]
  finishStatusIds: string[]
  maxConcurrent: string
  maxRetryAttempts: string
  timeoutMinutes: string
  stallTimeoutMinutes: string
  isActive: boolean
}

export type WorkflowLifecyclePayload = {
  agent_id: string
  finish_status_ids: string[]
  hooks?: Record<string, unknown> | null
  is_active: boolean
  max_concurrent: number
  max_retry_attempts: number
  name: string
  type: string
  pickup_status_ids: string[]
  stall_timeout_minutes: number
  timeout_minutes: number
}

type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

export function createWorkflowLifecycleDraft(workflow: WorkflowSummary): WorkflowLifecycleDraft {
  return {
    agentId: workflow.agentId ?? '',
    name: workflow.name,
    typeLabel: workflow.type,
    pickupStatusIds: [...workflow.pickupStatusIds],
    finishStatusIds: [...workflow.finishStatusIds],
    maxConcurrent: workflow.maxConcurrent > 0 ? String(workflow.maxConcurrent) : '',
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
  if (!draft.agentId) {
    return { ok: false, error: 'Bound agent is required.' }
  }
  if (!draft.typeLabel.trim()) {
    return { ok: false, error: 'Workflow type label must not be empty.' }
  }
  if (draft.pickupStatusIds.length === 0) {
    return { ok: false, error: 'At least one pickup status is required.' }
  }
  if (draft.finishStatusIds.length === 0) {
    return { ok: false, error: 'At least one finish status is required.' }
  }

  const maxConcurrent = parseOptionalPositiveIntegerField(draft.maxConcurrent, 'Max concurrent')
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
      agent_id: draft.agentId,
      finish_status_ids: [...draft.finishStatusIds],
      is_active: draft.isActive,
      max_concurrent: maxConcurrent.value,
      max_retry_attempts: maxRetryAttempts.value,
      name,
      type: draft.typeLabel.trim(),
      pickup_status_ids: [...draft.pickupStatusIds],
      stall_timeout_minutes: stallTimeoutMinutes.value,
      timeout_minutes: timeoutMinutes.value,
    },
  }
}

export function toggleWorkflowStatusSelection(selected: string[], statusId: string) {
  return selected.includes(statusId)
    ? selected.filter((value) => value !== statusId)
    : [...selected, statusId]
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

function parseOptionalPositiveIntegerField(value: string, label: string): ParseResult<number> {
  const normalized = value.trim()
  if (!normalized) {
    return { ok: true, value: 0 }
  }

  if (!/^\d+$/.test(normalized)) {
    return { ok: false, error: `${label} must be a whole number.` }
  }

  const parsed = Number.parseInt(normalized, 10)
  if (parsed < 1) {
    return { ok: false, error: `${label} must be a positive integer.` }
  }

  return { ok: true, value: parsed }
}
