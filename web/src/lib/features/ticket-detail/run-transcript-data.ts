import type {
  TicketRunDetailPayload,
  TicketRunListPayload,
  TicketRunRecord,
  TicketRunStepRecord,
  TicketRunTraceRecord,
} from '$lib/api/contracts'
import type {
  TicketRun,
  TicketRunCompletionSummary,
  TicketRunDetail,
  TicketRunStepEntry,
  TicketRunTraceEntry,
} from './types'

export function mapTicketRuns(payload: TicketRunListPayload): TicketRun[] {
  return payload.runs.map(mapTicketRun)
}

export function mapTicketRunDetail(payload: TicketRunDetailPayload): TicketRunDetail {
  return {
    run: mapTicketRun(payload.run),
    traceEntries: payload.trace_entries.map(mapTicketRunTraceEntry),
    stepEntries: payload.step_entries.map(mapTicketRunStepEntry),
  }
}

type TicketRunLifecycleEvent = {
  eventType: string
  message: string
  createdAt: string
}

type TicketRunLifecycleEventRecord = {
  event_type?: string
  eventType?: string
  message?: string
  created_at?: string
  createdAt?: string
}

export function mapTicketRun(item: TicketRunRecord): TicketRun {
  return {
    id: item.id,
    attemptNumber: item.attempt_number,
    agentId: item.agent_id,
    agentName: item.agent_name,
    provider: item.provider,
    status: normalizeRunStatus(item.status),
    currentStepStatus: item.current_step_status ?? undefined,
    currentStepSummary: item.current_step_summary ?? undefined,
    createdAt: item.created_at,
    runtimeStartedAt: item.runtime_started_at ?? undefined,
    lastHeartbeatAt: item.last_heartbeat_at ?? undefined,
    terminalAt: item.terminal_at ?? undefined,
    completedAt: item.completed_at ?? undefined,
    lastError: item.last_error ?? undefined,
    completionSummary: mapTicketRunCompletionSummary(item.completion_summary),
  }
}

type TicketRunCompletionSummaryRecord =
  | {
      status?: string
      markdown?: string | null
      json?: Record<string, unknown> | null
      generated_at?: string | null
      error?: string | null
    }
  | null
  | undefined

export function mapTicketRunCompletionSummary(
  item: TicketRunCompletionSummaryRecord,
): TicketRunCompletionSummary | undefined {
  if (!item?.status) {
    return undefined
  }
  if (item.status !== 'pending' && item.status !== 'completed' && item.status !== 'failed') {
    return undefined
  }
  return {
    status: item.status,
    markdown: item.markdown ?? undefined,
    json: item.json ?? undefined,
    generatedAt: item.generated_at ?? undefined,
    error: item.error ?? undefined,
  }
}

export function mapTicketRunStreamLifecycleEvent(
  item: TicketRunLifecycleEventRecord,
): TicketRunLifecycleEvent {
  return {
    eventType: item.event_type ?? item.eventType ?? '',
    message: item.message ?? '',
    createdAt: item.created_at ?? item.createdAt ?? '',
  }
}

export function mapTicketRunTraceEntry(item: TicketRunTraceRecord): TicketRunTraceEntry {
  return {
    id: item.id,
    agentRunId: item.agent_run_id,
    sequence: item.sequence,
    provider: item.provider,
    kind: item.kind,
    stream: item.stream,
    output: item.output,
    payload: item.payload ?? {},
    createdAt: item.created_at,
  }
}

export function mapTicketRunStepEntry(item: TicketRunStepRecord): TicketRunStepEntry {
  return {
    id: item.id,
    agentRunId: item.agent_run_id,
    stepStatus: item.step_status,
    summary: item.summary,
    sourceTraceEventId: item.source_trace_event_id ?? undefined,
    createdAt: item.created_at,
  }
}

function normalizeRunStatus(status: string): TicketRun['status'] {
  if (
    status === 'launching' ||
    status === 'ready' ||
    status === 'executing' ||
    status === 'ended' ||
    status === 'failed' ||
    status === 'completed'
  ) {
    return status
  }

  if (status === 'stalled') {
    return 'ended'
  }

  return 'launching'
}
