import type { TicketDetail } from '../types'
import type { TicketPickupDiagnosis } from '../pickup-diagnosis'
import {
  buildCapacityLine,
  buildLegacyRepeatedStallDetail,
  diagnosisLabel,
  diagnosisTone,
  formatRetryCountdown,
  humanizeControlState,
  humanizePauseReason,
  humanizeProviderState,
} from './ticket-runtime-state-card-format'

export type RuntimeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'
export type RuntimeSummary = {
  label: string
  tone: RuntimeTone
  message: string
  timestamp?: string
  nextActionHint?: string
  countdownLine?: string
  detailItems: Array<{ label: string; value: string }>
}

export function summarizeRuntime(ticket: TicketDetail, now: number): RuntimeSummary {
  if (ticket.pickupDiagnosis) {
    return summarizeDiagnosis(ticket.pickupDiagnosis, ticket, now)
  }

  return summarizeLegacy(ticket)
}

export function runtimeDotClass(tone: RuntimeTone) {
  switch (tone) {
    case 'info':
      return 'bg-sky-400'
    case 'success':
      return 'bg-emerald-400'
    case 'warning':
      return 'bg-amber-400'
    case 'danger':
      return 'bg-red-400'
    default:
      return 'bg-muted-foreground/50'
  }
}

export function runtimeLabelClass(tone: RuntimeTone) {
  switch (tone) {
    case 'info':
      return 'text-sky-400'
    case 'success':
      return 'text-emerald-400'
    case 'warning':
      return 'text-amber-400'
    case 'danger':
      return 'text-red-400'
    default:
      return 'text-muted-foreground'
  }
}

function summarizeDiagnosis(
  diagnosis: TicketPickupDiagnosis,
  ticket: TicketDetail,
  now: number,
): RuntimeSummary {
  const detailItems: RuntimeSummary['detailItems'] = []

  if (diagnosis.workflow) {
    detailItems.push({
      label: 'Workflow',
      value: `${diagnosis.workflow.name} · ${diagnosis.workflow.isActive ? 'Active' : 'Inactive'}`,
    })
  }

  if (diagnosis.agent) {
    detailItems.push({
      label: 'Agent',
      value: `${diagnosis.agent.name} · ${humanizeControlState(diagnosis.agent.runtimeControlState)}`,
    })
  } else if (diagnosis.primaryReasonCode === 'workflow_missing_agent') {
    detailItems.push({ label: 'Agent', value: 'No agent is bound to the workflow.' })
  }

  if (diagnosis.provider) {
    detailItems.push({
      label: 'Provider',
      value: `${diagnosis.provider.name} · ${humanizeProviderState(diagnosis.provider)}`,
    })
  }

  if (diagnosis.blockedBy.length > 0) {
    detailItems.push({
      label: 'Dependencies',
      value: diagnosis.blockedBy.map((item) => `${item.identifier} ${item.title}`).join(', '),
    })
  }

  const capacityLine = buildCapacityLine(diagnosis)
  if (capacityLine) {
    detailItems.push({ label: 'Capacity', value: capacityLine })
  }

  const countdownLine = diagnosis.retry.nextRetryAt
    ? formatRetryCountdown(diagnosis.retry.nextRetryAt, now)
    : undefined
  if (diagnosis.retry.retryPaused && diagnosis.retry.pauseReason) {
    detailItems.push({
      label: 'Retry',
      value: humanizePauseReason(diagnosis.retry.pauseReason),
    })
  }

  return {
    label: diagnosisLabel(diagnosis.state),
    tone: diagnosisTone(diagnosis.state, diagnosis.primaryReasonCode),
    message: diagnosis.primaryReasonMessage,
    timestamp:
      diagnosis.state === 'running'
        ? ticket.startedAt
        : diagnosis.state === 'completed'
          ? ticket.completedAt
          : undefined,
    nextActionHint: diagnosis.nextActionHint,
    countdownLine,
    detailItems,
  }
}

function summarizeLegacy(ticket: TicketDetail): RuntimeSummary {
  if (ticket.completedAt) {
    return {
      label: 'Completed',
      tone: 'success',
      message: 'Execution finished successfully.',
      timestamp: ticket.completedAt,
      detailItems: [],
    }
  }

  if (ticket.retryPaused && ticket.pauseReason === 'repeated_stalls') {
    return {
      label: 'Blocked',
      tone: 'warning',
      message: 'Paused after repeated stalls — manual retry required.',
      detailItems: buildLegacyRepeatedStallDetail(),
    }
  }

  if (ticket.retryPaused) {
    return {
      label: 'Waiting',
      tone: 'warning',
      message: 'Waiting for retry conditions to change.',
      detailItems: [],
    }
  }

  if (ticket.assignedAgent?.runtimeControlState === 'paused') {
    return {
      label: 'Unavailable',
      tone: 'warning',
      message: 'Assigned agent is paused.',
      detailItems: [{ label: 'Agent', value: `${ticket.assignedAgent.name} · Paused` }],
    }
  }

  switch (ticket.assignedAgent?.runtimePhase) {
    case 'failed':
      return {
        label: 'Failed',
        tone: 'danger',
        message: 'Latest attempt failed — needs attention.',
        detailItems: [],
      }
    case 'launching':
      return {
        label: 'Launching',
        tone: 'info',
        message: 'Agent is spinning up the runtime.',
        detailItems: [],
      }
    case 'ready':
    case 'executing':
      return {
        label: 'Running',
        tone: 'success',
        message: 'Agent runtime is live.',
        timestamp: ticket.startedAt,
        detailItems: [],
      }
    default:
      return {
        label: ticket.assignedAgent ? 'Assigned' : 'Waiting',
        tone: 'neutral',
        message: ticket.assignedAgent
          ? 'Agent bound, no active runtime.'
          : 'No agent runtime attached yet.',
        detailItems: [],
      }
  }
}
