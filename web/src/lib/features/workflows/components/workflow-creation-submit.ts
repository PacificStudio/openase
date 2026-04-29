import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
import {
  parseWorkflowHooksDraft,
  type WorkflowHooksDraft,
  type WorkflowHooksPayload,
} from '../workflow-hooks'
import { findOverlappingStatusIds } from '../workflow-lifecycle'
import type { WorkflowTemplateDraft } from '../types'

type T = (key: TranslationKey, params?: TranslationParams) => string

export type WorkflowCreationValidationInput = {
  projectId: string
  name: string
  typeLabel: string
  agentId: string
  templateStatusError: string
  pickupStatusIds: string[]
  finishStatusIds: string[]
  hookDraft: WorkflowHooksDraft
}

export type WorkflowCreationValidationResult =
  | { ok: true; hooks: WorkflowHooksPayload | undefined }
  | { ok: false; error: string; openAdvanced?: boolean }

export function validateWorkflowCreationInputs(
  input: WorkflowCreationValidationInput,
  t: T,
): WorkflowCreationValidationResult {
  if (!input.projectId) {
    return { ok: false, error: t('workflows.creation.dialog.errors.noProject') }
  }
  if (!input.name.trim()) {
    return { ok: false, error: t('workflows.creation.dialog.errors.nameRequired') }
  }
  if (!input.agentId) {
    return { ok: false, error: t('workflows.creation.dialog.errors.agentRequired') }
  }
  if (!input.typeLabel.trim()) {
    return { ok: false, error: t('workflows.creation.dialog.errors.typeRequired') }
  }
  if (input.templateStatusError) {
    return { ok: false, error: input.templateStatusError }
  }
  if (input.pickupStatusIds.length === 0 || input.finishStatusIds.length === 0) {
    return { ok: false, error: t('workflows.creation.dialog.errors.statusRequired') }
  }
  if (findOverlappingStatusIds(input.pickupStatusIds, input.finishStatusIds).length > 0) {
    return { ok: false, error: t('workflows.creation.dialog.errors.statusExclusive') }
  }

  const parsedHooks = parseWorkflowHooksDraft(input.hookDraft)
  if (!parsedHooks.ok) {
    return { ok: false, error: parsedHooks.error, openAdvanced: true }
  }

  return { ok: true, hooks: parsedHooks.value }
}

export function buildWorkflowCreationPayload(args: {
  agentId: string
  name: string
  typeLabel: string
  pickupStatusIds: string[]
  finishStatusIds: string[]
  hooks: WorkflowHooksPayload | undefined
  templateDraft: WorkflowTemplateDraft | null
}) {
  const trimmedName = args.name.trim()
  return {
    agentId: args.agentId,
    name: trimmedName,
    workflowType: args.typeLabel.trim(),
    roleSlug: args.templateDraft?.roleSlug ?? '',
    roleName: args.templateDraft?.roleName ?? trimmedName,
    roleDescription: args.templateDraft?.roleDescription ?? '',
    platformAccessAllowed: args.templateDraft?.platformAccessAllowed ?? [],
    skillNames: args.templateDraft?.skillNames ?? [],
    harnessPath: args.templateDraft?.harnessPath ?? null,
    pickupStatusIds: args.pickupStatusIds,
    finishStatusIds: args.finishStatusIds,
    hooks: args.hooks,
  }
}
