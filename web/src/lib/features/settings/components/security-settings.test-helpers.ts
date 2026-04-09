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
          : 'OIDC is inactive. Browser access on this machine goes through local bootstrap links until you enable OIDC, and the saved OIDC draft remains available for rollout.',
      recommended_mode:
        authStore.authMode === 'oidc'
          ? 'Use OIDC for multi-user or networked deployments, then keep bootstrap admin emails narrow after first login.'
          : 'Use local bootstrap for personal or recovery access, and enable OIDC when you need managed multi-user browser login.',
      public_exposure_risk: 'local_only',
      warnings: [
        'OIDC is inactive on a loopback-bound instance. Use local bootstrap links for browser access, or enable OIDC before sharing the instance.',
      ],
      next_steps: [
        'Create a local bootstrap link for administrators who still need browser access on this machine.',
        'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for managed multi-user browser login.',
        'If an OIDC rollout locks you out, run `openase auth break-glass disable-oidc` locally before creating a fresh bootstrap link.',
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
            'Plan local bootstrap access, OIDC rollout, and instance_admin bootstrap coverage.',
        },
        {
          title: 'Dual-mode contract',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-dual-mode-contract.md',
          summary:
            'Read the access-control contract, YAML import behavior, and local recovery paths.',
        },
        {
          title: 'IAM rollout checklist',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-admin-console-rollout.md',
          summary:
            'Roll out IAM with validation checks plus a documented break-glass recovery procedure.',
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
      machine_env_vars_redacted: true,
      runtime_secret_responses_redacted: true,
      legacy_providers_requiring_migration: 1,
      legacy_provider_inline_secret_bindings: 2,
      legacy_machines_requiring_migration: 1,
      legacy_machine_secret_env_vars: 1,
      rollout_checklist: [
        {
          key: 'provider-inline-secrets',
          title: 'Migrate inline provider auth_config secrets',
          status: 'pending',
          summary: 'Move legacy inline provider auth_config secrets into scoped secrets.',
        },
        {
          key: 'machine-env-secrets',
          title: 'Migrate machine env var secrets',
          status: 'pending',
          summary: 'Replace secret-like machine env_vars before rollout.',
        },
        {
          key: 'audit-trail',
          title: 'Verify secret activity events',
          status: 'done',
          summary: 'Secret lifecycle events are published to activity.',
        },
      ],
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
        'OIDC is inactive on a loopback-bound instance. Use local bootstrap links for browser access, or enable OIDC before sharing the instance.',
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
