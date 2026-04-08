import { describe, expect, it } from 'vitest'

import type { ScopedSecretRecord } from '$lib/api/contracts'
import {
  buildProjectSecretInventory,
  isOverriddenInProject,
  isProjectOverride,
} from './scoped-secrets'

function secret(overrides: Partial<ScopedSecretRecord>): ScopedSecretRecord {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    organization_id: overrides.organization_id ?? 'org-1',
    project_id: overrides.project_id ?? null,
    scope: overrides.scope ?? 'organization',
    name: overrides.name ?? 'OPENAI_API_KEY',
    kind: overrides.kind ?? 'opaque',
    description: overrides.description ?? '',
    disabled: overrides.disabled ?? false,
    disabled_at: overrides.disabled_at ?? null,
    created_at: overrides.created_at ?? '2026-04-08T10:00:00Z',
    updated_at: overrides.updated_at ?? '2026-04-08T10:00:00Z',
    usage_count: overrides.usage_count ?? 1,
    usage_scopes: overrides.usage_scopes ?? ['organization'],
    encryption: overrides.encryption ?? {
      algorithm: 'aes-256-gcm',
      key_id: 'database-dsn-sha256:v1',
      key_source: 'database_dsn_sha256',
      rotated_at: '2026-04-08T10:00:00Z',
      value_preview: 'sk-live...1234',
    },
  }
}

describe('scoped secret inventory helpers', () => {
  it('promotes active project overrides into the effective inventory', () => {
    const organizationSecret = secret({
      id: 'org-secret',
      scope: 'organization',
      name: 'OPENAI_API_KEY',
    })
    const projectSecret = secret({
      id: 'project-secret',
      scope: 'project',
      project_id: 'project-1',
      name: 'OPENAI_API_KEY',
      usage_scopes: ['project'],
    })

    const inventory = buildProjectSecretInventory([organizationSecret, projectSecret])

    expect(inventory.effective.map((item) => item.id)).toEqual(['project-secret'])
    expect(isProjectOverride(projectSecret, inventory.organizationSecrets)).toBe(true)
    expect(isOverriddenInProject(organizationSecret, inventory.projectOverrides)).toBe(true)
  })

  it('falls back to the inherited organization secret when the project override is disabled', () => {
    const organizationSecret = secret({ id: 'org-secret', scope: 'organization', name: 'GH_TOKEN' })
    const disabledProjectSecret = secret({
      id: 'project-secret',
      scope: 'project',
      project_id: 'project-1',
      name: 'GH_TOKEN',
      disabled: true,
      disabled_at: '2026-04-08T11:00:00Z',
      usage_scopes: ['project'],
    })

    const inventory = buildProjectSecretInventory([organizationSecret, disabledProjectSecret])

    expect(inventory.effective.map((item) => item.id)).toEqual(['org-secret'])
    expect(isOverriddenInProject(organizationSecret, inventory.projectOverrides)).toBe(false)
  })
})
