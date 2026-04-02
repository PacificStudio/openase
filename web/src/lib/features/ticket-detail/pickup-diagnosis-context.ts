import type { TicketDetailPayload } from '$lib/api/contracts'
import type { TicketPickupDiagnosis } from './pickup-diagnosis'

type PickupDiagnosisResponse = TicketDetailPayload['pickup_diagnosis']

export function mapTicketPickupDiagnosis(
  diagnosis: PickupDiagnosisResponse,
): TicketPickupDiagnosis | undefined {
  if (!diagnosis) {
    return undefined
  }

  return {
    state: parseDiagnosisState(diagnosis.state),
    primaryReasonCode: diagnosis.primary_reason_code,
    primaryReasonMessage: diagnosis.primary_reason_message,
    nextActionHint: diagnosis.next_action_hint || undefined,
    reasons: diagnosis.reasons.map((reason) => ({
      code: reason.code,
      message: reason.message,
      severity: reason.severity as 'info' | 'warning' | 'error',
    })),
    workflow: diagnosis.workflow
      ? {
          id: diagnosis.workflow.id,
          name: diagnosis.workflow.name,
          isActive: diagnosis.workflow.is_active,
          pickupStatusMatch: diagnosis.workflow.pickup_status_match,
        }
      : undefined,
    agent: diagnosis.agent
      ? {
          id: diagnosis.agent.id,
          name: diagnosis.agent.name,
          runtimeControlState: diagnosis.agent.runtime_control_state,
        }
      : undefined,
    provider: diagnosis.provider
      ? {
          id: diagnosis.provider.id,
          name: diagnosis.provider.name,
          machineId: diagnosis.provider.machine_id,
          machineName: diagnosis.provider.machine_name,
          machineStatus: diagnosis.provider.machine_status,
          availabilityState: diagnosis.provider.availability_state,
          availabilityReason: diagnosis.provider.availability_reason || undefined,
        }
      : undefined,
    retry: {
      attemptCount: diagnosis.retry.attempt_count,
      retryPaused: diagnosis.retry.retry_paused,
      pauseReason: diagnosis.retry.pause_reason || undefined,
      nextRetryAt: diagnosis.retry.next_retry_at || undefined,
    },
    capacity: {
      workflow: {
        limited: diagnosis.capacity.workflow.limited,
        activeRuns: diagnosis.capacity.workflow.active_runs,
        capacity: diagnosis.capacity.workflow.capacity,
      },
      project: {
        limited: diagnosis.capacity.project.limited,
        activeRuns: diagnosis.capacity.project.active_runs,
        capacity: diagnosis.capacity.project.capacity,
      },
      provider: {
        limited: diagnosis.capacity.provider.limited,
        activeRuns: diagnosis.capacity.provider.active_runs,
        capacity: diagnosis.capacity.provider.capacity,
      },
      status: {
        limited: diagnosis.capacity.status.limited,
        activeRuns: diagnosis.capacity.status.active_runs,
        capacity: diagnosis.capacity.status.capacity ?? undefined,
      },
    },
    blockedBy: diagnosis.blocked_by.map((blocker) => ({
      id: blocker.id,
      identifier: blocker.identifier,
      title: blocker.title,
      statusId: blocker.status_id,
      statusName: blocker.status_name,
    })),
  }
}

function parseDiagnosisState(value: string): TicketPickupDiagnosis['state'] {
  switch (value) {
    case 'runnable':
    case 'waiting':
    case 'blocked':
    case 'running':
    case 'completed':
    case 'unavailable':
      return value
    default:
      return 'unavailable'
  }
}
