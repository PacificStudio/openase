import { currentOrg, currentProject } from './security-settings.test-helpers'

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
