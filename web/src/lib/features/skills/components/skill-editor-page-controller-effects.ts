import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat'
import { ApiError } from '$lib/api/client'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { applyInitialSkillLoad } from './skill-editor-page-controller-actions'
import type { SkillEditorPageControllerActionsState } from './skill-editor-page-controller-actions'
import { loadSkillEditorData } from './skill-editor-page.helpers'

type SkillEditorLoadEffectInput = {
  getSkillId: () => string
  getProjectId: () => string | undefined
  setLoading: (value: boolean) => void
  state: SkillEditorPageControllerActionsState
}

type SkillEditorProjectFocusInput = {
  projectId: string | undefined
  loading: boolean
  skill: import('$lib/api/contracts').Skill | null
  selectedFilePath: string | null
  hasDirtyChanges: boolean
}

const projectAIFocusOwner = 'skill-editor-page'

export function createSkillEditorLoadEffect(input: SkillEditorLoadEffectInput) {
  const skillId = input.getSkillId()
  if (!skillId) return

  let cancelled = false
  input.setLoading(true)
  void (async () => {
    try {
      const loaded = await loadSkillEditorData(skillId, input.getProjectId())
      if (!cancelled) applyInitialSkillLoad(input.state, loaded)
    } catch (err) {
      if (!cancelled) {
        toastStore.error(err instanceof ApiError ? err.detail : 'Failed to load skill.')
      }
    } finally {
      if (!cancelled) input.setLoading(false)
    }
  })()

  return () => {
    cancelled = true
  }
}

export function createSkillEditorProjectFocusEffect(input: SkillEditorProjectFocusInput) {
  if (!input.projectId || input.loading || !input.skill) {
    appStore.clearProjectAssistantFocus(projectAIFocusOwner)
    return
  }

  appStore.setProjectAssistantFocus(
    projectAIFocusOwner,
    {
      kind: 'skill',
      projectId: input.projectId,
      skillId: input.skill.id,
      skillName: input.skill.name,
      selectedFilePath: input.selectedFilePath ?? 'SKILL.md',
      boundWorkflowNames: input.skill.bound_workflows.map((workflow) => workflow.name),
      hasDirtyDraft: input.hasDirtyChanges,
    },
    PROJECT_AI_FOCUS_PRIORITY.workspace,
  )

  return () => {
    appStore.clearProjectAssistantFocus(projectAIFocusOwner)
  }
}
