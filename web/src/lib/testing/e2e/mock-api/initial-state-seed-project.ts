import {
  DEFAULT_AGENT_ID,
  DEFAULT_JOB_ID,
  DEFAULT_REPO_ID,
  DEFAULT_STATUS_IDS,
  DEFAULT_TICKET_ID,
  DEFAULT_WORKFLOW_ID,
  PROJECT_ID,
  nowIso,
} from './constants'
import { defaultHarnessContent } from './entities'
import { createMockTicketRecord } from './ticket-data'

export function createSeedTickets() {
  return [
    createMockTicketRecord({
      id: DEFAULT_TICKET_ID,
      identifier: 'ASE-101',
      title: 'Improve machine management UX',
      statusId: DEFAULT_STATUS_IDS.todo,
      statusName: 'Todo',
      workflowId: DEFAULT_WORKFLOW_ID,
    }),
  ]
}

export function createSeedStatuses() {
  return [
    {
      id: DEFAULT_STATUS_IDS.todo,
      project_id: PROJECT_ID,
      name: 'Todo',
      stage: 'unstarted',
      color: '#2563eb',
      icon: '',
      position: 1,
      active_runs: 1,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.review,
      project_id: PROJECT_ID,
      name: 'In Review',
      stage: 'started',
      color: '#f59e0b',
      icon: '',
      position: 2,
      active_runs: 0,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.done,
      project_id: PROJECT_ID,
      name: 'Done',
      stage: 'completed',
      color: '#16a34a',
      icon: '',
      position: 3,
      active_runs: 0,
      max_active_runs: null,
    },
  ]
}

export function createSeedRepos() {
  return [
    {
      id: DEFAULT_REPO_ID,
      project_id: PROJECT_ID,
      name: 'TodoApp',
      repository_url: 'https://github.com/BetterAndBetterII/TodoApp.git',
      default_branch: 'main',
      workspace_dirname: 'TodoApp',
      labels: [],
    },
  ]
}

export function createSeedWorkflows() {
  return [
    {
      id: DEFAULT_WORKFLOW_ID,
      project_id: PROJECT_ID,
      name: 'Coding Workflow',
      type: 'coding',
      agent_id: DEFAULT_AGENT_ID,
      pickup_status_ids: [DEFAULT_STATUS_IDS.todo],
      finish_status_ids: [DEFAULT_STATUS_IDS.review],
      max_concurrent: 0,
      max_retry_attempts: 1,
      timeout_minutes: 30,
      stall_timeout_minutes: 5,
      is_active: true,
      harness_path: '.openase/harnesses/coding-workflow.md',
      version: 3,
    },
  ]
}

export function createSeedHarnessByWorkflowId() {
  return {
    [DEFAULT_WORKFLOW_ID]: {
      content: defaultHarnessContent('Coding Workflow'),
      path: '.openase/harnesses/coding-workflow.md',
      version: 3,
      history: [
        {
          id: `${DEFAULT_WORKFLOW_ID}-v3`,
          version: 3,
          created_by: 'user:manual',
          created_at: nowIso,
        },
        {
          id: `${DEFAULT_WORKFLOW_ID}-v2`,
          version: 2,
          created_by: 'user:manual',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
  }
}

export function createSeedScheduledJobs() {
  return [
    {
      id: DEFAULT_JOB_ID,
      project_id: PROJECT_ID,
      workflow_id: DEFAULT_WORKFLOW_ID,
      name: 'Nightly regression sweep',
      cron_expression: '0 2 * * *',
      is_enabled: true,
      ticket_template: {
        title: 'Run nightly regression sweep',
        description: 'Verify the core project workflows still behave as expected.',
        priority: 'medium',
        type: 'feature',
        budget_usd: 12,
        created_by: 'scheduler',
      },
      last_run_at: '2026-03-26T02:00:00.000Z',
      next_run_at: '2026-03-28T02:00:00.000Z',
    },
  ]
}

export function createSeedSkills() {
  return [
    {
      id: 'skill-commit',
      project_id: PROJECT_ID,
      name: 'commit',
      description: 'Create a well-formed git commit.',
      path: '/skills/commit',
      current_version: 2,
      is_builtin: true,
      is_enabled: true,
      created_by: 'system:init',
      created_at: nowIso,
      bound_workflows: [{ id: DEFAULT_WORKFLOW_ID }],
      content: '# Commit\n\nCreate a well-formed git commit.',
      history: [
        { id: 'skill-commit-v2', version: 2, created_by: 'user:manual', created_at: nowIso },
        {
          id: 'skill-commit-v1',
          version: 1,
          created_by: 'system:init',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
    {
      id: 'skill-deploy-openase',
      project_id: PROJECT_ID,
      name: 'deploy-openase',
      description: 'Build and redeploy OpenASE locally.',
      path: '/skills/deploy-openase',
      current_version: 1,
      is_builtin: false,
      is_enabled: true,
      created_by: 'user:manual',
      created_at: nowIso,
      bound_workflows: [],
      content: '# Deploy OpenASE\n\nBuild and redeploy OpenASE locally.',
      history: [
        {
          id: 'skill-deploy-openase-v1',
          version: 1,
          created_by: 'user:manual',
          created_at: nowIso,
        },
      ],
    },
  ]
}

export function createSeedBuiltinRoles() {
  return [
    {
      id: 'builtin-coding',
      workflow_type: 'coding',
      name: 'coding',
      content: defaultHarnessContent('Coding Workflow'),
    },
  ]
}

export function createSeedHarnessVariables() {
  return {
    groups: [
      {
        name: 'ticket',
        variables: [
          { name: 'ticket.identifier', description: 'Ticket identifier' },
          { name: 'ticket.title', description: 'Ticket title' },
        ],
      },
      {
        name: 'project',
        variables: [{ name: 'project.name', description: 'Project name' }],
      },
    ],
  }
}

export function createSeedCounters() {
  return {
    machine: 2,
    repo: 1,
    workflow: 1,
    agent: 1,
    skill: 2,
    scheduledJob: 1,
    projectUpdateThread: 0,
    projectUpdateComment: 0,
    projectConversation: 0,
    projectConversationEntry: 0,
    projectConversationTurn: 0,
  }
}
