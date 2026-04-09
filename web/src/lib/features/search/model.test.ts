import { describe, expect, it } from 'vitest'

import type { Agent, Organization, Project, Ticket, Workflow } from '$lib/api/contracts'
import { buildSearchIndex, groupSearchItems } from './model'

const organizationFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  status: 'active',
  default_agent_provider_id: null,
}

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: 'Automation platform',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const ticketFixture: Ticket = {
  id: 'ticket-1',
  project_id: 'project-1',
  identifier: 'ASE-178',
  title: 'Fix global search',
  description: '',
  status_id: 'status-1',
  status_name: 'In Progress',
  priority: 'high',
  type: 'feature',
  archived: false,
  workflow_id: 'workflow-1',
  current_run_id: null,
  target_machine_id: null,
  created_by: 'codex',
  parent: null,
  children: [],
  dependencies: [],
  external_links: [],
  pull_request_urls: [],
  external_ref: '',
  budget_usd: 0,
  cost_tokens_input: 0,
  cost_tokens_output: 0,
  cost_tokens_total: 0,
  cost_amount: 0,
  attempt_count: 0,
  consecutive_errors: 0,
  started_at: null,
  completed_at: null,
  next_retry_at: null,
  retry_paused: false,
  pause_reason: '',
  created_at: '2026-03-21T00:00:00Z',
}

const workflowFixture: Workflow = {
  id: 'workflow-1',
  project_id: 'project-1',
  agent_id: 'agent-1',
  name: 'Coding',
  type: 'coding',
  workflow_family: 'coding',
  workflow_classification: {
    family: 'coding',
    confidence: 1,
    reasons: ['fixture'],
  },
  harness_path: '.openase/harnesses/coding.md',
  harness_content: null,
  hooks: {},
  max_concurrent: 1,
  max_retry_attempts: 0,
  timeout_minutes: 30,
  stall_timeout_minutes: 10,
  version: 1,
  is_active: true,
  pickup_status_ids: ['status-1'],
  finish_status_ids: ['status-2'],
}

const agentFixture: Agent = {
  id: 'agent-1',
  provider_id: 'provider-1',
  project_id: 'project-1',
  name: 'Codex Worker',
  runtime_control_state: 'active',
  runtime: {
    active_run_count: 1,
    status: 'running',
    current_run_id: null,
    current_ticket_id: 'ticket-1',
    session_id: 'session-1',
    runtime_phase: 'ready',
    runtime_started_at: null,
    last_error: '',
    last_heartbeat_at: null,
    current_step_status: null,
    current_step_summary: null,
    current_step_changed_at: null,
  },
  total_tokens_used: 0,
  total_tickets_completed: 0,
}

function buildFixtureSearchIndex() {
  return buildSearchIndex({
    organizations: [organizationFixture],
    projects: [projectFixture],
    currentOrg: organizationFixture,
    currentProject: projectFixture,
    currentSection: 'tickets',
    tickets: [ticketFixture],
    workflows: [workflowFixture],
    agents: [agentFixture],
    newTicketEnabled: true,
  })
}

describe('global search index', () => {
  it('builds project navigation and workspace command entries', () => {
    const items = buildFixtureSearchIndex()

    expect(items.some((item) => item.id === 'command-open-project-ai')).toBe(true)
    expect(items.some((item) => item.id === 'command-new-ticket')).toBe(true)
    expect(items.some((item) => item.id === 'command-toggle-theme')).toBe(true)
    expect(items.some((item) => item.id === 'page-tickets' && item.badge === 'Current')).toBe(true)
    expect(items.some((item) => item.id === 'page-scheduled-jobs')).toBe(true)
    expect(items.some((item) => item.id === 'project-project-1')).toBe(true)
    expect(items.some((item) => item.id === 'organization-org-1')).toBe(true)
  })

  it('builds ticket, workflow, and agent entries from current project data', () => {
    const items = buildFixtureSearchIndex()

    expect(items.some((item) => item.id === 'ticket-ticket-1')).toBe(true)
    expect(items.some((item) => item.id === 'workflow-workflow-1')).toBe(true)
    expect(items.some((item) => item.id === 'agent-agent-1')).toBe(true)
  })

  it('groups entries in command palette order and omits empty groups', () => {
    const groups = groupSearchItems([
      {
        id: 'command-toggle-theme',
        group: 'Commands',
        kind: 'command',
        title: 'Toggle Theme',
        subtitle: 'Switch theme',
        badge: 'Command',
        searchText: 'Toggle Theme Switch theme Command',
        action: { kind: 'toggle_theme' },
      },
      {
        id: 'command-open-project-ai',
        group: 'Commands',
        kind: 'command',
        title: 'Ask AI',
        subtitle: 'Open project AI.',
        badge: 'Command',
        searchText: 'Ask AI Open project AI. Command',
        action: { kind: 'open_project_ai' },
      },
      {
        id: 'organization-org-1',
        group: 'Organizations',
        kind: 'organization',
        title: 'Acme',
        subtitle: 'Open Acme overview.',
        badge: 'Org',
        searchText: 'Acme Open Acme overview. Org',
        action: { kind: 'navigate', href: '/orgs/org-1' },
      },
    ])

    expect(groups).toHaveLength(2)
    expect(groups[0]?.heading).toBe('Commands')
    expect(groups[1]?.heading).toBe('Organizations')
  })
})
