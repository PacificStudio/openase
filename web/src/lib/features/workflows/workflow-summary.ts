import {
  normalizeWorkflowClassification,
  normalizeWorkflowFamily,
  normalizeWorkflowType,
} from './model'
import type { WorkflowSummary } from './types'
import { readWorkflowHooksPayload } from './workflow-hooks'

type WorkflowMetadataFields = {
  role_slug?: string
  role_name?: string
  role_description?: string
  platform_access_allowed?: string[]
}

type WorkflowRecord = {
  id: string
  name: string
  type: string
  workflow_family?: string
  workflow_classification?: {
    family?: string
    confidence?: number
    reasons?: unknown[]
  } | null
  agent_id?: string | null
  harness_path?: string | null
  pickup_status_ids?: string[] | null
  finish_status_ids?: string[] | null
  max_concurrent: number
  max_retry_attempts: number
  timeout_minutes: number
  stall_timeout_minutes?: number | null
  is_active: boolean
  version: number
  hooks?: Record<string, unknown> | null
}

type BuildWorkflowSummaryOptions = {
  resolveStatusName(statusId: string): string
  fallbackMetadata?: {
    roleSlug?: string
    roleName?: string
    roleDescription?: string
    platformAccessAllowed?: string[]
  }
}

function readWorkflowMetadata<T extends object>(workflow: T): T & WorkflowMetadataFields {
  return workflow as T & WorkflowMetadataFields
}

function formatWorkflowStatusLabel(
  statusIds: readonly string[] | null | undefined,
  resolveStatusName: (statusId: string) => string,
) {
  return (statusIds ?? []).map(resolveStatusName).join(', ')
}

export function buildWorkflowSummary(
  workflow: WorkflowRecord,
  options: BuildWorkflowSummaryOptions,
): WorkflowSummary {
  const workflowMeta = readWorkflowMetadata(workflow)
  const pickupStatusIds = workflow.pickup_status_ids ?? []
  const finishStatusIds = workflow.finish_status_ids ?? []

  return {
    id: workflow.id,
    name: workflow.name,
    type: normalizeWorkflowType(workflow.type),
    workflowFamily: normalizeWorkflowFamily(workflow.workflow_family ?? ''),
    classification: normalizeWorkflowClassification(
      workflow.workflow_classification,
      workflow.workflow_family ?? '',
    ),
    agentId: workflow.agent_id ?? null,
    roleSlug: workflowMeta.role_slug ?? options.fallbackMetadata?.roleSlug ?? '',
    roleName: workflowMeta.role_name ?? options.fallbackMetadata?.roleName ?? workflow.name,
    roleDescription:
      workflowMeta.role_description ?? options.fallbackMetadata?.roleDescription ?? '',
    platformAccessAllowed:
      workflowMeta.platform_access_allowed ?? options.fallbackMetadata?.platformAccessAllowed ?? [],
    harnessPath: workflow.harness_path ?? '',
    pickupStatusIds,
    pickupStatusLabel: formatWorkflowStatusLabel(pickupStatusIds, options.resolveStatusName),
    finishStatusIds,
    finishStatusLabel: formatWorkflowStatusLabel(finishStatusIds, options.resolveStatusName),
    maxConcurrent: workflow.max_concurrent,
    maxRetry: workflow.max_retry_attempts,
    timeoutMinutes: workflow.timeout_minutes,
    stallTimeoutMinutes: workflow.stall_timeout_minutes ?? 0,
    isActive: workflow.is_active,
    lastModified: new Date().toISOString(),
    recentSuccessRate: 0,
    version: workflow.version,
    history: [],
    hooks: readWorkflowHooksPayload(workflow.hooks),
    rawHooks: workflow.hooks ?? undefined,
  }
}
