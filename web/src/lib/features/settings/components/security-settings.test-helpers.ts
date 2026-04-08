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
    auth: {
      active_mode: authStore.authMode || 'disabled',
      configured_mode: authStore.authMode || 'disabled',
      issuer_url: authStore.issuerURL || '',
      local_principal: 'local_instance_admin:default',
      mode_summary:
        authStore.authMode === 'oidc'
          ? 'OIDC is active. Browser sessions, RBAC, cached users, memberships, invitations, and auth audit diagnostics are enforced from this control plane.'
          : 'Disabled mode keeps OpenASE in local single-user operation. The current user keeps local highest privilege without browser login or OIDC dependency.',
      recommended_mode:
        authStore.authMode === 'oidc'
          ? 'Use OIDC for multi-user or networked deployments, then keep bootstrap admin emails narrow after first login.'
          : 'Keep disabled mode for personal or local-only use. Move to OIDC + instance_admin when you need real multi-user browser access control.',
      public_exposure_risk: 'local_only',
      warnings: [
        'Disabled mode is appropriate for local-only or single-user use on a loopback-bound instance.',
      ],
      next_steps: [
        'You can keep disabled mode for local single-user use with no extra IAM overhead.',
        'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for multi-user browser login.',
      ],
      config_path: '/home/test/.openase/config.yaml',
      bootstrap_state: {
        status: 'configured',
        admin_emails: ['admin@example.com'],
        summary:
          '1 bootstrap admin email(s) will receive instance_admin on first successful OIDC login.',
      },
      session_policy: {
        session_ttl: '8h0m0s',
        session_idle_ttl: '30m0s',
      },
      last_validation: {
        status: 'ok',
        message:
          'OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.',
        checked_at: '2026-04-07T04:12:00Z',
        issuer_url: 'https://idp.example.com',
        authorization_endpoint: 'https://idp.example.com/authorize',
        token_endpoint: 'https://idp.example.com/token',
        redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
        warnings: [],
      },
      oidc_draft: {
        issuer_url: 'https://idp.example.com',
        client_id: 'openase',
        client_secret_configured: true,
        redirect_mode: 'fixed',
        fixed_redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
        scopes: ['openid', 'profile', 'email', 'groups'],
        allowed_email_domains: ['example.com'],
        bootstrap_admin_emails: ['admin@example.com'],
      },
      docs: [
        {
          title: 'Mode selection guide',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/human-auth-oidc-rbac.md',
          summary:
            'Choose between disabled mode and OIDC, including local-user and instance_admin guidance.',
        },
        {
          title: 'Dual-mode contract',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-dual-mode-contract.md',
          summary:
            'Read the long-term disabled versus OIDC contract and the explicit enable / rollback flow.',
        },
        {
          title: 'IAM rollout checklist',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-admin-console-rollout.md',
          summary:
            'Roll out the full IAM console in stages with migration checks, rollback steps, and validation coverage.',
        },
      ],
    },
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

export function disabledSecurity() {
  return {
    ...configuredSecurity(),
    auth: {
      ...configuredSecurity().auth,
      active_mode: 'disabled',
      configured_mode: 'disabled',
      issuer_url: '',
      warnings: [
        'Disabled mode is appropriate for local-only or single-user use on a loopback-bound instance.',
      ],
    },
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
