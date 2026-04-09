import { authStore } from '$lib/stores/auth.svelte'
import { currentOrg } from './security-settings.test-helpers'

type MockScope = 'instance' | 'organization' | 'project'

export function oidcUser() {
  return {
    id: 'user-1',
    primaryEmail: 'alice@example.com',
    displayName: 'Alice Control Plane',
  }
}

export function hydrateOidcAuth() {
  authStore.hydrate({
    authMode: 'oidc',
    loginRequired: true,
    authenticated: true,
    principalKind: 'human_session',
    authConfigured: true,
    sessionGovernanceAvailable: true,
    canManageAuth: true,
    issuerURL: 'https://idp.example.com',
    csrfToken: 'csrf-token',
    user: oidcUser(),
    roles: ['instance_admin'],
    permissions: ['org.update'],
  })
}

export function effectivePermissionsMock(scope: MockScope, id = '') {
  return {
    user: {
      id: oidcUser().id,
      primary_email: oidcUser().primaryEmail,
      display_name: oidcUser().displayName,
    },
    scope: { kind: scope, id },
    roles:
      scope === 'instance'
        ? ['instance_admin']
        : scope === 'organization'
          ? ['org_admin']
          : ['project_admin'],
    permissions:
      scope === 'instance'
        ? ['rbac.manage', 'security_setting.read', 'security_setting.update']
        : [`${scope}.read`, 'rbac.manage'],
    groups: [{ group_key: 'platform-admins', group_name: 'Platform Admins', issuer: 'oidc' }],
  }
}

export function configuredSessionGovernance() {
  return {
    authMode: 'oidc' as const,
    currentSessionID: 'session-current',
    sessions: [
      {
        id: 'session-current',
        current: true,
        device: { kind: 'desktop', os: 'Linux', browser: 'Firefox', label: 'Firefox on Linux' },
        createdAt: '2026-04-04T10:00:00Z',
        lastActiveAt: '2026-04-04T10:30:00Z',
        expiresAt: '2026-04-04T18:00:00Z',
        idleExpiresAt: '2026-04-04T11:00:00Z',
      },
    ],
    auditEvents: [
      {
        id: 'audit-1',
        eventType: 'login.success',
        actorID: 'user:user-1',
        message: 'Signed in via OIDC.',
        metadata: {},
        createdAt: '2026-04-04T10:00:00Z',
      },
    ],
    stepUp: {
      status: 'reserved',
      summary: 'Reserved for future high-risk actions.',
      supportedMethods: [],
    },
  }
}

export async function mockEffectivePermissionsByScope({
  orgId,
  projectId,
}: {
  orgId?: string
  projectId?: string
}) {
  if (!orgId && !projectId) {
    return effectivePermissionsMock('instance')
  }
  if (orgId) {
    return effectivePermissionsMock('organization', orgId)
  }
  return effectivePermissionsMock('project', projectId ?? '')
}

export function organizationGroupBinding() {
  return {
    id: 'binding-1',
    scopeKind: 'organization',
    scopeID: currentOrg().id,
    subjectKind: 'group',
    subjectKey: 'platform-admins',
    roleKey: 'org_admin',
    grantedBy: 'user:user-1',
    createdAt: '2026-04-04T09:00:00Z',
  }
}

export function createdOrganizationUserBinding() {
  return {
    id: 'binding-2',
    scopeKind: 'organization',
    scopeID: currentOrg().id,
    subjectKind: 'user',
    subjectKey: 'bob@example.com',
    roleKey: 'org_member',
    grantedBy: 'user:user-1',
    createdAt: '2026-04-04T10:00:00Z',
  }
}
