import { ApiError } from '$lib/api/client'
import type { Skill, Workflow } from '$lib/api/contracts'
import { bindSkill, deleteSkill, disableSkill, enableSkill, unbindSkill } from '$lib/api/openase'
import { toastStore } from '$lib/stores/toast.svelte'

type SkillMutationInput = {
  getSkill: () => Skill | null
  setSkill: (value: Skill) => void
  getWorkflows: () => Workflow[]
  setBusy: (value: boolean) => void
  navigateBack: () => void
}

export async function handleToggleSkillEnabled(input: SkillMutationInput) {
  const skill = input.getSkill()
  if (!skill) return

  input.setBusy(true)
  try {
    input.setSkill(
      skill.is_enabled ? (await disableSkill(skill.id)).skill : (await enableSkill(skill.id)).skill,
    )
    const nextSkill = input.getSkill()
    if (!nextSkill) return
    toastStore.success(`${nextSkill.is_enabled ? 'Enabled' : 'Disabled'} ${nextSkill.name}.`)
  } catch (err) {
    toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill state.')
  } finally {
    input.setBusy(false)
  }
}

export async function handleDeleteSkill(input: SkillMutationInput) {
  const skill = input.getSkill()
  if (!skill || !window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)) return

  input.setBusy(true)
  try {
    await deleteSkill(skill.id)
    toastStore.success(`Deleted ${skill.name}.`)
    input.navigateBack()
  } catch (err) {
    toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete skill.')
  } finally {
    input.setBusy(false)
  }
}

export async function handleWorkflowBindingMutation(
  input: SkillMutationInput,
  workflowId: string,
  shouldBind: boolean,
) {
  const skill = input.getSkill()
  if (!skill) return

  input.setBusy(true)
  try {
    input.setSkill(
      shouldBind
        ? (await bindSkill(skill.id, [workflowId])).skill
        : (await unbindSkill(skill.id, [workflowId])).skill,
    )
    const nextSkill = input.getSkill()
    if (!nextSkill) return
    const workflowName =
      input.getWorkflows().find((workflow) => workflow.id === workflowId)?.name ?? 'workflow'
    toastStore.success(
      `${shouldBind ? 'Bound' : 'Unbound'} ${nextSkill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`,
    )
  } catch (err) {
    toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill binding.')
  } finally {
    input.setBusy(false)
  }
}
