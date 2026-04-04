import type { WorkflowStatusOption, WorkflowSummary } from './types'

export type WorkflowLifecycleDraft = {
  agentId: string
  name: string
  typeLabel: string
  roleName?: string
  roleDescription?: string
  platformAccessAllowed?: string
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
  role_description?: string
  role_name?: string
  platform_access_allowed?: string[]
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
    roleName: workflow.roleName ?? workflow.name,
    roleDescription: workflow.roleDescription ?? '',
    platformAccessAllowed: (workflow.platformAccessAllowed ?? []).join('\n'),
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
  const roleName = draft.roleName?.trim() || name
  if (draft.pickupStatusIds.length === 0) {
    return { ok: false, error: 'At least one pickup status is required.' }
  }
  if (draft.finishStatusIds.length === 0) {
    return { ok: false, error: 'At least one finish status is required.' }
  }
  if (findOverlappingStatusIds(draft.pickupStatusIds, draft.finishStatusIds).length > 0) {
    return { ok: false, error: 'Pickup and finish statuses must be mutually exclusive.' }
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
      role_description: draft.roleDescription?.trim() || '',
      role_name: roleName,
      platform_access_allowed: parseStringList(draft.platformAccessAllowed ?? ''),
      pickup_status_ids: [...draft.pickupStatusIds],
      stall_timeout_minutes: stallTimeoutMinutes.value,
      timeout_minutes: timeoutMinutes.value,
    },
  }
}

export function toggleWorkflowStatusSelection(
  selected: string[],
  statusId: string,
  blockedMap?: Record<string, string>,
) {
  if (!selected.includes(statusId) && blockedMap?.[statusId]) {
    return selected
  }
  return selected.includes(statusId)
    ? selected.filter((value) => value !== statusId)
    : [...selected, statusId]
}

/**
 * Build a map of pickup status IDs blocked by other workflows.
 * Returns `{ [statusId]: reason }` for statuses that are already bound
 * as pickup statuses by a workflow other than `excludeWorkflowId`.
 */
export function buildPickupStatusBlockedReasonMap(
  workflows: ReadonlyArray<{ id: string; name: string; pickupStatusIds: string[] }>,
  excludeWorkflowId?: string,
): Record<string, string> {
  const map: Record<string, string> = {}
  for (const workflow of workflows) {
    if (workflow.id === excludeWorkflowId) continue
    for (const statusId of workflow.pickupStatusIds) {
      if (!map[statusId]) {
        map[statusId] = `Used by "${workflow.name}" as a pickup status.`
      }
    }
  }
  return map
}

export function buildSelfStatusBlockedReasonMap(
  statusIds: string[],
  reason: string,
): Record<string, string> {
  const map: Record<string, string> = {}
  for (const statusId of statusIds) {
    map[statusId] = reason
  }
  return map
}

export function mergeStatusBlockedReasonMaps(
  ...maps: Array<Record<string, string> | undefined>
): Record<string, string> {
  const merged: Record<string, string> = {}
  for (const map of maps) {
    if (!map) continue
    for (const [statusId, reason] of Object.entries(map)) {
      if (!merged[statusId]) {
        merged[statusId] = reason
      }
    }
  }
  return merged
}

export function findOverlappingStatusIds(pickupStatusIds: string[], finishStatusIds: string[]) {
  const finishSet = new Set(finishStatusIds)
  return pickupStatusIds.filter((statusId) => finishSet.has(statusId))
}

export function buildDispatcherFinishStatusIds(
  statuses: WorkflowStatusOption[],
  workflows: ReadonlyArray<Pick<WorkflowSummary, 'roleSlug' | 'isActive' | 'pickupStatusIds'>>,
  pickupStatusIds: string[],
) {
  const blocked = new Set(pickupStatusIds)
  const statusById = new Map(statuses.map((status) => [status.id, status]))

  const activeWorkflowTargets = collectDispatcherFinishTargets(
    workflows.flatMap((workflow) => {
      if (!workflow.isActive) return []
      if ((workflow.roleSlug ?? '').trim().toLowerCase() === 'dispatcher') return []
      return workflow.pickupStatusIds
    }),
    statusById,
    blocked,
  )
  if (activeWorkflowTargets.length > 0) {
    return activeWorkflowTargets
  }

  const unstartedFallback = statuses
    .filter((status) => status.stage === 'unstarted' && !blocked.has(status.id))
    .map((status) => status.id)
  if (unstartedFallback.length > 0) {
    return unstartedFallback
  }

  return statuses
    .filter((status) => status.stage === 'started' && !blocked.has(status.id))
    .map((status) => status.id)
}

function collectDispatcherFinishTargets(
  statusIds: string[],
  statusById: Map<string, WorkflowStatusOption>,
  blocked: Set<string>,
) {
  const unstarted: string[] = []
  const started: string[] = []
  const seen = new Set<string>()

  for (const statusId of statusIds) {
    if (blocked.has(statusId) || seen.has(statusId)) continue
    const status = statusById.get(statusId)
    if (!status) continue
    if (status.stage === 'unstarted') {
      unstarted.push(statusId)
      seen.add(statusId)
      continue
    }
    if (status.stage === 'started') {
      started.push(statusId)
      seen.add(statusId)
    }
  }

  return unstarted.length > 0 ? unstarted : started
}

function parseStringList(value: string) {
  const normalized = value.replaceAll(',', '\n')
  const items = normalized
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
  return [...new Set(items)]
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
