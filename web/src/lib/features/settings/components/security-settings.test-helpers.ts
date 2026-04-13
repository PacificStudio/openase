import { authStore } from '$lib/stores/auth.svelte'

export function currentProject() {
  return {
    /* i18n-exempt */ id: '9f34ff64-f08b-4a06-b555-f47b34957860',
    /* i18n-exempt */ organization_id: 'org-1',
    /* i18n-exempt */ name: 'Atlas',
    /* i18n-exempt */ slug: 'atlas',
    /* i18n-exempt */ description: '',
    /* i18n-exempt */ status: 'active',
    default_agent_provider_id: null,
    project_ai_platform_access_allowed: ['projects.update', 'projects.add_repo'],
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
  }
}

export function currentOrg() {
  return {
    /* i18n-exempt */ id: 'org-1',
    /* i18n-exempt */ name: 'Acme',
    /* i18n-exempt */ slug: 'acme',
    default_agent_provider_id: '',
    /* i18n-exempt */ status: 'active',
  }
}

export function configuredSecurity() {
  return {
    project_id: currentProject().id,
    auth: {
      active_mode: authStore.authMode || /* i18n-exempt */ 'disabled',
      configured_mode: authStore.authMode || /* i18n-exempt */ 'disabled',
      issuer_url: authStore.issuerURL || '',
      /* i18n-exempt */ local_principal: 'local_instance_admin:default',
      mode_summary:
        authStore.authMode === 'oidc'
          ? /* i18n-exempt */ 'OIDC is active. Browser sessions, RBAC, cached users, memberships, invitations, and auth audit diagnostics are enforced from this control plane.'
          : /* i18n-exempt */ 'OIDC is inactive. Browser access on this machine goes through local bootstrap links until you enable OIDC, and the saved OIDC draft remains available for rollout.',
      recommended_mode:
        authStore.authMode === 'oidc'
          ? /* i18n-exempt */ 'Use OIDC for multi-user or networked deployments, then keep bootstrap admin emails narrow after first login.'
          : /* i18n-exempt */ 'Use local bootstrap for personal or recovery access, and enable OIDC when you need managed multi-user browser login.',
      public_exposure_risk: /* i18n-exempt */ 'local_only',
      warnings: [
        /* i18n-exempt */ 'OIDC is inactive on a loopback-bound instance. Use local bootstrap links for browser access, or enable OIDC before sharing the instance.',
      ],
      next_steps: [
        /* i18n-exempt */ 'Create a local bootstrap link for administrators who still need browser access on this machine.',
        /* i18n-exempt */ 'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for managed multi-user browser login.',
        /* i18n-exempt */ 'If an OIDC rollout locks you out, run `openase auth break-glass disable-oidc` locally before creating a fresh bootstrap link.',
      ],
      /* i18n-exempt */ config_path: '/home/test/.openase/config.yaml',
      bootstrap_state: {
        status: 'configured',
        admin_emails: ['admin@example.com'],
        summary:
          /* i18n-exempt */ '1 bootstrap admin email(s) will receive instance_admin on first successful OIDC login.',
      },
      session_policy: {
        session_ttl: '8h0m0s',
        session_idle_ttl: '30m0s',
      },
      last_validation: {
        status: 'ok',
        message:
          /* i18n-exempt */ 'OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.',
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
          title: /* i18n-exempt */ 'Mode selection guide',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/human-auth-oidc-rbac.md',
          summary:
            /* i18n-exempt */ 'Plan local bootstrap access, OIDC rollout, and instance_admin bootstrap coverage.',
        },
        {
          title: /* i18n-exempt */ 'Dual-mode contract',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-dual-mode-contract.md',
          summary:
            /* i18n-exempt */ 'Read the access-control contract, YAML import behavior, and local recovery paths.',
        },
        {
          title: /* i18n-exempt */ 'IAM rollout checklist',
          href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-admin-console-rollout.md',
          summary:
            /* i18n-exempt */ 'Roll out IAM with validation checks plus a documented break-glass recovery procedure.',
        },
      ],
    },
    agent_tokens: {
      transport: 'Bearer token',
      environment_variable: 'OPENASE_AGENT_TOKEN',
      token_prefix: 'ase_agent_',
      default_scopes: ['tickets.create', 'tickets.list'],
      supported_project_scopes: ['projects.update', 'projects.add_repo'],
      supported_scope_groups: [
        {
          category: 'projects',
          scopes: ['projects.update', 'projects.add_repo'],
        },
      ],
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
          title: /* i18n-exempt */ 'Migrate inline provider auth_config secrets',
          status: 'pending',
          summary:
            /* i18n-exempt */ 'Move legacy inline provider auth_config secrets into scoped secrets.',
        },
        {
          key: 'machine-env-secrets',
          title: /* i18n-exempt */ 'Migrate machine env var secrets',
          status: 'pending',
          summary: /* i18n-exempt */ 'Replace secret-like machine env_vars before rollout.',
        },
        {
          key: 'audit-trail',
          title: /* i18n-exempt */ 'Verify secret activity events',
          status: 'done',
          summary: /* i18n-exempt */ 'Secret lifecycle events are published to activity.',
        },
      ],
    },
    approval_policies: {
      status: 'reserved',
      rules_count: 0,
      summary:
        /* i18n-exempt */ 'Approval policy storage is reserved for future second-factor or approver requirements and stays separate from RBAC grants.',
    },
    deferred: [
      {
        key: 'github-device-flow',
        title: /* i18n-exempt */ 'GitHub Device Flow',
        summary: /* i18n-exempt */ 'Deferred until OAuth app wiring is available.',
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
        /* i18n-exempt */ 'OIDC is inactive on a loopback-bound instance. Use local bootstrap links for browser access, or enable OIDC before sharing the instance.',
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

export function configuredSecurityWithProjectGitHubOverride() {
  const security = configuredSecurity()
  return {
    ...security,
    github: {
      ...security.github,
      effective: {
        ...security.github.organization,
        scope: 'project' as const,
        source: 'gh_cli_import',
      },
      project_override: {
        ...security.github.organization,
        scope: 'project' as const,
        source: 'gh_cli_import',
      },
    },
  }
}
