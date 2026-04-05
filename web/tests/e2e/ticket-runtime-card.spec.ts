import type { Page } from '@playwright/test'
import { expect, test } from './fixtures'

const detailRoute = '**/api/v1/projects/project-e2e/tickets/ticket-1/detail'

test('ticket runtime card shows retry backoff countdown with UTC timestamp and updates live', async ({
  page,
  projectPath,
}) => {
  const nextRetryAt = new Date(Date.now() + 90_000).toISOString()
  await stubTicketDetail(page, {
    assigned_agent: null,
    pickup_diagnosis: {
      state: 'waiting',
      primary_reason_code: 'retry_backoff',
      primary_reason_message: 'Waiting for retry backoff to expire.',
      next_action_hint:
        'The ticket will become schedulable automatically after the retry window expires.',
      reasons: [
        {
          code: 'retry_backoff',
          message: `Next retry is scheduled for ${nextRetryAt}.`,
          severity: 'info',
        },
      ],
      workflow: {
        id: 'workflow-coding',
        name: 'Coding Workflow',
        is_active: true,
        pickup_status_match: true,
      },
      agent: {
        id: 'agent-coder',
        name: 'coding-main',
        runtime_control_state: 'active',
      },
      provider: {
        id: 'provider-1',
        name: 'codex-app-server',
        machine_id: 'machine-1',
        machine_name: 'local-dev',
        machine_status: 'online',
        availability_state: 'available',
        availability_reason: null,
      },
      retry: {
        attempt_count: 2,
        retry_paused: false,
        pause_reason: '',
        next_retry_at: nextRetryAt,
      },
      capacity: baseCapacity(),
      blocked_by: [],
    },
  })

  await openDefaultTicket(page, projectPath)

  const absolute = formatUTC(nextRetryAt)
  const countdownLine = page.getByText(
    new RegExp(`Retrying in .* \\(at ${escapeRegExp(absolute)}\\)`),
  )
  await expect(countdownLine).toBeVisible()
  const initialCountdown = await countdownLine.textContent()

  await page.waitForTimeout(2000)

  await expect(countdownLine).toBeVisible()
  const updatedCountdown = await countdownLine.textContent()
  expect(updatedCountdown).not.toEqual(initialCountdown)
})

test('ticket runtime card names blocking tickets from pickup diagnosis', async ({
  page,
  projectPath,
}) => {
  await stubTicketDetail(page, {
    assigned_agent: null,
    pickup_diagnosis: {
      state: 'blocked',
      primary_reason_code: 'blocked_dependency',
      primary_reason_message: 'Waiting for blocking tickets to finish.',
      next_action_hint: 'Resolve the blocking tickets or move them to a terminal status.',
      reasons: [
        {
          code: 'blocked_dependency',
          message: 'Blocked by ASE-77 Stabilize project conversation restore.',
          severity: 'warning',
        },
      ],
      workflow: {
        id: 'workflow-coding',
        name: 'Coding Workflow',
        is_active: true,
        pickup_status_match: true,
      },
      agent: {
        id: 'agent-coder',
        name: 'coding-main',
        runtime_control_state: 'active',
      },
      provider: {
        id: 'provider-1',
        name: 'codex-app-server',
        machine_id: 'machine-1',
        machine_name: 'local-dev',
        machine_status: 'online',
        availability_state: 'available',
        availability_reason: null,
      },
      retry: {
        attempt_count: 1,
        retry_paused: false,
        pause_reason: '',
        next_retry_at: null,
      },
      capacity: baseCapacity(),
      blocked_by: [
        {
          id: 'ticket-blocker',
          identifier: 'ASE-77',
          title: 'Stabilize project conversation restore',
          status_id: 'status-review',
          status_name: 'In Review',
        },
      ],
    },
    ticket: {
      dependencies: [
        {
          id: 'dependency-1',
          type: 'blocked_by',
          target: {
            id: 'ticket-blocker',
            identifier: 'ASE-77',
            title: 'Stabilize project conversation restore',
            status_id: 'status-review',
            status_name: 'In Review',
          },
        },
      ],
    },
  })

  await openDefaultTicket(page, projectPath)

  await expect(page.getByText('Waiting for blocking tickets to finish.')).toBeVisible()
  await expect(page.getByText('ASE-77 Stabilize project conversation restore')).toBeVisible()
})

test('ticket runtime card explains when no active workflow picks up the current status', async ({
  page,
  projectPath,
}) => {
  await stubTicketDetail(page, {
    assigned_agent: null,
    pickup_diagnosis: {
      state: 'unavailable',
      primary_reason_code: 'no_matching_active_workflow',
      primary_reason_message: "No workflow picks up the ticket's current status.",
      next_action_hint:
        'Add an active workflow for this status or move the ticket into a pickup status.',
      reasons: [
        {
          code: 'no_matching_active_workflow',
          message: 'No workflow in this project picks up status Todo.',
          severity: 'error',
        },
      ],
      workflow: null,
      agent: null,
      provider: null,
      retry: {
        attempt_count: 0,
        retry_paused: false,
        pause_reason: '',
        next_retry_at: null,
      },
      capacity: baseCapacity(),
      blocked_by: [],
    },
  })

  await openDefaultTicket(page, projectPath)

  await expect(page.getByText("No workflow picks up the ticket's current status.")).toBeVisible()
  await expect(
    page.getByText(
      'Add an active workflow for this status or move the ticket into a pickup status.',
    ),
  ).toBeVisible()
})

test('ticket runtime card surfaces provider unavailability instead of a generic waiting state', async ({
  page,
  projectPath,
}) => {
  await stubTicketDetail(page, {
    assigned_agent: null,
    pickup_diagnosis: {
      state: 'unavailable',
      primary_reason_code: 'provider_unavailable',
      primary_reason_message: 'Provider is unavailable.',
      next_action_hint: 'Fix the provider health issue before expecting pickup.',
      reasons: [
        {
          code: 'provider_unavailable',
          message: 'Provider CLI is not ready.',
          severity: 'error',
        },
      ],
      workflow: {
        id: 'workflow-coding',
        name: 'Coding Workflow',
        is_active: true,
        pickup_status_match: true,
      },
      agent: {
        id: 'agent-coder',
        name: 'coding-main',
        runtime_control_state: 'active',
      },
      provider: {
        id: 'provider-1',
        name: 'codex-app-server',
        machine_id: 'machine-1',
        machine_name: 'local-dev',
        machine_status: 'online',
        availability_state: 'unavailable',
        availability_reason: 'not_ready',
      },
      retry: {
        attempt_count: 2,
        retry_paused: false,
        pause_reason: '',
        next_retry_at: null,
      },
      capacity: baseCapacity(),
      blocked_by: [],
    },
  })

  await openDefaultTicket(page, projectPath)

  await expect(page.getByText('Provider is unavailable.')).toBeVisible()
  await expect(page.getByText('Provider', { exact: true })).toBeVisible()
  await expect(page.getByText('codex-app-server · Unavailable (CLI not ready)')).toBeVisible()
})

async function openDefaultTicket(page: Page, projectPath: (section: string) => string) {
  const ticketLink = page.getByText('ASE-101').first()

  await page.goto(projectPath('tickets'))
  if (
    !(await ticketLink.isVisible({ timeout: 2_000 }).catch(() => false)) &&
    (await page
      .getByRole('heading', { name: '500' })
      .isVisible({ timeout: 1_000 })
      .catch(() => false))
  ) {
    await page.goto(projectPath('tickets'))
  }

  await expect(ticketLink).toBeVisible({ timeout: 10_000 })
  await ticketLink.click()
  await expect(page.getByRole('dialog', { name: /ASE-101/i })).toBeVisible({ timeout: 10_000 })
}

async function stubTicketDetail(page: Page, overrides: Record<string, unknown>) {
  await page.route(detailRoute, async (route) => {
    const payload = buildTicketDetailPayload(overrides)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(payload),
    })
  })
}

function buildTicketDetailPayload(overrides: Record<string, unknown>) {
  const ticketOverrides = (overrides.ticket ?? {}) as Record<string, unknown>
  return {
    assigned_agent: overrides.assigned_agent ?? {
      id: 'agent-coder',
      name: 'coding-main',
      provider: 'codex-app-server',
      runtime_control_state: 'active',
      runtime_phase: 'executing',
    },
    pickup_diagnosis: overrides.pickup_diagnosis,
    ticket: {
      id: 'ticket-1',
      project_id: 'project-e2e',
      identifier: 'ASE-101',
      title: 'Improve machine management UX',
      description: 'Playwright stubbed ticket detail payload.',
      status_id: 'status-todo',
      status_name: 'Todo',
      priority: 'high',
      type: 'feature',
      workflow_id: 'workflow-coding',
      current_run_id: null,
      target_machine_id: null,
      created_by: 'playwright',
      parent: null,
      children: [],
      dependencies: [],
      external_links: [],
      external_ref: '',
      budget_usd: 10,
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
      created_at: new Date().toISOString(),
      ...ticketOverrides,
    },
    repo_scopes: [],
    comments: [],
    timeline: [
      {
        id: 'description:ticket-1',
        ticket_id: 'ticket-1',
        item_type: 'description',
        actor_name: 'playwright',
        actor_type: 'user',
        title: 'Improve machine management UX',
        body_markdown: 'Playwright stubbed ticket detail payload.',
        body_text: null,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        edited_at: null,
        is_collapsible: false,
        is_deleted: false,
        metadata: { identifier: 'ASE-101' },
      },
    ],
    activity: [],
    hook_history: [],
  }
}

function baseCapacity() {
  return {
    workflow: { limited: false, active_runs: 0, capacity: 0 },
    project: { limited: false, active_runs: 0, capacity: 0 },
    provider: { limited: false, active_runs: 0, capacity: 0 },
    status: { limited: false, active_runs: 0, capacity: null },
  }
}

function formatUTC(value: string) {
  const date = new Date(value)
  const year = date.getUTCFullYear()
  const month = String(date.getUTCMonth() + 1).padStart(2, '0')
  const day = String(date.getUTCDate()).padStart(2, '0')
  const hours = String(date.getUTCHours()).padStart(2, '0')
  const minutes = String(date.getUTCMinutes()).padStart(2, '0')
  const seconds = String(date.getUTCSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} UTC`
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
