import type { TicketDetailPayload } from '$lib/api/contracts'

export const detailPickupDiagnosisFixture: NonNullable<TicketDetailPayload['pickup_diagnosis']> = {
  state: 'running',
  primary_reason_code: 'running_current_run',
  primary_reason_message: 'Ticket already has an active run.',
  next_action_hint: 'Wait for the current run to finish or inspect the active runtime.',
  reasons: [
    {
      code: 'running_current_run',
      message: 'Current run is still attached to the ticket.',
      severity: 'info',
    },
  ],
  workflow: {
    id: 'workflow-1',
    name: 'Todo App Coding Workflow',
    is_active: true,
    pickup_status_match: true,
  },
  agent: {
    id: 'agent-1',
    name: 'todo-app-coding-01',
    runtime_control_state: 'active',
  },
  provider: {
    id: 'provider-1',
    name: 'codex-cloud',
    machine_id: 'machine-1',
    machine_name: 'builder-01',
    machine_status: 'online',
    availability_state: 'available',
    availability_reason: null,
  },
  retry: {
    attempt_count: 0,
    retry_paused: false,
    pause_reason: '',
    next_retry_at: null,
  },
  capacity: {
    workflow: { limited: true, active_runs: 1, capacity: 2 },
    project: { limited: true, active_runs: 1, capacity: 5 },
    provider: { limited: true, active_runs: 1, capacity: 3 },
    status: { limited: false, active_runs: 1, capacity: null },
  },
  blocked_by: [],
}
