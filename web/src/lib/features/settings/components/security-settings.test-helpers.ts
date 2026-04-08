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
        redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
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

export function scopedSecrets() {
  return [
    {
      id: 'secret-project-openai',
      organization_id: currentOrg().id,
      project_id: currentProject().id,
      scope: 'project',
      name: 'OPENAI_API_KEY',
      kind: 'opaque',
      description: 'Primary runtime key',
      disabled: false,
      disabled_at: null,
      created_at: '2026-04-08T12:00:00Z',
      updated_at: '2026-04-08T12:30:00Z',
      encryption: {
        algorithm: 'aes-256-gcm',
        key_source: 'database_dsn_sha256',
        key_id: 'database-dsn-sha256:v1',
        value_preview: 'sk-live...cdef',
        rotated_at: '2026-04-08T12:00:00Z',
      },
    },
    {
      id: 'secret-org-gh',
      organization_id: currentOrg().id,
      project_id: null,
      scope: 'organization',
      name: 'GH_TOKEN',
      kind: 'opaque',
      description: 'Org-wide GitHub token',
      disabled: false,
      disabled_at: null,
      created_at: '2026-04-08T11:00:00Z',
      updated_at: '2026-04-08T11:30:00Z',
      encryption: {
        algorithm: 'aes-256-gcm',
        key_source: 'database_dsn_sha256',
        key_id: 'database-dsn-sha256:v1',
        value_preview: 'ghu_test...1234',
        rotated_at: '2026-04-08T11:00:00Z',
      },
    },
  ]
}

export function workflowCatalog() {
  return [
    {
      id: 'workflow-fullstack',
      project_id: currentProject().id,
      agent_id: 'agent-1',
      name: 'Fullstack Developer Workflow',
      type: 'coding',
      harness_path: '.openase/harnesses/fullstack.md',
      is_active: true,
      pickup_status_ids: ['status-todo'],
      finish_status_ids: ['status-preview'],
      created_at: '2026-04-08T10:00:00Z',
      updated_at: '2026-04-08T10:30:00Z',
      version: 1,
      max_concurrent: 1,
      max_retry_attempts: 3,
      timeout_minutes: 60,
      stall_timeout_minutes: 5,
      role_slug: 'fullstack-developer',
      role_name: 'Fullstack Developer',
      role_description: 'Implements end-to-end changes.',
      platform_access_allowed: ['tickets.update.self'],
      hooks: {},
      current_version_id: 'workflow-version-1',
    },
  ]
}

export function ticketCatalog() {
  return [
    {
      id: 'ticket-ase-115',
      project_id: currentProject().id,
      identifier: 'ASE-115',
      title: '[Secrets] Add workflow and ticket secret bindings',
      description: '',
      status_id: 'status-in-progress',
      status_name: 'In Progress',
      archived: false,
      priority: 'high',
      type: 'feature',
      workflow_id: 'workflow-fullstack',
      current_run_id: null,
      target_machine_id: null,
      created_by: 'agent:fullstack-developer-01',
      parent_ticket_id: null,
      external_ref: '',
      budget_usd: 0,
      cost_tokens_input: 0,
      cost_tokens_output: 0,
      cost_tokens_total: 0,
      cost_amount: 0,
      attempt_count: 0,
      consecutive_errors: 0,
      retry_paused: false,
      created_at: '2026-04-08T10:00:00Z',
      started_at: null,
      completed_at: null,
    },
  ]
}

export function scopedSecretBindings() {
  return [
    {
      id: 'binding-1',
      organization_id: currentOrg().id,
      project_id: currentProject().id,
      secret_id: 'secret-project-openai',
      scope: 'workflow',
      scope_resource_id: 'workflow-fullstack',
      binding_key: 'OPENAI_API_KEY',
      created_at: '2026-04-08T12:10:00Z',
      updated_at: '2026-04-08T12:10:00Z',
      secret: {
        id: 'secret-project-openai',
        name: 'OPENAI_API_KEY',
        scope: 'project',
        kind: 'opaque',
        description: 'Primary runtime key',
        project_id: currentProject().id,
        disabled: false,
      },
      target: {
        id: 'workflow-fullstack',
        scope: 'workflow',
        name: 'Fullstack Developer Workflow',
        identifier: '',
      },
    },
  ]
}
