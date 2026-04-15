const E2E_PROJECT_ID = 'project-e2e'

export function toursAllowed(projectId?: string): boolean {
  if (projectId === E2E_PROJECT_ID) return false
  return typeof navigator === 'undefined' || !navigator.webdriver
}
