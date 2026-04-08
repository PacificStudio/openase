import type { ScopedSecretRecord } from '$lib/api/contracts'

export type ProjectSecretInventory = {
  effective: ScopedSecretRecord[]
  projectOverrides: ScopedSecretRecord[]
  organizationSecrets: ScopedSecretRecord[]
}

export function buildProjectSecretInventory(secrets: ScopedSecretRecord[]): ProjectSecretInventory {
  const organizationSecrets = sortSecrets(secrets.filter((item) => item.scope === 'organization'))
  const projectOverrides = sortSecrets(secrets.filter((item) => item.scope === 'project'))
  const activeProjectOverrideNames = new Set(
    projectOverrides.filter((item) => !item.disabled).map((item) => item.name),
  )

  const effective = sortSecrets(
    secrets.filter((item) => {
      if (item.disabled) {
        return false
      }
      if (item.scope === 'project') {
        return true
      }
      return !activeProjectOverrideNames.has(item.name)
    }),
  )

  return {
    effective,
    projectOverrides,
    organizationSecrets,
  }
}

export function isProjectOverride(
  secret: ScopedSecretRecord,
  organizationSecrets: ScopedSecretRecord[],
) {
  if (secret.scope !== 'project') {
    return false
  }
  return organizationSecrets.some((item) => item.name === secret.name)
}

export function isOverriddenInProject(
  secret: ScopedSecretRecord,
  projectOverrides: ScopedSecretRecord[],
) {
  if (secret.scope !== 'organization') {
    return false
  }
  return projectOverrides.some((item) => item.name === secret.name && !item.disabled)
}

export function formatSecretTimestamp(value?: string | null) {
  if (!value) {
    return 'Never'
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

export function usageIndicator(secret: ScopedSecretRecord) {
  if (secret.usage_count <= 0) {
    return 'Unbound'
  }
  if (secret.usage_count === 1) {
    return '1 binding'
  }
  return `${secret.usage_count} bindings`
}

export function normalizeUsageScopes(secret: ScopedSecretRecord) {
  return [...(secret.usage_scopes ?? [])].sort((left, right) => left.localeCompare(right))
}

function sortSecrets(items: ScopedSecretRecord[]) {
  return [...items].sort(
    (left, right) =>
      left.name.localeCompare(right.name) ||
      left.scope.localeCompare(right.scope) ||
      left.id.localeCompare(right.id),
  )
}
