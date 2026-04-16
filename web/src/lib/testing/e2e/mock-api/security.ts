import type {
  MockGitHubCredentialScope,
  MockGitHubProbe,
  MockGitHubSlot,
  MockSecuritySettings,
  MockState,
} from './constants'
import { nowIso } from './constants'
import { clone } from './helpers'

export function resolveSecuritySettings(state: MockState, projectId: string): MockSecuritySettings {
  const existing = state.securitySettingsByProjectId[projectId]
  if (existing) {
    return existing
  }
  const created = createDefaultSecuritySettings(projectId)
  state.securitySettingsByProjectId[projectId] = created
  return created
}

export function createDefaultSecuritySettings(projectId: string): MockSecuritySettings {
  const organization = createEmptyGitHubSlot('organization')
  const projectOverride = createEmptyGitHubSlot('project')
  return {
    project_id: projectId,
    auth: {
      active_mode: 'disabled',
      configured_mode: 'disabled',
      issuer_url: '',
      local_principal: 'local_instance_admin:default',
      mode_summary:
        'OIDC is inactive. Browser access on this machine goes through local bootstrap links until you enable OIDC, and the saved OIDC draft remains available for rollout.',
      recommended_mode:
        'Use local bootstrap for personal or recovery access, and enable OIDC when you need managed multi-user browser login.',
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
        checked_at: nowIso,
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
      effective: clone(organization),
      organization,
      project_override: projectOverride,
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

function createEmptyGitHubSlot(scope: MockGitHubCredentialScope): MockGitHubSlot {
  return {
    scope,
    configured: false,
    source: '',
    token_preview: '',
    probe: createUnconfiguredGitHubProbe(),
  }
}

export function createConfiguredGitHubProbe(login: string): MockGitHubProbe {
  return {
    state: 'valid',
    configured: true,
    valid: true,
    login,
    permissions: ['repo', 'read:org'],
    repo_access: 'granted',
    checked_at: nowIso,
    last_error: '',
  }
}

export function createUnconfiguredGitHubProbe(): MockGitHubProbe {
  return {
    state: 'missing',
    configured: false,
    valid: false,
    permissions: [],
    repo_access: 'not_checked',
    checked_at: undefined,
    last_error: '',
  }
}

export function resolveGitHubCredentialSlot(
  security: MockSecuritySettings,
  scope: MockGitHubCredentialScope,
): MockGitHubSlot {
  return scope === 'organization' ? security.github.organization : security.github.project_override
}

export function syncEffectiveGitHubSlot(security: MockSecuritySettings) {
  security.github.effective = clone(
    security.github.project_override.configured
      ? security.github.project_override
      : security.github.organization,
  )
}

export function parseGitHubCredentialScope(
  raw: string | null | undefined,
): MockGitHubCredentialScope | null {
  return raw === 'organization' || raw === 'project' ? raw : null
}

export function previewToken(token: string): string {
  return token.length <= 8 ? token : `${token.slice(0, 4)}...${token.slice(-4)}`
}
