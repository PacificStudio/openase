import type { WorkflowSummary } from '../types'
import type { WorkflowLifecycleDraft } from '../workflow-lifecycle'
import { workflowHooksDraftSignature, type WorkflowHooksDraft } from '../workflow-hooks'

export function workflowLifecycleDraftKey(workflow: WorkflowSummary): string {
  return [
    workflow.id,
    workflow.version,
    workflow.agentId ?? '',
    workflow.name,
    workflow.type,
    workflow.roleName ?? '',
    workflow.roleDescription ?? '',
    (workflow.platformAccessAllowed ?? []).join(','),
    workflow.isActive,
    workflow.pickupStatusIds.join(','),
    workflow.finishStatusIds.join(','),
    workflow.maxConcurrent,
    workflow.maxRetry,
    workflow.timeoutMinutes,
    workflow.stallTimeoutMinutes,
    JSON.stringify(workflow.rawHooks ?? workflow.hooks ?? {}),
  ].join(':')
}

export function isWorkflowLifecycleDraftDirty(
  draft: WorkflowLifecycleDraft,
  baseDraft: WorkflowLifecycleDraft,
  hookDraft: WorkflowHooksDraft,
  baseHookDraft: WorkflowHooksDraft,
): boolean {
  return (
    draft.agentId !== baseDraft.agentId ||
    draft.name !== baseDraft.name ||
    draft.typeLabel !== baseDraft.typeLabel ||
    draft.roleName !== baseDraft.roleName ||
    draft.roleDescription !== baseDraft.roleDescription ||
    draft.platformAccessAllowed !== baseDraft.platformAccessAllowed ||
    draft.pickupStatusIds.join(':') !== baseDraft.pickupStatusIds.join(':') ||
    draft.finishStatusIds.join(':') !== baseDraft.finishStatusIds.join(':') ||
    draft.maxConcurrent !== baseDraft.maxConcurrent ||
    draft.maxRetryAttempts !== baseDraft.maxRetryAttempts ||
    draft.timeoutMinutes !== baseDraft.timeoutMinutes ||
    draft.stallTimeoutMinutes !== baseDraft.stallTimeoutMinutes ||
    draft.isActive !== baseDraft.isActive ||
    workflowHooksDraftSignature(hookDraft) !== workflowHooksDraftSignature(baseHookDraft)
  )
}
