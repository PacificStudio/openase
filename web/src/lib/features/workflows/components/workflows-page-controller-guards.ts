import { beforeNavigate } from '$app/navigation'

export function createWorkflowsPageBeforeUnloadGuard(isDirty: boolean) {
  if (!isDirty) return

  const handler = (event: BeforeUnloadEvent) => {
    event.preventDefault()
  }
  window.addEventListener('beforeunload', handler)
  return () => window.removeEventListener('beforeunload', handler)
}

export function registerWorkflowsPageNavigationGuard(getIsDirty: () => boolean) {
  beforeNavigate((navigation) => {
    if (getIsDirty() && !confirm('You have unsaved changes. Are you sure you want to leave?')) {
      navigation.cancel()
    }
  })
}
