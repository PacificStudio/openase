export const ORG_ID = 'org-e2e'
export const PROJECT_ID = 'project-e2e'
export const LOCAL_MACHINE_ID = 'machine-local'
export const GPU_MACHINE_ID = 'machine-gpu'
export const DEFAULT_PROVIDER_ID = '1c7cae12-cafc-4359-90ed-5ab8a8574c63'
export const CLAUDE_PROVIDER_ID = '22b906e4-d906-4b21-908a-9bf1e28d075a'
export const GEMINI_PROVIDER_ID = 'ed7e5e5e-7d06-4685-8de6-45b502d7d393'
export const OPENAI_PROVIDER_ID = '92619809-3a75-42a7-83e6-a6f93f6b3a6b'
export const DEFAULT_AGENT_ID = 'agent-coder'
export const DEFAULT_WORKFLOW_ID = 'workflow-coding'
export const DEFAULT_REPO_ID = 'repo-todo'
export const DEFAULT_JOB_ID = 'job-nightly'
export const DEFAULT_TICKET_ID = 'ticket-1'
export const DEFAULT_STATUS_IDS = {
  todo: 'status-todo',
  review: 'status-review',
  done: 'status-done',
} as const

export type JsonValue = unknown
export type MockGitHubCredentialScope = 'organization' | 'project'

export type MockGitHubProbe = {
  state: string
  configured: boolean
  valid: boolean
  login?: string
  permissions: string[]
  repo_access: string
  checked_at?: string
  last_error: string
}

export type MockGitHubSlot = {
  scope: MockGitHubCredentialScope
  configured: boolean
  source: string
  token_preview: string
  probe: MockGitHubProbe
}

export type MockSecuritySettings = {
  project_id: string
  auth: {
    active_mode: string
    configured_mode: string
    issuer_url: string
    local_principal: string
    mode_summary: string
    recommended_mode: string
    public_exposure_risk: string
    warnings: string[]
    next_steps: string[]
    config_path: string
    bootstrap_state: {
      status: string
      admin_emails: string[]
      summary: string
    }
    session_policy: {
      session_ttl: string
      session_idle_ttl: string
    }
    last_validation: {
      status: string
      message: string
      checked_at: string
      issuer_url: string
      authorization_endpoint: string
      token_endpoint: string
      redirect_url: string
      warnings: string[]
    }
    oidc_draft: {
      issuer_url: string
      client_id: string
      client_secret_configured: boolean
      redirect_mode: string
      fixed_redirect_url: string
      scopes: string[]
      allowed_email_domains: string[]
      bootstrap_admin_emails: string[]
    }
    docs: Array<{
      title: string
      href: string
      summary: string
    }>
  }
  agent_tokens: {
    transport: string
    environment_variable: string
    token_prefix: string
    default_scopes: string[]
    supported_project_scopes: string[]
  }
  github: {
    effective: MockGitHubSlot
    organization: MockGitHubSlot
    project_override: MockGitHubSlot
  }
  webhooks: {
    connector_endpoint: string
  }
  secret_hygiene: {
    notification_channel_configs_redacted: boolean
  }
  approval_policies: {
    status: string
    rules_count: number
    summary: string
  }
  deferred: Array<{
    key: string
    title: string
    summary: string
  }>
}

export type MockState = {
  organizations: Record<string, unknown>[]
  projects: Record<string, unknown>[]
  machines: Record<string, unknown>[]
  providers: Record<string, unknown>[]
  agents: Record<string, unknown>[]
  agentRuns: Record<string, unknown>[]
  activityEvents: Record<string, unknown>[]
  projectUpdates: Record<string, unknown>[]
  tickets: Record<string, unknown>[]
  statuses: Record<string, unknown>[]
  repos: Record<string, unknown>[]
  workflows: Record<string, unknown>[]
  harnessByWorkflowId: Record<
    string,
    {
      content: string
      path: string
      version: number
      history: Array<{ id: string; version: number; created_by: string; created_at: string }>
    }
  >
  scheduledJobs: Record<string, unknown>[]
  projectConversations: Record<string, unknown>[]
  projectConversationEntries: Record<string, unknown>[]
  skills: Record<string, unknown>[]
  builtinRoles: Record<string, unknown>[]
  securitySettingsByProjectId: Record<string, MockSecuritySettings>
  harnessVariables: { groups: Record<string, unknown>[] }
  counters: {
    machine: number
    repo: number
    workflow: number
    agent: number
    skill: number
    scheduledJob: number
    projectUpdateThread: number
    projectUpdateComment: number
    projectConversation: number
    projectConversationEntry: number
    projectConversationTurn: number
  }
}

export const nowIso = '2026-03-27T10:00:00.000Z'
export const encoder = new TextEncoder()
