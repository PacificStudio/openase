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

export function configuredOidcAuthState() {
  return {
    authMode: 'oidc' as const,
    authenticated: true,
    issuerURL: 'https://idp.example.com',
    csrfToken: 'csrf-token',
    user: {
      id: 'user-1',
      primaryEmail: 'alice@example.com',
      displayName: 'Alice Control Plane',
    },
    roles: ['instance_admin'],
    permissions: ['org.update'],
  }
}

export function configuredEffectivePermissions(
  scopeKind: 'organization' | 'project',
  scopeID: string,
) {
  return {
    user: {
      id: 'user-1',
      primary_email: 'alice@example.com',
      display_name: 'Alice Control Plane',
    },
    scope: { kind: scopeKind, id: scopeID },
    roles: [scopeKind === 'organization' ? 'org_admin' : 'project_admin'],
    permissions: [scopeKind === 'organization' ? 'org.read' : 'project.read', 'rbac.manage'],
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
