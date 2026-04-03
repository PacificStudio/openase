import type { BuiltinRole } from '$lib/api/contracts'
import { normalizeWorkflowType } from './model'
import type { WorkflowSummary, WorkflowTemplateDraft } from './types'

type BuiltinRoleMetadata = BuiltinRole & {
  pickup_status_names?: string[]
  finish_status_names?: string[]
  skill_names?: string[]
  platform_access_allowed?: string[]
}

export function buildWorkflowTemplateDraft(role: BuiltinRole): WorkflowTemplateDraft {
  const roleMeta = role as BuiltinRoleMetadata

  return {
    name: role.name,
    content: role.workflow_content || role.content,
    workflowType: normalizeWorkflowType(role.workflow_type),
    workflowFamily:
      typeof role.workflow_family === 'string'
        ? (role.workflow_family as WorkflowTemplateDraft['workflowFamily'])
        : undefined,
    roleSlug: role.slug,
    roleName: role.name,
    roleDescription: role.summary,
    platformAccessAllowed: roleMeta.platform_access_allowed ?? [],
    skillNames: roleMeta.skill_names ?? [],
    pickupStatusNames: roleMeta.pickup_status_names ?? [],
    finishStatusNames: roleMeta.finish_status_names ?? [],
    harnessPath: role.harness_path,
  }
}

export function applyWorkflowVersionRefresh(
  workflows: WorkflowSummary[],
  selectedId: string,
  update: {
    harnessPath?: string | null
    version?: number
    history: WorkflowSummary['history']
  },
) {
  return workflows.map((workflow) =>
    workflow.id === selectedId
      ? {
          ...workflow,
          harnessPath: update.harnessPath ?? workflow.harnessPath,
          version: update.version ?? workflow.version,
          history: update.history,
        }
      : workflow,
  )
}
