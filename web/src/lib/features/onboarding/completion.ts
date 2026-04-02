const onboardingCompletionStoragePrefix = 'openase.project-onboarding.completed'

function storageKey(projectId: string) {
  return `${onboardingCompletionStoragePrefix}.${projectId}`
}

export function readProjectOnboardingCompletion(projectId: string): boolean {
  if (typeof window === 'undefined') {
    return false
  }

  const trimmedProjectId = projectId.trim()
  if (!trimmedProjectId) {
    return false
  }

  try {
    return window.localStorage.getItem(storageKey(trimmedProjectId)) === '1'
  } catch {
    return false
  }
}

export function markProjectOnboardingCompleted(projectId: string) {
  if (typeof window === 'undefined') {
    return
  }

  const trimmedProjectId = projectId.trim()
  if (!trimmedProjectId) {
    return
  }

  try {
    window.localStorage.setItem(storageKey(trimmedProjectId), '1')
  } catch {
    // Ignore localStorage failures.
  }
}
