export const REQUIRED_WORKFLOW_PLATFORM_SCOPE = 'tickets.update.self'
export const REQUIRED_WORKFLOW_SKILL_NAME = 'openase-platform'

export function ensureWorkflowRequiredPlatformScopes(scopes: readonly string[] | null | undefined) {
  const merged = [...(scopes ?? [])]
  if (!merged.includes(REQUIRED_WORKFLOW_PLATFORM_SCOPE)) {
    merged.push(REQUIRED_WORKFLOW_PLATFORM_SCOPE)
  }
  return merged
}

export function isWorkflowRequiredSkillName(name: string) {
  return name.trim() === REQUIRED_WORKFLOW_SKILL_NAME
}
