import { authStore } from '$lib/stores/auth.svelte'

export function currentProject() {
  return {
    id: '9f34ff64-f08b-4a06-b555-f47b34957860',
    organization_id: 'org-1',
    name: 'Atlas',
    slug: 'atlas',
    description: '',
    status: 'active',
    default_agent_provider_id: null,
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
  }
}

export function currentOrg() {
  return {
    id: 'org-1',
    name: 'Acme',
    slug: 'acme',
    default_agent_provider_id: '',
    status: 'active',
  }
}

export function configuredSecurity() {
  return {
    project_id: currentProject().id,
    agent_tokens: {
      transport: 'Bearer token',
      environment_variable: 'OPENASE_AGENT_TOKEN',
      token_prefix: 'ase_agent_',
      default_scopes: ['tickets.create', 'tickets.list'],
      supported_project_scopes: ['projects.update', 'projects.add_repo'],
    },
    github: {
      effective: {
        scope: 'organization',
        configured: true,
        source: 'gh_cli_import',
        token_preview: 'ghu_test...1234',
        probe: {
          state: 'valid',
          configured: true,
          valid: true,
          login: 'octocat',
          permissions: ['repo', 'read:org'],
          repo_access: 'granted',
          checked_at: '2026-03-28T12:00:00Z',
          last_error: '',
        },
      },
      organization: {
        scope: 'organization',
        configured: true,
        source: 'gh_cli_import',
        token_preview: 'ghu_test...1234',
        probe: {
          state: 'valid',
          configured: true,
          valid: true,
          login: 'octocat',
          permissions: ['repo', 'read:org'],
          repo_access: 'granted',
          checked_at: '2026-03-28T12:00:00Z',
          last_error: '',
        },
      },
      project_override: {
        scope: 'project',
        configured: false,
        source: '',
        token_preview: '',
        probe: {
          state: 'missing',
          configured: false,
          valid: false,
          permissions: [],
          repo_access: 'not_checked',
          checked_at: undefined,
          last_error: '',
        },
      },
    },
    webhooks: {
      connector_endpoint: 'POST /api/v1/webhooks/:connector/:provider',
    },
    secret_hygiene: {
      notification_channel_configs_redacted: true,
    },
    approval_policies: {
      status: 'reserved',
      rules_count: 0,
      summary:
        'Approval policy storage is reserved for future second-factor or approver requirements and stays separate from RBAC grants.',
    },
    deferred: [
      {
        key: 'github-device-flow',
        title: 'GitHub Device Flow',
        summary: 'Deferred until OAuth app wiring is available.',
      },
    ],
  }
}

export function configuredSecurityWithNullPermissions() {
  const security = configuredSecurity()
  return {
    ...security,
    github: {
      ...security.github,
      effective: {
        ...security.github.effective,
        probe: {
          ...security.github.effective.probe,
          permissions: null,
        },
      },
      organization: {
        ...security.github.organization,
        probe: {
          ...security.github.organization.probe,
          permissions: null,
        },
      },
      project_override: {
        ...security.github.project_override,
        probe: {
          ...security.github.project_override.probe,
          permissions: null,
        },
      },
    },
  }
}

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
    authenticated: true,
    issuerURL: 'https://idp.example.com',
    csrfToken: 'csrf-token',
    user: oidcUser(),
    roles: ['instance_admin'],
    permissions: ['org.update'],
  })
}

type MockScope = 'instance' | 'organization' | 'project'

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
    permissions: scope === 'instance' ? ['rbac.manage'] : [`${scope}.read`, 'rbac.manage'],
    groups: [{ group_key: 'platform-admins', group_name: 'Platform Admins', issuer: 'oidc' }],
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
