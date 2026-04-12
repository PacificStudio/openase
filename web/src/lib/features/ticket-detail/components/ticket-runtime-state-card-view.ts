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
import { i18nStore } from '$lib/i18n/store.svelte'

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
      label: i18nStore.t('ticketDetail.runtime.detail.workflow'),
      value: `${diagnosis.workflow.name} · ${i18nStore.t(
        diagnosis.workflow.isActive
          ? 'ticketDetail.runtime.workflowState.active'
          : 'ticketDetail.runtime.workflowState.inactive',
      )}`,
    })
  }

  if (diagnosis.agent) {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.agent'),
      value: `${diagnosis.agent.name} · ${humanizeControlState(
        diagnosis.agent.runtimeControlState,
      )}`,
    })
  } else if (diagnosis.primaryReasonCode === 'workflow_missing_agent') {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.agent'),
      value: i18nStore.t('ticketDetail.runtime.detail.agent.unbound'),
    })
  }

  if (diagnosis.provider) {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.provider'),
      value: `${diagnosis.provider.name} · ${humanizeProviderState(diagnosis.provider)}`,
    })
  }

  if (diagnosis.blockedBy.length > 0) {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.dependencies'),
      value: diagnosis.blockedBy.map((item) => `${item.identifier} ${item.title}`).join(', '),
    })
  }

  const capacityLine = buildCapacityLine(diagnosis)
  if (capacityLine) {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.capacity'),
      value: capacityLine,
    })
  }

  const countdownLine = diagnosis.retry.nextRetryAt
    ? formatRetryCountdown(diagnosis.retry.nextRetryAt, now)
    : undefined
  if (diagnosis.retry.retryPaused && diagnosis.retry.pauseReason) {
    detailItems.push({
      label: i18nStore.t('ticketDetail.runtime.detail.retry'),
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
      label: i18nStore.t('ticketDetail.runtime.summary.completed.label'),
      tone: 'success',
      message: i18nStore.t('ticketDetail.runtime.summary.completed.message'),
      timestamp: ticket.completedAt,
      detailItems: [],
    }
  }

  if (ticket.retryPaused && ticket.pauseReason === 'repeated_stalls') {
    return {
      label: i18nStore.t('ticketDetail.runtime.summary.blocked.label'),
      tone: 'warning',
      message: i18nStore.t('ticketDetail.runtime.summary.blocked.message'),
      detailItems: buildLegacyRepeatedStallDetail(),
    }
  }

  if (ticket.retryPaused) {
    return {
      label: i18nStore.t('ticketDetail.runtime.summary.waiting.label'),
      tone: 'warning',
      message: i18nStore.t('ticketDetail.runtime.summary.waiting.retry.message'),
      detailItems: [],
    }
  }

  if (ticket.assignedAgent?.runtimeControlState === 'paused') {
    return {
      label: i18nStore.t('ticketDetail.runtime.summary.unavailable.label'),
      tone: 'warning',
      message: i18nStore.t('ticketDetail.runtime.summary.unavailable.message'),
      detailItems: [
        {
          label: i18nStore.t('ticketDetail.runtime.detail.agent'),
          value: `${ticket.assignedAgent.name} · ${i18nStore.t(
            'ticketDetail.runtime.agentState.paused',
          )}`,
        },
      ],
    }
  }

  switch (ticket.assignedAgent?.runtimePhase) {
    case 'failed':
      return {
        label: i18nStore.t('ticketDetail.runtime.summary.failed.label'),
        tone: 'danger',
        message: i18nStore.t('ticketDetail.runtime.summary.failed.message'),
        detailItems: [],
      }
    case 'launching':
      return {
        label: i18nStore.t('ticketDetail.runtime.summary.launching.label'),
        tone: 'info',
        message: i18nStore.t('ticketDetail.runtime.summary.launching.message'),
        detailItems: [],
      }
    case 'ready':
    case 'executing':
      return {
        label: i18nStore.t('ticketDetail.runtime.summary.running.label'),
        tone: 'success',
        message: i18nStore.t('ticketDetail.runtime.summary.running.message'),
        timestamp: ticket.startedAt,
        detailItems: [],
      }
    default:
      if (ticket.assignedAgent) {
        return {
          label: i18nStore.t('ticketDetail.runtime.summary.assigned.label'),
          tone: 'neutral',
          message: i18nStore.t('ticketDetail.runtime.summary.assigned.message'),
          detailItems: [],
        }
      }
      return {
        label: i18nStore.t('ticketDetail.runtime.summary.waiting.label'),
        tone: 'neutral',
        message: i18nStore.t('ticketDetail.runtime.summary.waiting.noAgentMessage'),
        detailItems: [],
      }
  }
}
