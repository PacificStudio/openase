import { beforeNavigate } from '$app/navigation'

type SkillEditorNavigationInput = {
  getHasDirtyChanges: () => boolean
  getBusy: () => boolean
  handleSave: () => Promise<void>
}

export function registerSkillEditorNavigationGuard(input: SkillEditorNavigationInput) {
  beforeNavigate(({ cancel }) => {
    if (
      input.getHasDirtyChanges() &&
      !window.confirm('You have unsaved changes. Leave without saving?')
    ) {
      cancel()
    }
  })
}

export function handleSkillEditorKeydown(input: SkillEditorNavigationInput, event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key === 's') {
    event.preventDefault()
    if (!input.getBusy() && input.getHasDirtyChanges()) void input.handleSave()
  }
}

export function handleSkillEditorBeforeUnload(hasDirtyChanges: boolean, event: BeforeUnloadEvent) {
  if (hasDirtyChanges) event.preventDefault()
}
